# Project Wisdom: Risk Mitigation & Validation Report

## 1. Graph Mamba: Competitive Advantage vs. Research Sinkhole
**Analysis:**
Research into mid-2026 SOTA reveals that **Graph Mamba (G-SSMs)** has effectively superseded GAT and GCN for large-scale graph reasoning. Breakthroughs like **DirGraphSSM** (directed graphs) and **NeuralWalker** (random walk sequence modeling) provide **Linear $O(N)$ scaling** and superior long-range dependency capture.
- **Risk Level:** Low (Validated as a production-grade standard in 2026).
- **Mitigation:** 
  - Standardize on **Mamba-2 kernels** in the Python sidecar for hardware-aware efficiency.
  - Implement a **Fallback Mechanism**: The Cerebellum will maintain a lightweight GCN/GAT baseline for small clusters or as a safety buffer if Mamba kernels are unavailable.

## 2. Sidecar Latency (Go-Python gRPC)
**Analysis:**
Network/serialization overhead between Go (Thalamus) and Python (Cerebellum) can degrade real-time performance.
- **Risk Level:** Medium.
- **Mitigation:**
  - **gRPC + Protobuf:** Replace JSON/HTTP with gRPC for the internal sidecar communication. This provides strongly-typed contracts and zero-copy serialization.
  - **Async-First Execution:** All ML-intensive tasks (GAT attention weights, Mamba path reasoning) will be executed as **Asynchronous Jobs**.
  - **Predictive Prefetching:** The Thalamus will broadcast "intent signals" to the Cerebellum sidecar *before* the retrieval loop completes, allowing the sidecar to warm up its state/tensors.

## 3. Dual-Tier Complexity (The Abstraction Guard)
**Analysis:**
Maintaining behavioral parity between SQLite (Edge/Local) and Neo4j/Postgres (Cloud/Enterprise) is a major maintenance risk.
- **Risk Level:** High.
- **Mitigation:**
  - **Behavioral TDD (Compliance Suite):** Implement a unified `EngineComplianceTestSuite` in Go. Every new `StorageEngine` implementation (SQLite, Postgres, Neo4j) *must* pass this 100% identical test suite before being merged.
  - **Strict Interface Segregation:** The `StorageEngine` is broken down into atomic interfaces (`NodeStore`, `VectorStore`, `GraphStore`). This prevents "Interface Bloat" and allows partial implementations (e.g., a pure Vector engine).
  - **Semantic Versioning for Schema:** DB schemas for all tiers will be versioned and migrated using a unified internal tool (`wisdom-migrate`) to prevent drift.

## 4. Architectural Decision Log (ADR)
| ID | Decision | Rationale |
| :--- | :--- | :--- |
| ADR-001 | **gRPC for Sidecar** | Minimizes 20-50ms serialization latency typical of HTTP/JSON. |
| ADR-002 | **Mamba-2 Standard** | Ensures Wisdom can scale to billion-node reasoning with $O(N)$ complexity. |
| ADR-003 | **Recursive SQL for Tier 1** | Avoids the cost of Neo4j for serverless/startup tiers while maintaining multi-hop capabilities. |

---
*This report serves as the technical validation for the maturation plan approved on May 11, 2026.*