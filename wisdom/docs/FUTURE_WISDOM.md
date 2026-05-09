# Future Wisdom: Roadmap for Robustness

## Proposed Features

### 1. The Vector Cortex (Semantic Search)
- **Problem:** Keyword search (`LIKE %query%`) is insufficient for complex SRE analogies.
- **Solution:** Integrate a local vector database (e.g., a Go-native kNN implementation or `hnswlib` wrapper) for true semantic retrieval using embeddings.

### 2. Proactive Circuit Breaking (AI-Driven)
- **Problem:** Current circuit breaker is reactive (fails after N errors).
- **Solution:** Use metabolic metrics (high latency, high token usage with low TSR) to proactively "throttle" or "warn" about a tool before it hard-fails.

### 3. Distributed Substrate (Redis Integration)
- **Problem:** SQLite is local-first, making horizontal scaling difficult.
- **Solution:** Implement a Redis-backed `pkg/thalamus/cache` and `pkg/cerebellum/jobs` to allow multiple Wisdom nodes to share session state and execution results.

### 4. Continuous Knowledge Distillation
- **Problem:** SRE logs grow rapidly and become "noise".
- **Solution:** A background "REM Cycle" service that periodically scans recent tool outputs, distills them into Markdown notes via LLM, and creates links in the Cortex.

## Optimizing for Gemini CLI

### 1. Context Paging (Dynamic)
Wisdom can monitor the Gemini CLI context window and automatically trigger `anchor_wisdom` or "compression" steps when tokens exceed a certain threshold (reported via `pkg/metabolism`).

### 2. Semantic Prefetching
When a user asks a question, Wisdom can proactively propagate signals in the graph to pre-load related facts into the Thalamic session cache, reducing "thought" latency.

### 3. Tool-Call Dry Runs
Integrate JSON Schema validation deeper into the Gemini CLI prompt, allowing the LLM to "see" and "validate" its own tool-call arguments locally before sending them to the backend.
