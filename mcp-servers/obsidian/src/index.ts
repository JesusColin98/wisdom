import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import axios, { AxiosInstance } from "axios";
import { z } from "zod";

// ─── Config ───────────────────────────────────────────────────────────────────
// Obsidian Local REST API plugin must be installed and running.
// Generate an API key in: Obsidian → Settings → Local REST API → API Keys.
const OBSIDIAN_API_URL = process.env.OBSIDIAN_API_URL ?? "https://localhost:27123";
const OBSIDIAN_API_KEY = process.env.OBSIDIAN_API_KEY ?? "";
const WISDOM_VAULT_ROOT = process.env.WISDOM_VAULT_ROOT ?? ""; // e.g., "Wisdom/"

if (!OBSIDIAN_API_KEY) {
  console.error("ERROR: OBSIDIAN_API_KEY environment variable is required.");
  process.exit(1);
}

// ─── Obsidian REST Client ─────────────────────────────────────────────────────
function createObsidianClient(): AxiosInstance {
  return axios.create({
    baseURL: OBSIDIAN_API_URL,
    headers: {
      Authorization: `Bearer ${OBSIDIAN_API_KEY}`,
      "Content-Type": "application/json",
    },
    // Obsidian Local REST API uses a self-signed cert by default.
    httpsAgent: new (require("https").Agent)({ rejectUnauthorized: false }),
    timeout: 10_000,
  });
}

const obsidian = createObsidianClient();

// ─── MCP Server ───────────────────────────────────────────────────────────────
const server = new McpServer({
  name: "wisdom-obsidian",
  version: "1.0.0",
});

// ── Tool: search_vault ────────────────────────────────────────────────────────
server.tool(
  "search_vault",
  "Search the Obsidian vault using a keyword or semantic query.",
  {
    query: z.string().describe("The search term or phrase to look for in the vault."),
    limit: z.number().optional().default(10).describe("Max number of results to return."),
  },
  async ({ query, limit }) => {
    try {
      const resp = await obsidian.post("/search/simple/", { query, contextLength: 200 });
      const results = (resp.data as any[]).slice(0, limit).map((r: any) => ({
        path: r.filename,
        score: r.score,
        excerpt: r.context,
      }));
      return {
        content: [{ type: "text", text: JSON.stringify(results, null, 2) }],
      };
    } catch (err: any) {
      return {
        content: [{ type: "text", text: `Search error: ${err.message}` }],
        isError: true,
      };
    }
  }
);

// ── Tool: read_note ───────────────────────────────────────────────────────────
server.tool(
  "read_note",
  "Read the content and frontmatter of a specific Obsidian note by its vault path.",
  {
    path: z.string().describe("Vault-relative path to the note, e.g. 'Chess/Openings/Caro-Kann.md'"),
  },
  async ({ path }) => {
    try {
      const resp = await obsidian.get(`/vault/${encodeURIComponent(path)}`);
      return {
        content: [{ type: "text", text: resp.data as string }],
      };
    } catch (err: any) {
      return {
        content: [{ type: "text", text: `Read error: ${err.message}` }],
        isError: true,
      };
    }
  }
);

// ── Tool: create_note ─────────────────────────────────────────────────────────
server.tool(
  "create_note",
  "Create a new Markdown note in the Obsidian vault with YAML frontmatter.",
  {
    path: z.string().describe("Vault-relative path for the new note, e.g. 'Tech/React/RSC.md'"),
    title: z.string().describe("The title of the note (also used in YAML frontmatter)."),
    tags: z.array(z.string()).optional().describe("YAML frontmatter tags, e.g. ['#chess/openings']"),
    aliases: z.array(z.string()).optional().describe("YAML frontmatter aliases."),
    mastery_score: z.number().optional().describe("Current mastery score 0.0–1.0 for this concept."),
    content: z.string().describe("The Markdown body of the note (after frontmatter)."),
    relationships: z.array(z.string()).optional().describe("Wikilinks to related concepts, e.g. ['[[React Hooks]]']"),
  },
  async ({ path, title, tags, aliases, mastery_score, content, relationships }) => {
    try {
      // Build YAML frontmatter per INGESTION_STANDARDS.md spec.
      const timestamp = Date.now().toString().slice(0, -3); // yyyymmddhhmmss
      const frontmatter = [
        "---",
        `id: ${timestamp}`,
        `title: "${title}"`,
        aliases?.length ? `aliases: [${aliases.map((a) => `"${a}"`).join(", ")}]` : "",
        tags?.length ? `tags: [${tags.join(", ")}]` : "",
        mastery_score !== undefined ? `mastery_score: ${mastery_score}` : "",
        "---",
      ]
        .filter(Boolean)
        .join("\n");

      // Append relationship wikilinks at the bottom.
      const relSection =
        relationships?.length
          ? `\n\n## Related\n${relationships.map((r) => `- ${r}`).join("\n")}`
          : "";

      const fullContent = `${frontmatter}\n\n# ${title}\n\n${content}${relSection}`;
      const targetPath = WISDOM_VAULT_ROOT ? `${WISDOM_VAULT_ROOT}${path}` : path;

      await obsidian.put(`/vault/${encodeURIComponent(targetPath)}`, fullContent, {
        headers: { "Content-Type": "text/markdown" },
      });

      return {
        content: [{ type: "text", text: `✅ Note created: ${targetPath}` }],
      };
    } catch (err: any) {
      return {
        content: [{ type: "text", text: `Create error: ${err.message}` }],
        isError: true,
      };
    }
  }
);

