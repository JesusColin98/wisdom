# Markdown → Anki Export Pipeline

The `md_to_anki` pipeline allows the user to convert **any existing Obsidian Markdown note** into Anki flashcards, without requiring an AI agent to regenerate the content from scratch.

> **Complementary to the AI pipeline:** Expert Agents generate cards on the fly via the `Integrations` service. This pipeline handles the *user-authored* and *AI-generated-but-already-saved* notes that exist in the vault.

---

## 1. Triggering Mechanisms

The pipeline can be activated in three ways:

### 1.1 Portal UI (Primary)
In the **Anki Export Review Panel** (View 4 of `FRONTEND_SPEC.md`) or the **Note Editor** (View 2):
- A `[Convert to Anki cards]` button appears on each note.
- The user clicks it → the pipeline runs → generated cards appear in the Anki queue for review before being sent to AnkiConnect.

### 1.2 REST API (Automation / CLI)
```
POST /v1/integrations/md_to_anki
Body: {
  "note_path": "30_Resources/Chess/Caro-Kann.md",
  "deck": "Chess::Openings::Caro-Kann",
  "overwrite": false
}
Response: {
  "cards_generated": 5,
  "cards_updated": 1,
  "cards_skipped": 0,
  "queue_id": "uuid-xyz"
}
```

### 1.3 Obsidian MCP Tool (Programmatic)
The `obsidian-mcp-server` exposes a tool `convert_to_anki(path)` that calls the REST API above. This allows Expert Agents to trigger the pipeline on existing notes without re-generating them.

---

## 2. Markdown Parsing Rules

The parser reads the note and maps Markdown structures to Anki note types deterministically.

### 2.1 Rule 1: Heading → `Wisdom-Basic` Card
Each `##` or `###` heading (not `#` — that's the note title) generates one Basic card.

```markdown
## What is the Caro-Kann Defense?
The Caro-Kann is a chess opening starting with 1.e4 c6...
```
Maps to:
- **Front:** `What is the Caro-Kann Defense?`
- **Back:** `The Caro-Kann is a chess opening starting with 1.e4 c6...` + `[[Caro-Kann Defense]]`

**Rule:** The paragraph(s) immediately following the heading (until the next heading or `---`) form the back of the card.

### 2.2 Rule 2: Block Reference → `Wisdom-Cloze` Card
Any paragraph ending with a block ID (`^blockid`) generates a Cloze card.

```markdown
Dopamine functions as a **teaching signal** that reinforces neural pathways. ^1a2b3c
```
Maps to:
- **Text:** `Dopamine functions as a {{c1::teaching signal}} that reinforces neural pathways.`
- **Extra:** Source: `[[The Role of Dopamine in Learning]]`

**Rule:** The bolded (`**text**`) or italicized (`*text*`) phrase within the paragraph is wrapped in `{{c1::...}}`. If no emphasis is found, the AI generates a cloze from the most semantically significant phrase.

### 2.3 Rule 3: Definition Lists → `Wisdom-Basic` Cards
Markdown definition-style patterns:

```markdown
**FSRS** — Free Spaced Repetition Scheduler. A modern algorithm...
```
Maps to:
- **Front:** `What is FSRS?`
- **Back:** `Free Spaced Repetition Scheduler. A modern algorithm...`

### 2.4 Rule 4: Tables → Multiple `Wisdom-Basic` Cards
Each table row (non-header) generates one card where the first column is the front and the row content is the back.

```markdown
| Opening | ECO Code | First Move |
|---|---|---|
| Caro-Kann | B10 | 1.e4 c6 |
| Sicilian | B20 | 1.e4 c5 |
```
Generates:
- Card 1 → Front: `Caro-Kann opening` | Back: `ECO: B10 | First Move: 1.e4 c6`
- Card 2 → Front: `Sicilian opening` | Back: `ECO: B20 | First Move: 1.e4 c5`

### 2.5 Rule 5: Checkboxes → Skipped
`- [ ]` and `- [x]` items are **not** converted to cards. They represent tasks, not knowledge.

### 2.6 Rule 6: YAML `mastery_score` Seeding
If the note's YAML frontmatter contains `mastery_score: 0.45`, this value is used to pre-seed the `MasteryScore` in Cortex for the generated cards. This prevents the user from starting at zero for concepts they already partially know.

---

## 3. Idempotency Strategy

**The pipeline must never create duplicate cards.**

### 3.1 Deduplication Key
Each generated card is identified by a stable composite key:
```
card_key = SHA256(note_id + "#" + heading_text)
```
Where `note_id` is the `id` field from the note's YAML frontmatter.

**Requirement:** Notes without an `id` field in YAML are **rejected** by the pipeline with an error: *"Note is missing a unique `id` field. Add one or use the Note Editor to auto-generate it."*

### 3.2 Update vs. Create
Before writing to AnkiConnect, the pipeline queries `find_notes(query)` via the Anki MCP server:
- If a card with the same `card_key` tag already exists → **Update** the card fields in place.
- If no matching card exists → **Create** a new card.
- Update and Create counts are reported back to the UI.

### 3.3 Idempotency Tag
Every generated card receives a hidden tag: `Wisdom::id::<note_id>`. This is the lookup key for future update runs.

---

## 4. Deck Assignment Logic

The pipeline determines the target Anki deck from:
1. **Explicit:** User specifies a deck in the UI or API call.
2. **From YAML `tags`:** The first tag's path is mapped to a deck:
   - `#chess/openings` → `Chess::Openings`
   - `#tech/react` → `Tech::React`
3. **Fallback:** If no deck can be inferred → cards go to `Wisdom::Inbox` for manual sorting.

---

## 5. YAML `mastery_score` → Cortex Seeding

When the pipeline converts a note, it also calls `Cortex.Memorize()` to register the concept and its initial mastery state:

```protobuf
IngestRequest {
  content: "<note title>",
  metadata: { "anki_card_key": "<card_key>", "source_note_id": "<id>" },
  is_immutable: false,
  target_stratum: COLD
}
```

The `Trace` service uses this as the starting `MasteryScore` for the concept, ensuring Wisdom's internal SRS and Anki are synchronized from day one.

---

## 6. Error Handling

| Error | Cause | Action |
|---|---|---|
| `MISSING_ID` | Note has no `id` in YAML | Reject; prompt user to add or auto-generate |
| `ANKI_OFFLINE` | AnkiConnect not reachable | Queue cards in Cortex with `PENDING_SYNC` status |
| `INVALID_FRONTMATTER` | YAML parse error | Reject; highlight the invalid line in the editor |
| `NO_PARSEABLE_CONTENT` | Note has no headings or block refs | Warn: "No convertible content found. Add `##` headings." |
| `DUPLICATE_HEADING` | Two `##` headings with identical text | Deduplicate; warn user to rename one |

---

## 7. Additions to `INGESTION_STANDARDS.md`

This pipeline adds **Section 4** to `INGESTION_STANDARDS.md`:

> **Section 4: Markdown → Anki Parsing Conventions**
> - `##`/`###` headings + following paragraph → `Wisdom-Basic`
> - Paragraph with block ID + bolded phrase → `Wisdom-Cloze`
> - `**Term** — Definition` pattern → `Wisdom-Basic`
> - Table rows → `Wisdom-Basic` (first column = front)
> - Checkboxes → skipped
> - Notes without `id` YAML field → rejected
> - `mastery_score` YAML field → seeds Cortex `MasteryScore`
