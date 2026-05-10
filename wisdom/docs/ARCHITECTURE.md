# Project Wisdom: The Knowledge Runtime

## Overview
Wisdom is a high-performance **Knowledge Runtime** written in Go. Moving beyond static retrieval-augmented generation, it implements a dynamic, self-evolving system that orchestrates topological graph reasoning, linear state-space modeling, and neural-socratic dialogue. It follows a biological metaphor for its component organization, now optimized for the May 2026 State-of-the-Art.

## Scientific Philosophy: Neuro-Architectural Engineering
Project Wisdom is a **Cognitive Architecture** grounded in four scientific pillars:

### 1. Neurobiology: Neural-Socratic Gating
Just as the biological Thalamus dynamically filters and modulates information based on cortical feedback, Wisdom's **Thalamic Gate** implements **Neural-Socratic Graph Dialogue**. Retrieval is not a one-shot operation but an iterative loop where the system identifies knowledge gaps and issues targeted diagnostic messages ($\pi_{map}$) to refine the retrieval space based on model uncertainty ($\sigma$).

### 2. Global Workspace Theory: Information Broadcasting
Wisdom implements **Global Workspace Broadcasting**, where specialized processes (vector search, path enumeration, etc.) compete for access to a limited-capacity "global workspace". Rhythmic broadcasting, mimicking **10 Hz alpha oscillations**, synchronizes retrieval with reasoning turns ("Thinking in Documents"), ensuring context is injected only when behaviorally relevant.

### 3. Psychology: Piagetian Synaptic Plasticity
Knowledge evolution follows **Piagetian Dynamics**:
- **Assimilation:** Grounding new inputs into the existing cognitive schema via **Schema-Constrained Generative Memory (SCG-Mem)**.
- **Accommodation:** Expanding the **Prefix Trie** with novel concepts when information falls outside the current epistemic boundary.
- **Structural Safety:** Use of Prefix Tries ensures 100% retrieval validity by mathematically masking invalid tokens at the logit level, eliminating structural hallucinations.

### 4. Biology: Metabolic Homeostasis & Adaptive Compression
Wisdom treats tokens and compute as **Metabolic Currency**.
- **TSR (Token-to-Signal Ratio):** Measures the nutritional value of data.
- **Adaptive Contextual Compression (ACC):** Dynamically prioritizes critical information using dependency-aware sentence fusion.
- **Context Folding:** Agents summarize their reasoning branches to preserve the main context window's TSR.

---

## Core Components
...

### 1. The Cortex (`pkg/cortex`)
**Role:** Stratified Semantic Memory & Schema-Constrained Reasoning.
- **Stratification:**
    - **Superficial Knowledge (Hot):** High-frequency facts and current context stored in SQLite (`storage_node.go`) for $O(1)$ retrieval.
    - **Deep Knowledge (Cold):** Long-term relational meshes and archived historical patterns stored in the Graph Mesh (`storage_graph.go`).
- **Mechanism:** Implements **SCG-Mem** and modular storage handlers (Node, Graph, Vector, Spaced Repetition).
- **Link-Based Multimodality:** Nodes store structured `ExternalLinks` (Bugs, Videos, Images) and `SourceMimeType`. Provenance tracking includes `author_id`, `edit_history`, and `certainty_weight`.
- **Active Pruning:** Implements "Synaptic Pruning" and auto-migration between `HOT` and `COLD` strata during REM cycles.

### 2. The Thalamus (`pkg/thalamus`)
**Role:** Executive Orchestration & Neural-Socratic Gating.
- **Responsibility:** Intent classification, RAG orchestration, and Global Workspace management.
- **Multi-User isolation:** Orchestrates isolated `Session` objects and `Hippocampus` traces per `UserID`, while accessing a **Shared Cortex**.
- **Intent Classifier v2:** Automatically detects `CODE`, `RELATIONAL`, or `GENERAL` queries to route between GrepRAG and Graph-RAG.
- **Cost Switch:** Implements `DetermineRetrieveMode` to toggle between **Low-Cost** (HOT only) and **High-Cost** (HOT+COLD + Deep Graph Crawl) based on model uncertainty and token budget.
- **Neural-Socratic Loop:** Implements an iterative dialogue between the LLM and the Graph. Generates diagnostic messages ($\pi_{map}$) to bridge knowledge gaps or resolve ambiguity.

- **Broadcasting:** Manages a limited-capacity "Global Workspace" where specialized processes compete for access.
- **Rhythmic Gating:** Synchronizes retrieval with reasoning turns ("Thinking in Documents") via 10 Hz alpha oscillations (simulated).
- **Chat:** Provides grounded conversational capabilities by searching the Cortex before prompting the LLM, applying metabolic budgets to maximize TSR.

### 3. The Cerebellum (`pkg/cerebellum`)
**Role:** Motor Control & Linear State-Space Execution.
- **Responsibility:** Safe execution of retrieval and reasoning tasks at billion-node scale.
- **Mechanism:** Integrates **Graph Mamba** blocks with **Cross-Batch Aggregation (COMBA)**.
    - Uses structured recurrence to capture long-range dependencies with $O(N)$ complexity.
- **Resilience:** Implements **GraphTARIF** (Linear Graph Transformer) for high expressivity without quadratic compute bottlenecks.
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

---

## Specialized Cognitive Capabilities

### Neural-Socratic Dialogue (The Inquirer)
Wisdom extends its generic memory to support iterative retrieval refinement.
- **Diagnostic Mapping:** Translates model uncertainty ($\sigma$) into targeted graph edits or seed expansions ($\Delta G$).
- **Seed Expansion:** Identifies new entry points in the graph not initially visible through semantic similarity alone.

### Coaching & Adaptive Learning (The Coach)
Wisdom extends its generic memory to support personalized coaching (e.g., Chess, Languages) without domain-specific code.

- **Genetic Patterns:** Knowledge is decomposed into atomic `PATTERN` and `CONCEPT` nodes.
- **State Links:** The relationship between a `PERSON` and a `PATTERN` is modeled via semantic links:
    - `MASTERED_BY`: High confidence, extended review interval.
    - `STRUGGLES_WITH`: Low confidence, frequent review.
- **Dependency Awareness:** `PREREQUISITE_OF` links enable the engine to detect when a user is missing foundational knowledge required for a more advanced concept they are attempting to master.
- **Spaced Repetition (SM-2):** The Thalamic Scheduler implements the SM-2 algorithm. It automatically calculates the next review date based on user recall quality, ensuring efficient long-term retention.

---

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
