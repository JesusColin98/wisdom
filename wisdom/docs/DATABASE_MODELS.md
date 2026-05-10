# Database Models & Latent Graph Learning

This document analyzes the storage substrates and edge-construction techniques utilized by Project Wisdom to enable high-fidelity semantic memory and multi-hop reasoning.

## 1. Substrate Strategy: Billion-Scale Efficiency

### OdinANN: Direct Insert Stability
- **Problem:** Performance degradation during index updates in traditional on-disk indices.
- **Solution:** **OdinANN** introduces "Direct Insert," writing vectors directly to the on-disk index.
- **Mechanism:** GC-free update combining by overprovisioning disk space. Reserved free record slots in pages enable out-of-place updates without expensive garbage collection.
- **Impact:** Stable search latency even in highly dynamic, streaming datasets.

### GateANN: I/O-Efficient Filtered Search
- **Problem:** Pre-filtering breaks graph connectivity; post-filtering wastes SSD I/O.
- **Solution:** **GateANN** uses "Graph Tunneling" to decouple graph traversal from vector retrieval.
- **Impact:** I/O-efficient filtered search (e.g., by `SERVICE_ID` or `TIMESTAMP`) on an unmodified graph index.

---

## 2. SQL vs. Graph Databases (Multi-hop Efficiency)
...

### Technical Advantage
- **Continuous Representation:** Unlike discrete keywords, embeddings represent entities in a high-dimensional continuous vector space.
- **Semantic Continuity:** Allows the system to detect that "Server Latency" and "RPC Timeout" are semantically adjacent even if they share zero common words.
- **Dimensionality:** Wisdom standardized on **768-dimensional** embeddings to balance semantic resolution with computational cost.

---

## 3. Latent Graph Learning (Automatic Edge Construction)

Project Wisdom does not rely solely on explicit links. It uses **Latent Graph Learning** to discover "Hidden Synapses" based on embedding similarity.

### Technique A: k-Nearest Neighbors (k-NN)
- **Logic:** For every new node $N$, the system automatically creates edges to its $k$ closest neighbors in vector space.
- **Parameter:** $k = 5$ to $10$.
- **Use Case:** Clustering related concepts and building "Local Knowledge Hubs".

### Technique B: $\epsilon$-Neighborhood (Threshold-Based)
- **Logic:** Create an edge between node $A$ and node $B$ if their Cosine Similarity ($S$) exceeds a threshold $\epsilon$.
- **Parameter:** $\epsilon = 0.88$ (High semantic confidence).
- **Use Case:** Identifying synonyms and redundant documentation fragments.

### Technique C: Graph Attention (GAT) Weights
- **Logic:** Once edges are created via k-NN or $\epsilon$, a **Graph Attention Network (GAT)** learns a weight $\alpha_{ij}$ for each edge.
- **Effect:** The system "learns" that the link between `DISK_FAILURE` and `IO_WAIT` is more significant for troubleshooting than the link between `DISK_FAILURE` and `LOG_CLEANUP`.

## 4. Stratified Storage & Multimodality Schema

Project Wisdom utilizes a dual-tier storage strategy to balance performance with reasoning depth, managed via the `stratum` attribute.

### Tier 1: Superficial Knowledge (Hot Cache)
- **Engine:** SQLite with JSONB columns (implemented in `storage_node.go`).
- **Content:** Current session context, high-frequency metadata, and entity summaries.
- **Attributes:**
    - `stratum`: Set to `HOT`. These nodes are always searched during standard retrieval.
    - `impact_score`: Determines if the node survives pruning.
    - `source_mime_type`: Identifies the original format (e.g., `application/pdf`, `text/x-go`).
    - `external_links`: Structured JSON array of multimodal pointers.
    - `provenance`: `{"author": "LDAP", "version": 2, "certainty": 0.98}`.

### Tier 2: Deep Knowledge (Graph Mesh / Cold Storage)
- **Engine:** In-Memory Adjacency List + OdinANN (On-disk).
- **Content:** Full relational history, causal failure chains, and complex concept dependencies.
- **Stratum:** Set to `COLD`. These nodes are only accessed during high-uncertainty or deep-reasoning tasks via the **Cost Switch** gating logic.
- **Temporal Logic:** Every link carries `valid_from` and `valid_until` timestamps for historical lineage reconstruction.

### Synaptic Pruning (Utility Control)
During the **REM Cycle**, the system applies an entropy-based decay function to certainty weights:
$$W_{t+1} = W_t \cdot e^{-\lambda \Delta t}$$
where $\lambda$ is the entropy factor and $\Delta t$ is the time since last reinforcement. Nodes with $W < Threshold$ are migrated to `COLD` storage (archived) or deleted to maintain utility.


## Tests and Validation
- **Traversal Benchmark:** Measure latency for 3-hop traversal using SQLite CTE vs Go-native adjacency map.
- **Threshold Validation:** Verify that $\epsilon=0.88$ successfully links "Latency" to "Slow Response" while ignoring unrelated "Throughput" nodes.
- **k-NN Stability:** Ensure adding 1,000 nodes does not degrade retrieval performance of the nearest neighbor index.
