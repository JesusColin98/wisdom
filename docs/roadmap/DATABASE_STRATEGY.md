# Universal Database Strategy & Conflict Resolution

To minimize costs, maximize flexibility, and avoid "table bloat," Wisdom uses a **Universal Graph Schema** built on top of Postgres/Supabase using `jsonb` for dynamic attributes.

## 1. Unified Schema (Nodes & Edges)
Instead of 10+ tables for different concepts (Users, Facts, Signals, Rules), we use exactly two core tables:

### Table: `nodes`
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID (PK) | Unique identifier. |
| `type` | ENUM | `Fact`, `Signal`, `Concept`, `User`. |
| `payload` | JSONB | All dynamic data (e.g., Markdown content, URL, author). |
| `confidence` | FLOAT | 0.0 to 1.0. Higher weight wins in conflicts. |
| `requires_human` | BOOLEAN| True if involved in an unresolved conflict. |
| `ttl` | TIMESTAMP| Expiration date for Signals (Garbage Collection). |

### Table: `edges`
| Column | Type | Description |
| :--- | :--- | :--- |
| `source_id` | UUID (FK) | Origin node. |
| `target_id` | UUID (FK) | Destination node. |
| `relation` | ENUM | `THEORY_OF`, `CONTRADICTS`, `PREREQUISITE_OF`, `MASTERED_BY`. |

## 2. Fact Conflict Resolution (The Weight System)
Conflicts in knowledge are inevitable. They are modeled natively in the graph:
1. Two nodes exist: Node A (Weight 0.9, Source: Book) vs Node B (Weight 0.3, Source: Tweet).
2. Cerebellum draws an edge: `A --[CONTRADICTS]--> B`.
3. **Retrieval**: `Cortex` always returns the node with the highest `confidence` automatically.
4. **Resolution**: The UI shows a warning. A human resolves it by deleting the loser or merging them, clearing the `requires_human` flag.

## 3. Learning Paths (Obsidian MOC Standard)
Wisdom adopts the **Map of Content (MOC)** pattern popularized by the Obsidian community.
- A Learning Path is NOT a rigid table hierarchy.
- It is a single Node of type `Concept` (e.g., "Chess Openings MOC").
- The `payload` contains Obsidian Canvas (`.canvas`) JSON or Markdown with `[[Wikilinks]]` representing the DAG (Directed Acyclic Graph) of learning.
- **Flexibility**: When a new sub-topic is discovered, it is simply injected into the MOC payload as a new link. External viewers (like Obsidian) can render it natively.
