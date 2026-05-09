# Project Wisdom Architecture

## Overview
Wisdom is a high-performance Cognitive SRE Engine written in Go. It is designed to replace the legacy Python-based `nexusstate` substrate with a more parallel, observable, and resilient architecture. It follows a biological metaphor for its component organization.

## Scientific Philosophy: Biological Engineering
Project Wisdom is not just a software system; it is a **Cognitive Architecture** grounded in three scientific pillars:

### 1. Neurobiology: The Thalamic Gate
Just as the biological Thalamus filters sensory information before it reaches the cerebral cortex, Wisdom implements **Thalamic Gating**. By strictly validating inputs and filtering "background noise" (logs/metrics) through high-signal summaries, the system prevents "Context Overflow", ensuring the Gemini CLI (Prefrontal Cortex) only processes what is vital.

### 2. Psychology: Synaptic Plasticity & REM Cycles
Knowledge in SRE is not static. Wisdom follows the principle of **Synaptic Plasticity**:
- **Learning:** Every session is a period of "Active Learning" where facts are gathered in the **Hippocampus**.
- **Consolidation:** The **REM Cycle** (Rapid Evidence Mapping) is our technical equivalent of sleep. It distills transient session data into "Universal Truths" (Synapses) within the **Cortex**.
- **Forgetting:** Instead of crude deletion, Wisdom uses **Synaptic Layering**. Cold knowledge is stratified into history, mimicking how the human brain prioritizes recent, high-utility information while keeping long-term memories in deeper layers.

### 3. Biology: Metabolic Homoeostasis
A cognitive engine must be efficient to survive. Project Wisdom treats tokens and compute as **Metabolic Currency**.
- **TSR (Token-to-Signal Ratio):** Measures the nutritional value of data. High signal with low token cost is the goal.
- **Circuit Breakers:** Mimics biological refractory periods, where a failing "organ" (tool) is temporarily inhibited to prevent system-wide exhaustion.

---

## Core Components
...

### 1. The Cortex (`pkg/cortex`)
**Role:** Semantic Memory.
- **Responsibility:** Persisting facts, observations, and relationships (nodes and links).
- **Mechanism:** Uses SQLite (pure Go) for high-fidelity provenance and **RP Forest** (Random Projection Forest) for scalable vector search.
- **Novelty Filter:** Implements a "Gating" mechanism where new knowledge is compared against existing nodes. If similarity > 0.92, it triggers "Synaptic Strengthening" (updating confidence) instead of creating duplicates.
- **Logic:** Implements Personalized PageRank (PPR) for graph propagation to re-rank knowledge based on relevance to a seed set of nodes.

### 2. The Thalamus (`pkg/thalamus`)
**Role:** Executive Orchestration & Gating.
- **Responsibility:** Admission control, session management, and planning.
- **Gating:** Implements "Reactive Gating" using JSON Schema validation to ensure all incoming requests are structurally sound.
- **Orchestration:** Orchestrates knowledge retrieval and tool triggering, applying metabolic budgets to maximize the Token-to-Signal Ratio (TSR).
- **Chat:** Provides grounded conversational capabilities by searching the Cortex before prompting the LLM.

### 3. The Cerebellum (`pkg/cerebellum`)
**Role:** Motor Control (Execution).
- **Responsibility:** Safe, non-blocking execution of tools and actions.
- **Runner:** Managed worker pool with semaphore-based concurrency control.
- **Resilience:** Implements a Circuit Breaker pattern (Closed, Open, Half-Open) per tool to prevent cascading failures.
- **Jobs:** Asynchronous job tracking with unique IDs and state lifecycle (`Pending` -> `Running` -> `Finished/Failed`).

### 4. Metabolism (`pkg/metabolism`)
**Role:** Resource Regulation.
- **Responsibility:** Tracking token consumption, latency, and costs.
- **Metrics:** Calculates Token-to-Signal Ratio (TSR) and Metabolic Rate.
- **Budgeting:** Enforces hard limits per session to prevent runaway resource usage.

### 5. API Layer (`pkg/api`)
**Role:** Communication Interface.
- **Responsibility:** Exposing Wisdom's internal state via a high-performance REST API.
- **Observability:** 100% instrumented with OpenTelemetry (OTel) for tracing "thinking steps".

## Substrate Strategy: Evolutionary Performance
Wisdom uses an **Evolutionary Substrate** designed for low-cost starts and extreme scalability.

- **Tier 1 (Flat):** For < 5,000 nodes, Wisdom uses a linear SQLite scan. It is 100% accurate and requires zero extra RAM.
- **Tier 2 (RP Forest):** Beyond 5,000 nodes, the system promotes the substrate to a **Random Projection Forest**. This native Go index provides $O(\log N)$ search performance, enabling millions of nodes with millisecond latencies.
- **Tier 3 (Cloud Scale):** For multi-instance or >10M nodes, Wisdom is designed to offload vector search to **Firestore Vector Search** (see `substrate_firestore.go`).
- **Rationale:** This tiered approach ensures Wisdom is always "Low Cost" by default but "Extreme Scale" when the knowledge base matures.

## Infrastructure & Deployment

### Cloud Run Specification
Wisdom is optimized for Google Cloud Run. Due to the high-performance nature of the Go binary:
- **Cold Start:** Redundant "Keep-Alive" functions are no longer required. The engine reaches an operational state in <2 seconds.
- **Identity:** The engine must run under the `nexusstate-mcp-sa` Service Account to maintain access to SRE tools and Secret Manager.
- **Storage:** The `wisdom.db` file should be stored on a persistent volume mount (GCS Fuse) to ensure semantic continuity across container restarts.

### Secret Management
LLM API keys and backend credentials should never be stored in environment variables. Wisdom is designed to fetch these from **GCP Secret Manager** during the Thalamic initialization phase.

## Data Flow
1. **Request:** Enters via `/chat` or tool call.
2. **Gating:** Thalamus validates parameters against JSON Schema.
3. **Retrieval:** Thalamus queries Cortex for relevant context (Nearby Wisdom).
4. **Action:** Cerebellum executes tools if needed, tracked via Jobs.
5. **Update:** Outcomes are recorded in the Cortex as new nodes/links.
6. **Reporting:** Metabolism tracks the entire cycle's cost and efficiency.
