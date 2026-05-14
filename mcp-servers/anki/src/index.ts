import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import axios, { AxiosInstance } from "axios";
import { z } from "zod";
import { createServer } from "http";

// ─── Config ───────────────────────────────────────────────────────────────────
// AnkiConnect add-on must be installed and Anki desktop must be running.
// AnkiConnect listens on http://localhost:8765 by default.
const ANKICONNECT_URL = process.env.ANKICONNECT_URL ?? "http://localhost:8765";
const PORT = parseInt(process.env.PORT ?? "3334", 10);

// ─── AnkiConnect Client ───────────────────────────────────────────────────────
const ankiHttp: AxiosInstance = axios.create({
  baseURL: ANKICONNECT_URL,
  headers: { "Content-Type": "application/json" },
  timeout: 10_000,
});

/** Low-level AnkiConnect API call. */
async function ankiAction(action: string, params?: Record<string, any>): Promise<any> {
  const resp = await ankiHttp.post("/", { action, version: 6, params });
  if (resp.data.error) {
    throw new Error(`AnkiConnect error: ${resp.data.error}`);
  }
  return resp.data.result;
}

// ─── MCP Server ───────────────────────────────────────────────────────────────
const server = new McpServer({
  name: "wisdom-anki",
  version: "1.0.0",
});

// ── Tool: add_note ────────────────────────────────────────────────────────────
server.tool(
  "add_note",
  "Add a flashcard to an Anki deck using the Wisdom-Basic or Wisdom-Cloze model.",
  {
    deck_name: z.string().describe("Target deck, e.g. 'Wisdom::Chess::Tactics'"),
    model_name: z.enum(["Wisdom-Basic", "Wisdom-Cloze"]).describe("Note type to use."),
    front: z.string().optional().describe("Front of card (Wisdom-Basic only). Markdown/HTML supported."),
    back: z.string().optional().describe("Back of card with answer + Obsidian Wikilink source."),
    cloze_text: z.string().optional().describe("Cloze deletion text (Wisdom-Cloze only)."),
    extra: z.string().optional().describe("Extra context (grammar rules, notes)."),
    tags: z.array(z.string()).optional().describe("Card tags, hierarchical e.g. ['Wisdom::Chess::Tactics']"),
    wisdom_node_id: z.string().optional().describe("Cortex node ID for mastery sync back-reference."),
  },
  async ({ deck_name, model_name, front, back, cloze_text, extra, tags, wisdom_node_id }) => {
    try {
      // Ensure the target deck exists — AnkiConnect creates it if missing.
      await ankiAction("createDeck", { deck: deck_name });

      // Build fields based on model type.
      const fields: Record<string, string> =
        model_name === "Wisdom-Cloze"
          ? { Text: cloze_text ?? "", Extra: extra ?? "" }
          : {
              Front: front ?? "",
              Back: `${back ?? ""}\n\n<!-- wisdom_node_id: ${wisdom_node_id ?? ""} -->`,
            };

      // Ensure the Wisdom note model exists.
      await ensureNoteModel(model_name);

      const noteId = await ankiAction("addNote", {
        note: {
          deckName: deck_name,
          modelName: model_name,
          fields,
          tags: ["Wisdom", ...(tags ?? [])],
          options: {
            allowDuplicate: false,
            duplicateScope: "deck",
          },
        },
      });

      return {
        content: [{ type: "text", text: JSON.stringify({ success: true, note_id: noteId }) }],
      };
    } catch (err: any) {
      return {
        content: [{ type: "text", text: `add_note error: ${err.message}` }],
        isError: true,
      };
    }
  }
);

// ── Tool: find_notes ──────────────────────────────────────────────────────────
server.tool(
  "find_notes",
  "Search Anki for cards using the AnkiConnect query syntax.",
  {
    query: z.string().describe("AnkiConnect search query, e.g. 'tag:Wisdom deck:Wisdom::Chess'"),
  },
  async ({ query }) => {
    try {
      const noteIds: number[] = await ankiAction("findNotes", { query });
      const notesInfo = await ankiAction("notesInfo", { notes: noteIds.slice(0, 50) });
      return {
        content: [{ type: "text", text: JSON.stringify(notesInfo, null, 2) }],
      };
    } catch (err: any) {
      return {
        content: [{ type: "text", text: `find_notes error: ${err.message}` }],
        isError: true,
      };
    }
  }
);

