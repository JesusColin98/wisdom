# SOTA Graph ML Research & Architectures

This document synthesizes state-of-the-art (SOTA) research in Graph Machine Learning and its integration with Large Language Models. These technologies represent the evolutionary path for Project Wisdom's **Cortex** and **Thalamus**.

## 1. Schema-Constrained Generative Memory (SCG-Mem)

### Technical Mechanism
- **The Problem:** "Structural Hallucination," where agents generate memory keys or entities that do not exist in the underlying index.
- **The Solution:** SCG-Mem reformulates memory access as a generative process governed by a **Prefix Trie**.
- **Implementation:** Masking invalid tokens at the logit level during generation. Probability $P_S$ is constrained to valid strings in the Trie.
- **Piagetian Dynamics:**
    - **Assimilation:** Grounding new inputs into the existing schema using constrained decoding to map information to valid keys.
    - **Accommodation:** Expanding the Prefix Trie with novel concepts when information perplexity is high.

## 2. Linear Attention & Graph Transformers (GraphTARIF)

### Technical Mechanism
- **The Problem:** Standard Transformer attention scales at $O(N^2)$, creating a memory bottleneck for large-scale graphs with thousands of nodes.
- **The Solution:** **GraphTARIF** proposes a linear attention mechanism for Graph Transformers.
- **Implementation:** By decomposing the attention matrix into lower-rank components or using kernel-based approximations, complexity is reduced to $O(N)$.
- **Benefit:** Allows Wisdom to process dense knowledge meshes without exponential increases in RAM usage or latency.

## 2. Hybrid LLM-GNN Integration (GLOW)

### Technical Mechanism
- **Architecture:** **GLOW** (IBM Research) utilizes a two-stage hybrid approach:
    1. **Structural Filter (GNN):** A Graph Neural Network (GNN) scans the global topology to identify high-probability candidate nodes and sub-graphs based on structural relevance.
    2. **Semantic Reasoner (LLM):** The LLM performs deep reasoning only on the filtered candidates, synthesizing the final response.
- **Benefit:** Combines the global structural "intuition" of GNNs with the local semantic "intelligence" of LLMs.

## 3. SSM and Mamba-graph (Linear Efficiency)

### Technical Mechanism
- **Evolution:** State Space Models (SSM) like **Mamba** evolve from Recurrent Neural Networks (RNNs), using linear differential equations to map inputs to outputs via a latent state.
- **Graph Application:** **Mamba-graph** applies SSM logic to graph structures, capturing local topological structures and global long-range interactions with linear computational complexity.
- **Cross-Batch Aggregation (COMBA):** Updates node embeddings across overlapping batches to address sampling bias in large-scale GNN training.
- **Application:** Ideal for processing "infinite" sequences of graph updates with linear complexity and constant memory.

## 4. Zero-Shot Inference (LINKLLM)

### Technical Mechanism
- **Core Principle:** **LINKLLM** demonstrates that LLMs can infer missing relationships in incomplete Knowledge Graphs via **Zero-Shot Reasoning**.
- **Logic:** By providing the LLM with the context of two nodes (e.g., "Node A: Server Error" and "Node B: Disk Full"), the model can correctly predict a `CAUSED_BY` link without prior training on that specific graph topology.
- **Application:** Enables "Automatic Neurogenesis" in the Cortex when gaps in documentation are detected.

## 5. Graph Attention Networks (GAT) & DGI

- **GAT:** Allows the model to learn the **importance** of each connection dynamically. Instead of static weights, attention coefficients are learned based on node features, allowing Wisdom to prioritize "High-Signal" synapses.
- **Deep Graph Infomax (DGI):** A self-supervised technique for learning node representations by maximizing mutual information between local and global graph summaries.
- **Large-Scale Risk Precedents:** Industry precedents for large-scale fraud detection emphasize the use of deep learning and graph-like feature stores for low-latency online serving, justifying Wisdom's transition to a billion-scale SSM substrate.

---

## Technical Concepts for Scale

- **Complexity Optimization:** Shifting from $N^2$ (Standard Transformers) to $O(N \log N)$ (RP Forest) or $O(N)$ (GraphTARIF/Mamba).
- **Structural-Semantic Hybridization:** Using GNNs for the "Where" (topology) and LLMs for the "What" (content).
- **Multi-hop Navigation:** The ability to traverse 3+ steps in a graph while maintaining context without "wandering agent" degradation.

---

## Technical Implementation Memos (For Autonomous Coding)

### Memo 1: Deep Graph Infomax (DGI) Logic
- **Objective:** Self-supervised node embedding without labels.
- **Workflow:**
    1. Define a **Discriminator** $\mathcal{D}$ that takes a node embedding and a global graph summary $s$.
    2. Maximize the mutual information between the local patch representation and the global summary.
    3. Generate "Corrupted" graph versions (shuffling nodes) to serve as negative samples for the discriminator.
- **Goal:** Allow Wisdom to cluster "Abnormal" SRE nodes (e.g., rare error patterns) without human labeling.

### Memo 2: Generalized Risk/Abuse Pattern (Entity Risk Scoring)
- **Objective:** Real-time relationship scoring for entity risk in social or logistics meshes.
- **Logic:**
    1. Construct a **Star-Mesh hybrid**: Central `ENTITY` node linked to unique identifiers like `IP`, `DEVICE_ID`, or `LOCATION` (Star).
    2. Identifier nodes are linked to other `ENTITY` nodes (Mesh).
    3. **Feature Extraction:** Use pointer-chasing to calculate `SharedAssetCount` and `RelationshipVelocity` within 2 hops.
    4. **Inference:** Pass these graph-derived features to a shallow classifier or the LLM (via GWT Broadcasting) for risk assessment.
- **Goal:** Enable sub-second detection of coordinated abuse patterns without hardcoded domain logic.

