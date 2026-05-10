# Graph Patterns and Technical Use Cases

This document provides a low-level technical specification for the graph patterns supported by Project Wisdom. It details their structural logic, performance characteristics, and standard orchestration paths for specialized use cases.

## 1. Tree Pattern (Hierarchical Topologies)

### Technical Specification
- **Structure:** Directed Acyclic Graph (DAG) where every node except the root has exactly one parent.
- **Enforcement Logic:**
    - **In-degree Constraint:** `InDegree(n) == 1` for all nodes except Root.
    - **Cycle Detection:** DFS-based validation on every `ADD_LINK` operation to prevent circular dependencies.
- **Storage Strategy (SQLite/Cortex):**
    - Optimized using **Recursive Common Table Expressions (CTE)** for path-to-root and subtree traversals.
    - `parent_id` column with an index for $O(1)$ child lookup.

### Standard Use Cases
- **Organization Charts:** Mapping reporting lines and management hierarchies.
- **Dependency Trees (Engineering):** Resolving library requirements or build-time prerequisites.
- **Bill of Materials (Logistics):** Decomposing products into atomic components.
- **Decision Trees:** Boolean logic paths for automated SRE troubleshooting.

### Orchestration Path
- **Intent:** `HIERARCHY`, `OWNERSHIP`, `DEPENDENCY`.
- **Query:** `SELECT * FROM nodes WHERE id IN (WITH RECURSIVE...)`.

---

## 2. Star Pattern (Attribute Mapping & Hub-and-Spoke)

### Technical Specification
- **Structure:** A central "Hub" node connected to multiple peripheral "Spoke" nodes. Peripheral nodes have no direct links between them.
- **Logic:**
    - **Cardinality:** High out-degree for the Hub node.
    - **Semantic Gating:** Spokes usually represent `ATTRIBUTE`, `METADATA`, or `LEAF_LOCATION`.
- **Performance:** Ideal for $O(1)$ attribute retrieval for a given entity.

### Standard Use Cases
- **User Profiles (Social Mapping):** A user node linked to interests, skills, and device IDs.
- **Hub-and-Spoke Logistics:** A central distribution center serving local warehouses.
- **Object Attributes:** Central product node linked to price, weight, and inventory nodes.

### Orchestration Path
- **Intent:** `PROFILING`, `ATTRIBUTE_LOOKUP`, `DISTRIBUTION`.
- **Storage:** Spoke nodes are often stored as JSONB attributes in the `nodes` table for $O(1)$ access without JOINs, unless the spoke itself is a shared entity.

---

## 3. Mesh Pattern (Redundancy & Resilience)

### Technical Specification
- **Structure:** Highly interconnected network where nodes have multiple paths between them.
- **Metrics:**
    - **Connectivity Index:** `PathCount(A, B) > 1` for critical nodes.
    - **Fault Tolerance:** Ability to reroute traffic if a node/edge is removed.
- **Algorithms:** Implementation of **Dijkstra** and **A*** for optimal pathfinding.

### Standard Use Cases
- **Route Optimization (Logistics):** Real-time road networks where closures require dynamic rerouting.
- **Service Mesh (Engineering):** Redundant communication paths between microservices.
- **Social Networks:** Identifying "six degrees of separation" or influence clusters.

### Orchestration Path
- **Intent:** `ROUTING`, `PATHFINDING`, `REDUNDANCY`.
- **Implementation:** Wisdom uses **Personalized PageRank (PPR)** to identify importance within a mesh and BFS for shortest path discovery.

---

## 4. Knowledge Graph (KG) (Semantic Metadata)

### Technical Specification
- **Structure:** Directed multigraph where edges carry semantic labels (Predicates).
- **Schema:** Triplet-based storage: `(Subject) -[Predicate]-> (Object)`.
- **Semantic Authority:** Nodes are linked to a canonical ontology to prevent ambiguity (e.g., "Revenue" vs "Gross Sales").

### Standard Use Cases
- **Fraud Detection:** Mapping relationships between accounts, IPs, and transactions to detect syndicates (e.g., Uber Michelangelo patterns).
- **SRE Insights:** Linking `INCIDENT` nodes to `LOG_PATTERN`, `ONCALL_LADDER`, and `ROOT_CAUSE`.
- **Entity Resolution:** Merging duplicate data from multiple sources into a single ground truth.

### Orchestration Path
- **Intent:** `REASONING`, `MULTI_HOP`, `CORRELATION`.
- **Storage:** Wisdom implements a `links` table with a `predicate` field, enabling complex multi-hop queries that outperform SQL JOINs through recursive graph traversal.

---

## Decision Matrix: Pattern Selection

| Use Case | Recommended Pattern | Primary Advantage | Fallback |
| :--- | :--- | :--- | :--- |
| **Org Hierarchies** | Tree | Simple recursion, no cycles | Mesh |
| **Inventory Attributes** | Star | $O(1)$ metadata retrieval | KG |
| **Traffic/Network Routing** | Mesh | Maximum redundancy | Tree (Spanning Tree) |
| **Abuse/Fraud Analysis** | Knowledge Graph | Multi-hop relational depth | Star |

## Automated Orchestration Logic
Project Wisdom's **Thalamus** component automatically selects the pattern based on the detected intent:
1. **Parser** extracts entities and relations.
2. **Classifier** determines if the query is hierarchical, relational, or path-based.
3. **Router** directs the request to the specialized graph traversal engine (e.g., BFS for Mesh, CTE for Tree).

## Tests and Validation
- **Unit Test (Tree):** Validate cycle detection on deep insertion.
- **Unit Test (Star):** Verify $O(1)$ retrieval of 100+ attributes.
- **Unit Test (Mesh):** Benchmark Dijkstra vs BFS for 5-hop pathfinding.
- **Unit Test (KG):** Validate triple consistency and predicate filtering.
