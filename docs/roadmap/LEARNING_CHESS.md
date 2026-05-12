# Ajedrez Graph Representation: Tactical & Strategic Learning

Wisdom uses a hybrid storage and reasoning model for Chess, prioritizing **Consolidated Rules** as graph nodes and **Raw Games** as blob data.

## 1. Data Standards
- **Source of Truth**: **PGN (Portable Game Notation)**. Stores the full history and move sequence. Stored in GCS for high density (~1KB per game).
- **Searchable State**: **FEN (Forsyth-Edwards Notation)**. Stores board snapshots for instant tactical retrieval. Stored in Spanner/Cloud SQL.
- **Rules & Concepts**: Graph Nodes. (e.g., "Opposition in King Endgames").

## 2. Pedagogical Model: Learn from Play
Wisdom doesn't just store games; it distills them into your personal Knowledge Graph.

### Use Case A: Theory Study
1.  User provides an Opening Name (e.g., "Ruy Lopez").
2.  `Researcher` pulls top PGN master games from GCS/Web.
3.  `Curriculum` identifies the "Branching Points" (where moves deviate).
4.  Each Branching Point is materialized as a Node linked via `THEORY_OF`.

### Use Case B: Error Analysis (The "Struggle" Loop)
1.  Wisdom ingests user PGNs.
2.  `Cerebellum` runs engine analysis to find Blunders (Score drop > 2.0).
3.  The specific FEN of the blunder is analyzed for **Tactical Motifs** (Pin, Fork).
4.  If user fails a "Pin" consistently, `Trace` creates a link `@User STRUGGLES_WITH #Tactics-Pin`.
5.  `LearningEngine` prioritizes "Pin Defense" modules in the next session.

## 3. Node-Based Rules (Consolidated Knowledge)
Consolidated knowledge is stored as immutable "Hechos":
- **Node**: `Rule-Opposition`
- **Content**: "When Kings face each other with one square between them, the one NOT to move has the opposition."
- **Relations**: `PREREQUISITE_OF` -> `Rule-Pawn-Endgames`.
