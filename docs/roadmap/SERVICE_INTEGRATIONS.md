# Service: Integrations (Anki Sync & MCP Orchestrator)

## Core Concept
The `Integrations` service is the **Single Authoritative Bridge** between internal engines and external UI ecosystems (Anki, Obsidian, Logseq).

## 1. The Ingestion Pipeline (Gap R3 Fix)
All Expert Agents route their knowledge creation requests through this service.
1.  Expert sends JSON payload (`action: create_note` or `create_card`).
2.  `Integrations` service validates the schema.
3.  `Integrations` maps the request to the correct local MCP server.
4.  If the server is offline, `Integrations` queues the task in `Cortex`.

## 2. MasteryScore Arithmetic (Gap R2 Fix)
Mastery is calculated on a 0.0 to 1.0 scale. 

### Clamping & Bounds
*   **Minimum:** 0.0 (Zero knowledge).
*   **Maximum:** 1.0 (Dominated).
*   **Rule:** Deltas are applied and then clamped: `score = Math.max(0.0, Math.min(1.0, score + delta))`.

### Grade Mapping
| Anki Grade | Action | Delta | Status Change |
| :--- | :--- | :--- | :--- |
| `1` (Again) | Fail | -0.30 | `FRAGILE` |
| `2` (Hard) | Pass | +0.05 | - |
| `3` (Good) | Pass | +0.15 | - |
| `4` (Easy) | Pass | +0.30 | `DOMINATED` |

## 3. The Scheduler Conflict (Gap R2 Fix)
To prevent "Dual Source of Truth" issues:
*   **Study in Anki:** Anki's internal scheduler (FSRS/SM-2) is the **Master Scheduler**. Wisdom is a passive observer, only syncing the grade signal to update the internal `MasteryScore`.
*   **Study in Wisdom Portal:** Wisdom's `Metabolism` engine is the **Master Scheduler**. 
*   **Sync Rule:** If a card is reviewed in Anki, Wisdom updates its `next_review_date` to match Anki's calculated date during the next sync poll.

## 4. Anki Polling Loop
*   **Cadence:** 15 minutes.
*   **Deduplication:** Uses `review_id` (Anki timestamp) checked against the `Cortex` processed events table.
