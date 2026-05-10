# Advanced Graph RAG & Orchestration Strategy

Project Wisdom employs a tiered, multi-strategy Retrieval-Augmented Generation (RAG) pipeline, evolving into a **Knowledge Runtime**. This document details the high-efficiency frameworks, orchestration logic, and fallback mechanisms designed to maximize accuracy while minimizing token metabolism and latency.

## 1. S-Path-RAG (Neural-Socratic Dialogue)

### Technical Specification
- **Core Principle:** Transition from one-shot retrieval to an iterative dialogue loop between the LLM and the Knowledge Graph.
- **Mechanism:**
    1. **Diagnostic Messaging:** When the LLM identifies a knowledge gap or ambiguity, it generates a concise diagnostic message.
    2. **Diagnostic Mapper ($\pi_{map}$):** Translates these messages into targeted graph edits or seed expansions ($\Delta G = \pi_{map}(Q, C, \sigma)$).
    3. **Seed Expansion:** Identifies new entry points in the graph that were not initially visible through semantic similarity alone.
- **Benefit:** Treats retrieval as a conditional, non-linear operation, superior to traditional one-shot methods.

## 2. TERAG (Token-Efficient Graph RAG)

### Technical Specification
...

## 3. Path-Augmented Reasoning (AGRAG)

### Technical Specification
- **Core Principle:** Formulate graph reasoning as a **Minimum Cost Maximum Influence (MCMI)** subgraph generation problem.
- **Mechanism:**
    1. **Flow-Based Pruning:** Identifies key relational paths between retrieved nodes to reduce redundancy and guide the reasoning chain.
    2. **MCMI Generation:** Selects subgraphs that maximize query-relevant influence while minimizing edge traversal "costs" (token overhead/logical jumps).
- **Benefit:** Surpasses global thematic community summaries by preserving fine-grained relational traces necessary for high-precision multi-hop QA.

## 4. GrepRAG (Index-Free Lexical Retrieval)
...

### Technical Specification
- **Core Principle:** Multi-hop reasoning guided by Reinforcement Learning (RL).
- **Optimization:** Fine-tuned via **Direct Preference Optimization (DPO)** using a 7-factor reward vector (Relevance, Redundancy, Efficiency, Correctness, etc.).
- **Curriculum:**
    - **Discovery Phase:** Broad exploration of retrieval pathways.
    - **Refinement Phase:** Concise synthesis of evidence-backed answers.
- **Benefit:** Boosts Exact Match accuracy by +4.6 points while reducing average retrieval depth by 15%.

## 5. Synaptic Reinforcement (Dopamine-like Feedback)

### Technical Specification
- **Core Principle:** Dynamically adjust the "wiring" of the Knowledge Substrate based on successful reasoning outcomes.
- **Mechanism:**
    1. **Usage-Based Strengthening:** Every time a node is included in a final context that leads to a successful user interaction, its `ImpactScore` is incremented.
    2. **Associative Linking:** Nodes that are co-retrieved and used together in the Global Workspace develop stronger directed links (`ASSOCIATED_WITH`).
    3. **Certainty Propagation:** Positive reinforcement from the user (e.g., "This was helpful") propagates certainty weights upstream through the dependency chain.
- **Benefit:** Self-optimizing memory that "remembers" which conceptual paths are most useful for specific domains (e.g., Chess tactics vs English idioms).

---

## 6. Metabolism-Driven Orchestration (Low/High Cost Switch)

The **Thalamus** dynamically modulates its reasoning depth based on the **Metabolic Budget**:

| Mode | Trigger | Engine Path | TSR Goal |
| :--- | :--- | :--- | :--- |
| **LOW_COST** | Default / Low Uncertainty | Naive RAG $\rightarrow$ SQLite Cache | High efficiency, low latency |
| **HIGH_COST**| Uncertainty > 0.7 / Multi-hop| TERAG $\rightarrow$ Neural-Socratic Loop | Maximum precision, factual grounding |

### Stratified Retrieval Workflow
1. **Perception:** Scan **Superficial Knowledge** (SQLite) for direct hits ($O(1)$).
2. **Evaluation:** If signal is low or intent is complex, upgrade to **Deep Knowledge** (Graph Mesh).
3. **Synthesis:** Inject path latents into the **Global Workspace** for model attention.

---

## Automated Orchestration Logic (Thalamus)

The **Thalamus** acts as the CPU (Cognitive Processing Unit), routing queries to the optimal retrieval engine based on detected intent:

| Detected Intent | Engine | Rationale |
| :--- | :--- | :--- |
| **Code/Identifiers** | GrepRAG | Lowest latency, high lexical precision. |
| **Relational/Entities**| TERAG | Token-efficient multi-hop mapping. |
| **Complex/Multi-step**| EVO-RAG | Agentic refinement for logic-heavy tasks. |
| **General/Creative** | Naive RAG | Simple similarity for flat knowledge. |

## 7. Intent-Driven Retrieval Orchestration (v2)

The **Thalamus** now employs a high-fidelity **Intent Classifier v2** to select the optimal retrieval pattern before execution:

- **CODE Intent:** Triggers **GrepRAG**. Instead of Confusing Vector Embeddings, we use sub-second lexical search (Zero-Index) via `ripgrep` for absolute identifier precision.
- **RELATIONAL Intent:** Triggers **MCMI (Minimum Cost Maximum Influence)** graph traversal. Uses deep semantic crawls (3+ iterations) to resolve "why" questions and causal chains.
- **HIERARCHY Intent:** Triggers **Tree-RAG** logic to traverse parent-child relationships (e.g., org charts or file systems).
- **GENERAL Intent:** Uses standard **Stratified RAG** (SQLite Hot Cache) to minimize latency and token cost.

### GrepRAG: Lexical Precision
For software engineering tasks, vector similarity often fails to distinguish between `get_user` and `list_users`. GrepRAG bypasses the index entirely, performing real-time grep-based extraction of code snippets, which are then injected into the Global Workspace as `HOT` stratum context.

---

## 8. Multi-User Shared Cortex Mandate

Wisdom is designed as a **Single Brain, Multiple Personas** system:
- **Shared Cortex:** All knowledge units (Nodes) and relational links (Edges) are stored in a unified global substrate to maximize cross-domain utility.
- **Isolated Thalamus:** User sessions, transient memory (Hippocampus), and active reasoning threads are isolated by `UserID`.
- **Expertise Tracking:** The system tracks `ImpactScore` reinforcement per user-interaction, but the resulting "wisdom" is immediately available to the entire ecosystem.

---

## Redundancy Rationale

- **Why TERAG over GraphRAG?** Standard GraphRAG is economically unsustainable for enterprise indexing ($33k for 5GB). TERAG delivers 80% of the accuracy at 3% of the token cost.
- **Why GrepRAG over Vector Search for code?** Vector embeddings often conflate "similar sounding" code patterns. `rg` provides the literal precision required for compiler-level accuracy with zero indexing overhead.
- **Why EVO-RAG is restricted?** Agentic reasoning is token-intensive. It is only triggered for `MULTI_HOP` intents to preserve the metabolic budget.

## Verification & Testing
- **Integration Test:** Simulate a "code search" intent and verify `ripgrep` trigger.
- **TSR Benchmark:** Measure token usage for a complex KG query vs standard extraction.
- **Fallback Test:** Intentionally break the graph index and verify auto-pivot to Fusion Retrieval.
