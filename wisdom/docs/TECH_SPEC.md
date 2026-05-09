# Wisdom: Technical Specification (Better, Faster, Stronger)

## 1. Remediation of Legacy Gaps (nexusstate)
During the initial audit, several critical gaps were identified in the `nexusstate` substrate that **Wisdom** will natively resolve:

### A. The "Invalid Parameter" Fragility (Error -32602)
- **Gap:** The legacy MCP server is extremely brittle regarding JSON payloads. Missing optional fields or subtle type mismatches crash the tool call.
- **Wisdom Fix:** 
    - Implement a **Strict Schema Middleware** in Go using `jsonschema`.
    - Every tool in `pkg/cerebellum` will have a self-documenting schema.
    - Automatic "Coercion Layer": If a tool expects a string and gets an int, Wisdom will attempt a safe cast or provide a human-readable "Did you mean...?" error rather than a raw -32602.

### B. High Latency in State Retrieval
- **Gap:** Python's synchronous IO and Spanner query patterns in `get_session_state` cause noticeable lag.
- **Wisdom Fix:**
    - **Go Concurrency:** Use Goroutines for parallel fetching of Session Context + Cortex Wisdom + Skill Manifests.
    - **Tiered Caching:** Implement a local LRU cache in `pkg/thalamus` to serve frequently accessed session flags in <1ms.

### C. Observability Blind Spots
- **Gap:** Identifying *where* a reasoning step failed requires manual log digging.
- **Wisdom Fix:**
    - **Native OTel:** Every internal function call is wrapped in a Span.
    - **Trace-ID Injection:** Every tool call carries a trace-ID that links back to the "Neural Atlas" (the UI), allowing visual debugging of the thought process.

## 2. Porting Strategy: Logic Mapping
| Component | nexusstate (Python) | Wisdom (Go) | Improvement |
| :--- | :--- | :--- | :--- |
| Semantic Memory | Spanner / Memory | `pkg/cortex` (SQLite/RP Forest) | Local-first, sub-linear vector search ($O(\log N)$). |
| Session Gating | Central Executive | `pkg/thalamus` | Typed Admission Control. |
| Tool Runner | subprocess.run | `pkg/cerebellum` | Parallel execution, non-blocking. |
| Cost Control | Heuristic metrics | `pkg/metabolism` | Real-time TSR tracking. |

## 3. Performance Targets
- **Session Start:** < 100ms (vs legacy ~2s).
- **Graph Multi-hop Search:** < 50ms (vs legacy ~500ms).
- **TSR (Token-to-Signal Ratio):** 30% reduction in token waste through intelligent context paging.
