# Ingestion & Export Standards

To ensure a seamless user experience, Wisdom delegates the User Interface for Knowledge Management to **Obsidian** and **Logseq**, and Spaced Repetition to **Anki**. This requires strict adherence to ingestion and formatting standards.

## 1. Obsidian Markdown Guarantees
When an Expert Agent generates a note, it MUST comply with the following standards so it integrates natively into the user's local Obsidian Vault via the `obsidian-mcp-server`:

*   **YAML Frontmatter:** Every file must start with valid YAML containing structural metadata.
    ```yaml
    ---
    id: 20260514120000
    title: "The Caro-Kann Defense"
    aliases: ["Caro-Kann", "B10"]
    tags: ["#chess/openings", "#black"]
    mastery_score: 0.45
    tipo: permanente       # permanente | literatura | efimera
    estado: borrador       # borrador | en-progreso | terminado
    fase: "[[Chess MOC]]" # Parent MOC (3-link rule enforcement)
    date: 2026-05-14
    ---
    ```
*   **Wikilinks:** Relationships between concepts must use strict Obsidian wikilinks: `[[Concept Name]]` or `[[Concept Name|Display Text]]`.
*   **Atomic Notes:** Information should be broken down into atomic, self-contained concepts to maximize graph connectivity.
*   **Directory Structure:** Files must be saved into a logical folder hierarchy (e.g., `Vault/Chess/Openings/Caro-Kann.md`).

## 2. Logseq Compatibility (Block-Level Graphs)
For users utilizing Logseq, Wisdom must support outliner-style logic:
*   **Block References:** When inserting content via `mcp-logseq`, use atomic blocks (bullet points).
*   **Hierarchy:** Maintain parent-child block relationships for properties and definitions.

## 3. Anki Export Schemas
Wisdom pushes flashcards to Anki via `anki-mcp-server` using strict, predefined Note Types to ensure the UI in Anki is clean and functional.

### Schema: `Wisdom-Basic`
*   **Front:** The question or prompt (Supports Markdown/HTML).
*   **Back:** The answer, explanation, and a `[[Wikilink]]` back to the Obsidian source note.
*   **Tags:** Hierarchical tags (e.g., `Wisdom::Chess::Tactics`).

### Schema: `Wisdom-Cloze` (For Languages/Code)
*   **Text:** Sentence with cloze deletions (e.g., `The capital of France is {{c1::Paris}}.`)
*   **Extra:** Contextual rules or grammar explanations.

## The Generation Pipeline
1. `Thalamus` receives a user request.
2. `Expert Agent` formulates the knowledge.
3. `Expert Agent` calls the MCP Server (`obsidian-mcp` or `anki-mcp`) passing the strictly formatted JSON/Markdown payloads.
4. The user sees the files magically appear perfectly formatted in their native applications.

---

## 4. Markdown → Anki Parsing Conventions

This section defines how existing Obsidian notes are parsed into Anki cards by the `md_to_anki` pipeline. See `MD_TO_ANKI_PIPELINE.md` for the full spec.

| Markdown Pattern | Anki Note Type | Rule |
|---|---|---|
| `##` or `###` heading + following paragraph | `Wisdom-Basic` | Heading = Front; paragraph(s) until next heading = Back |
| Paragraph with block ID (`^id`) + `**bolded**` phrase | `Wisdom-Cloze` | Bolded phrase becomes `{{c1::...}}` |
| `**Term** — Definition` pattern | `Wisdom-Basic` | "What is Term?" = Front; Definition = Back |
| Table rows (non-header) | `Wisdom-Basic` | First column = Front; remaining columns = Back |
| Checkboxes (`- [ ]`, `- [x]`) | Skipped | Tasks are not knowledge |
| `mastery_score` in YAML | Seeds `MasteryScore` in Cortex | Applied to all cards generated from this note |

**Rejection criteria:** Notes without an `id` YAML field are rejected. Notes without `##` headings or block references produce a warning: *"No convertible content found."*

---

## 5. Polish-with-Gemini Rewrite Contract

When the user triggers the "Polish" action on a note, the `Integrations` service calls Gemini with the following audit instructions. The response must be a structured diff (list of hunks), not a full rewrite.

### Audit Checklist for Gemini
1. **Prose redundancy:** Identify duplicate ideas across paragraphs. Collapse into one.
2. **Structure opportunities:** Paragraphs of >4 sentences on the same topic → suggest bullet list or table.
3. **Heading hierarchy:** Flag missing `#` title, incorrect heading order, or skipped levels.
4. **Wikilink gaps:** Suggest `[[links]]` for terms that match existing vault note titles.
5. **Atomic size:** If note body >500 words, suggest candidate split points and new note titles.
6. **YAML completeness:** Suggest values for empty required fields (`tipo`, `estado`, `fase`).

### Response Schema
```json
{
  "hunks": [
    {
      "original": "The original paragraph text...",
      "rewritten": "The improved version...",
      "reason": "Converted prose to bullet list to improve scannability"
    }
  ]
}
```

**Contract rule:** Gemini MUST NOT rewrite entire sections wholesale. Each hunk must map to a specific, locatable block in the original note. The user accepts or rejects each hunk independently.