// ── Tool: update_note ─────────────────────────────────────────────────────────
server.tool(
  "update_note",
  "Surgically append or replace content in an existing Obsidian note.",
  {
    path: z.string().describe("Vault-relative path of the note to update."),
    content: z.string().describe("New full content to write to the note."),
  },
  async ({ path, content }) => {
    try {
      const targetPath = WISDOM_VAULT_ROOT ? `${WISDOM_VAULT_ROOT}${path}` : path;
      await obsidian.put(`/vault/${encodeURIComponent(targetPath)}`, content, {
        headers: { "Content-Type": "text/markdown" },
      });
      return {
        content: [{ type: "text", text: `✅ Note updated: ${targetPath}` }],
      };
    } catch (err: any) {
      return {
        content: [{ type: "text", text: `Update error: ${err.message}` }],
        isError: true,
      };
    }
  }
);

// ── Tool: polish_note ─────────────────────────────────────────────────────────
server.tool(
  "polish_note",
  "Refactor an existing Obsidian note to comply with Wisdom Ingestion Standards using Gemini.",
  {
    path: z.string().describe("Vault-relative path of the note to polish."),
  },
  async ({ path }) => {
    try {
      const { execSync } = require("child_process");
      const scriptPath = "c:\\Users\\jesus\\wisdom\\wisdom\\scripts\\tools\\polish_note.py";
      const vaultRoot = "c:\\Users\\jesus\\obsidian\\";
      const absPath = `${vaultRoot}${path}`;

      console.log(`Executing: python ${scriptPath} "${absPath}" --inplace`);
      
      const output = execSync(`python "${scriptPath}" "${absPath}" --inplace`, {
        env: { ...process.env, GOOGLE_CLOUD_PROJECT: "jesuscolin2025-678c7" },
        encoding: "utf-8"
      });

      return {
        content: [{ type: "text", text: `✅ Note polished: ${path}\n\n${output}` }],
      };
    } catch (err: any) {
      return {
        content: [{ type: "text", text: `Polish error: ${err.message}` }],
        isError: true,
      };
    }
  }
);

// ─── HTTP Proxy Endpoint for Integrations Service ────────────────────────────
// The Go Integrations service calls this server via HTTP POST (not stdio MCP).
// This simple Express-like handler bridges the two protocols.
import { createServer } from "http";

const PORT = parseInt(process.env.PORT ?? "3333", 10);

const httpServer = createServer(async (req, res) => {
  if (req.method !== "POST") {
    res.writeHead(405);
    res.end("Method Not Allowed");
    return;
  }

  const chunks: Buffer[] = [];
  req.on("data", (chunk) => chunks.push(chunk));
  req.on("end", async () => {
    try {
      const body = JSON.parse(Buffer.concat(chunks).toString());
      const { action, ...params } = body;

      let result: any;
      switch (action) {
        case "create_note":
          result = await createNoteHandler(params);
          break;
        case "read_note":
          result = await readNoteHandler(params);
          break;
        case "search_vault":
          result = await searchVaultHandler(params);
          break;
        case "polish_note":
          result = await polishNoteHandler(params);
          break;
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

// Handler implementations (re-use same obsidian client).
async function createNoteHandler(params: any) {
  const { path, title, tags, aliases, mastery_score, content, relationships } = params;
  const timestamp = Date.now().toString().slice(0, -3);
  const frontmatter = [
    "---",
    `id: ${timestamp}`,
    `title: "${title}"`,
    aliases?.length ? `aliases: [${aliases.map((a: string) => `"${a}"`).join(", ")}]` : "",
    tags?.length ? `tags: [${tags.join(", ")}]` : "",
    mastery_score !== undefined ? `mastery_score: ${mastery_score}` : "",
    "---",
  ].filter(Boolean).join("\n");

  const relSection = relationships?.length
    ? `\n\n## Related\n${relationships.map((r: string) => `- ${r}`).join("\n")}` : "";

  const fullContent = `${frontmatter}\n\n# ${title}\n\n${content}${relSection}`;
  const targetPath = WISDOM_VAULT_ROOT ? `${WISDOM_VAULT_ROOT}${path}` : path;

  await obsidian.put(`/vault/${encodeURIComponent(targetPath)}`, fullContent, {
    headers: { "Content-Type": "text/markdown" },
  });
  return { success: true, path: targetPath };
}

async function readNoteHandler(params: any) {
  const resp = await obsidian.get(`/vault/${encodeURIComponent(params.path)}`);
  return { content: resp.data };
}

async function searchVaultHandler(params: any) {
  const resp = await obsidian.post("/search/simple/", { query: params.query, contextLength: 200 });
  return { results: resp.data };
}

async function polishNoteHandler(params: any) {
  const { execSync } = require("child_process");
  const scriptPath = "c:\\Users\\jesus\\wisdom\\wisdom\\scripts\\tools\\polish_note.py";
  const vaultRoot = "c:\\Users\\jesus\\obsidian\\";
  const absPath = `${vaultRoot}${params.path}`;

  const output = execSync(`python "${scriptPath}" "${absPath}" --inplace`, {
    env: { ...process.env, GOOGLE_CLOUD_PROJECT: "jesuscolin2025-678c7" },
    encoding: "utf-8"
  });
  return { success: true, path: params.path, output };
}

// ─── Start ────────────────────────────────────────────────────────────────────
httpServer.listen(PORT, () => {
  console.log(`Wisdom Obsidian MCP server listening on http://localhost:${PORT}`);
});

// Also start MCP stdio transport for direct LLM tool use.
const transport = new StdioServerTransport();
server.connect(transport).catch(console.error);
