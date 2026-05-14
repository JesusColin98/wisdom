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
