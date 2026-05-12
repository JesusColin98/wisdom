# Project Wisdom: Session Lifecycle & Cognitive Learning

## 1. Overview
Project Wisdom is designed as a **Cognitive Engine**, moving beyond static databases to an architecture that mimics biological learning. It leverages principles from neurobiology and psychology to manage SRE knowledge efficiently.

## 2. Interaction Model: Gemini CLI & Wisdom

### A. When does Gemini CLI call Wisdom?
Gemini CLI acts as the **Prefrontal Cortex** (the executive), while Wisdom acts as the **Limbic System & Memory Cortex**.

1.  **Context Gating (Retrieval):** Before starting a complex task, Gemini CLI queries Wisdom to "remember" relevant playbooks or previous incident patterns.
2.  **Anchoring (Learning):** When Gemini CLI discovers a new fix or architectural rule, it calls `anchor_wisdom` to store it permanently.
3.  **Proprioception (Observability):** Gemini CLI asks for a "Pulse" to feel the state of production via Wisdom's sensory buffers.

### B. Data Flow
- **Input:** Raw logs, terminal outputs, and user directives are fed into Wisdom.
- **Processing:** Wisdom filters this through the **Thalamus** (gating) to ensure only high-signal data consumes the context window.
- **Storage:** Data is layered. High-relevance data stays in the context; low-relevance but historical data is "stratified" into the **Cortex**.

---

## 3. The Cognitive Cycle (Lifecycle)

### Phase 1: Sensory Ingestion (The Sensory Buffer)
Wisdom constantly ingests signals from the environment (IRM, Monarch). These are kept in `pkg/sensory` as transient "background noise" until the system's attention is drawn to them.

### Phase 2: Active Working Memory (The Hippocampus)
Every interaction during a session is recorded in the **Hippocampus**. This is a volatile, high-speed buffer that stores:
- User queries.
- Tool execution results.
- LLM reasoning steps.

### Phase 3: Consolidation (The Automated REM Cycle)
The REM cycle (Rapid Evidence Mapping) is now **independent and automated**:
1.  **Trigger:** A Cloud Scheduler job hits `POST /rem/all` daily.
2.  **Inactivity Gating:** Wisdom identifies "inactive" sessions (those not updated in the last hour).
3.  **Harvesting:** It reads persistent logs from the SQLite `session_logs` table.
4.  **Distillation:** It uses an LLM to identify "Universal Truths".
5.  **Novelty Gating:** Before anchoring, Wisdom generates an embedding and performs a similarity check. If similarity > 0.92, it triggers **Synaptic Strengthening** on the existing node instead of creating a duplicate. This ensures the Cortex remains critical and avoids "knowledge bloat".
6.  **Anchoring:** Unique truths are written to the **Cortex** as new nodes.
7.  **Cleanup:** Once consolidated, transient logs are cleared to maintain metabolic efficiency.

### Phase 4: Synaptic Layering (Temporal Memory)
When a node is updated, the old version is not deleted. It is moved to `node_history` (The "Deep Brain"). This allows for **Deep Recall**, where the system can analyze how a system evolved over time.

---

## 4. Evidence-Based Implementation

### Use Cases Supported:
- **Analogy Retrieval:** Finding "that incident that looked like this one" via Hybrid Search.
- **Dynamic Skill Acquisition:** Learning a new tool interface during an incident via Neurogenesis.
- **Efficiency Optimization:** Tracking TSR (Token-to-Signal Ratio) to ensure the LLM isn't wasting energy on low-value data.

### Technical Limitations:
- **Cold Start:** The system takes ~2 seconds to initialize its semantic substrate.
- **Synaptic Latency:** Consolidation (REM) is an asynchronous process; new knowledge might take a few minutes to appear in "Deep Search".
- **Hardware Scale:** Local vector search uses an **Evolutionary Substrate**. It starts with kNN (linear) for perfect accuracy and automatically promotes to **RP Forest** (sub-linear) beyond 5,000 nodes, supporting millions of nodes with constant latency.

---

## 5. Recommended Future Features
1.  **Mirror Neurons:** Ability to observe other Gemini CLI instances and learn from their successes/failures in real-time.
2.  **Dopamine Rewards:** A feedback loop where the user can "upvote" a piece of wisdom, increasing its synaptic weight and retrieval priority.
3.  **Cognitive Sleep:** Automated "defragmentation" of the SQLite database during low-usage periods to optimize retrieval paths.

---

## 6. Scientific Values
- **Synaptic Plasticity:** Knowledge is not static; it strengthens with use and fades with neglect.
- **Thalamic Gating:** Prevention of "Context Overload" by strictly filtering sensory input.
- **Hierarchical Temporal Memory:** Understanding that technical truth is time-dependent.
