# Subsystem: Wisdom-Chess-Engine (Analysis Service)

A specialized microservice for Chess-specific logic, analysis, and pedagogical data processing.

## 1. Engine Integration: Stockfish
- **Deployment**: Lightweight C++ binary (optimized for Cloud CPUs) running in a dedicated container.
- **Protocol**: gRPC wrapper over standard UCI (Universal Chess Interface).
- **Tasks**:
    - **Blunder Detection**: Identify score drops > 2.0 centipawns in user games.
    - **Move Recommendation**: Provide the top 3 optimal moves for a given FEN.
    - **Theme Classification**: Detect tactical motifs (Pin, Fork, Skewer) from board states.

## 2. Data Sources & Theory
Wisdom leverages the **Lichess Open Database** for theory and high-quality game samples.

### A. The Theory Table (Spanner/SQL)
A specialized table for "Consolidated Chess Knowledge":
- `FEN`: The primary key.
- `PopularMoves`: JSON list of SAN moves and win/loss percentages from Lichess.
- `MasteryLevel`: Calculated aggregate based on user's `Trace`.

### B. The Game Archive (GCS)
- Stores raw **PGN** files for deep archival.
- Indexed in Firestore with metadata pointers.

## 3. Pedagogical Use Cases

| Case | Mechanism | Goal |
| :--- | :--- | :--- |
| **Tactical Drill** | Fetch FENs from Lichess Puzzles. | Test user on specific failed motifs. |
| **Opening Prep** | Query Theory Table for branch points. | Build user-specific opening repertoire. |
| **Post-Mortem** | Run Stockfish on user PGN. | Detect patterns of error. |

## 4. Interface (gRPC)

```protobuf
service ChessEngine {
  rpc AnalyzeGame(PGNRequest) returns (stream AnalysisResult);
  rpc GetTheory(FENRequest) returns (TheoryData);
  rpc DetectMotifs(FENRequest) returns (MotifList);
}

message AnalysisResult {
  string move_san = 1;
  float evaluation = 2;
  bool is_blunder = 3;
  string suggested_alternative = 4;
}
```