// ── Tool: get_reviews ─────────────────────────────────────────────────────────
// Critical for the Wisdom Sync feedback loop (used by Integrations poller).
server.tool(
  "get_reviews",
  "Get review history for Wisdom-tagged cards since a given timestamp.",
  {
    query: z.string().default("tag:Wisdom").describe("AnkiConnect search query to filter cards."),
    since: z.number().optional().describe("Unix timestamp in ms. Only return reviews after this time."),
  },
  async ({ query, since }) => {
    try {
      const cardIds: number[] = await ankiAction("findCards", { query });
      if (cardIds.length === 0) {
        return { content: [{ type: "text", text: JSON.stringify([]) }] };
      }

      // getReviewsOfCards returns a map of cardId -> reviews[].
      const reviewsMap: Record<string, any[]> = await ankiAction("getReviewsOfCards", {
        cards: cardIds.slice(0, 200), // Limit to avoid massive payloads.
      });

      const allReviews: any[] = [];
      for (const [cardId, reviews] of Object.entries(reviewsMap)) {
        for (const r of reviews) {
          // Filter by timestamp if provided.
          if (since && r.reviewTime < since) continue;
          allReviews.push({
            cardId: parseInt(cardId, 10),
            ease: r.ease,         // 1-4 (Again/Hard/Good/Easy)
            reviewTime: r.reviewTime, // Unix ms — used as dedup review_id
            reviewDuration: r.timeTaken,
          });
        }
      }

      return {
        content: [{ type: "text", text: JSON.stringify(allReviews, null, 2) }],
      };
    } catch (err: any) {
      return {
        content: [{ type: "text", text: `get_reviews error: ${err.message}` }],
        isError: true,
      };
    }
  }
);

// ─── Ensure Note Models Exist ─────────────────────────────────────────────────
async function ensureNoteModel(modelName: "Wisdom-Basic" | "Wisdom-Cloze") {
  const models: string[] = await ankiAction("modelNames");
  if (models.includes(modelName)) return;

  if (modelName === "Wisdom-Basic") {
    await ankiAction("createModel", {
      modelName: "Wisdom-Basic",
      inOrderFields: ["Front", "Back"],
      css: `
        .card { font-family: Inter, sans-serif; font-size: 16px; text-align: center; }
        .front { font-size: 20px; font-weight: bold; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 4px; }
      `,
      cardTemplates: [
        {
          Name: "Wisdom Card",
          Front: "{{Front}}",
          Back: "{{FrontSide}}<hr id=answer>{{Back}}",
        },
      ],
    });
  } else {
    await ankiAction("createModel", {
      modelName: "Wisdom-Cloze",
      inOrderFields: ["Text", "Extra"],
      isCloze: true,
      css: `.card { font-family: Inter, sans-serif; font-size: 16px; }`,
      cardTemplates: [
        {
          Name: "Wisdom Cloze",
          Front: "{{cloze:Text}}",
          Back: "{{cloze:Text}}<br>{{Extra}}",
        },
      ],
    });
  }
}

// ─── HTTP Proxy Endpoint for Go Integrations Service ─────────────────────────
const httpServer = createServer(async (req, res) => {
  if (req.method !== "POST") {
    res.writeHead(405);
    res.end("Method Not Allowed");
    return;
  }

  const chunks: Buffer[] = [];
  req.on("data", (c) => chunks.push(c));
  req.on("end", async () => {
    try {
      const body = JSON.parse(Buffer.concat(chunks).toString());
      const { action, ...params } = body;

      let result: any;
      switch (action) {
        case "add_note": {
          await ensureNoteModel(params.model ?? "Wisdom-Basic");
          const noteId = await ankiAction("addNote", {
            note: {
              deckName: params.deck_name,
              modelName: params.model ?? "Wisdom-Basic",
              fields:
                params.model === "Wisdom-Cloze"
                  ? { Text: params.cloze_text ?? "", Extra: params.extra ?? "" }
                  : { Front: params.front ?? "", Back: params.back ?? "" },
              tags: ["Wisdom", ...(params.tags ?? [])],
              options: { allowDuplicate: false, duplicateScope: "deck" },
            },
          });
          result = { success: true, note_id: noteId };
          break;
        }
        case "get_reviews": {
          const cardIds: number[] = await ankiAction("findCards", { query: params.query ?? "tag:Wisdom" });
          const reviewsMap = cardIds.length
            ? await ankiAction("getReviewsOfCards", { cards: cardIds.slice(0, 200) })
            : {};
          const reviews: any[] = [];
          for (const [cardId, rs] of Object.entries(reviewsMap as Record<string, any[]>)) {
            for (const r of rs) {
              if (params.since && r.reviewTime < params.since) continue;
              reviews.push({ cardId: parseInt(cardId, 10), ease: r.ease, reviewTime: r.reviewTime });
            }
          }
          result = reviews;
          break;
        }
        default:
          res.writeHead(400);
          res.end(JSON.stringify({ error: `Unknown action: ${action}` }));
          return;
      }

      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify(result));
    } catch (err: any) {
      res.writeHead(500);
      res.end(JSON.stringify({ error: err.message }));
    }
  });
});

httpServer.listen(PORT, () => {
  console.log(`Wisdom Anki MCP server listening on http://localhost:${PORT}`);
  console.log(`AnkiConnect URL: ${ANKICONNECT_URL}`);
});

// Also start stdio transport for direct LLM tool use.
const transport = new StdioServerTransport();
server.connect(transport).catch(console.error);
