# Wisdom Microservices Evolution: The Cognitive Runtime

Wisdom is evolving from a monolithic memory graph into a distributed **Cognitive Runtime**. This architecture ensures that deterministic data retrieval, structured learning, and personalized memory are handled by specialized, decoupled services.

## Core Vision: Everything is a Service
We transition from internal Go packages to independent services communicating via **gRPC** as the mandatory internal transport layer for maximum performance and low-latency neural loops.

### Protocol Standards
- **Internal**: **gRPC (Protobuf)**. Chosen for speed, binary efficiency, and strong typing.
- **External**: **REST / MCP (Model Context Protocol)**. For interoperability with other AI agents and web clients.

### 1. Subsystem: `Wisdom-Researcher` (Deterministic Investigation)
Deterministic engine for factual data gathering.
- **Responsibility**: Scrape, search, and download raw knowledge without LLM bias.
- **Sub-modules**:
  - `Book-Vault`: Integration with Anna's Archive/Z-Lib (DDL) for deep-dive PDFs.
  - `Blog-Crawler`: RSS/Atom crawler for specialized community knowledge.
  - `News-Stream`: Real-time monitoring of current events/trends.
- **Output**: Cleaned, tagged Markdown with Obsidian-style metadata.

### 2. Subsystem: `Wisdom-Curriculum` (Learning Path Orchestrator)
The "Teacher" that organizes information into logical hierarchies.
- **Responsibility**: Map "Big Topics" (Science, Philo, Chess) into "Sub-topics" and "Atomic Concepts."
- **Logic**: Implements a standard taxonomy while allowing dynamic expansion via discovery.
- **Levels**: Defines Beginner, Intermediate, and Advanced tiers for every concept.

### 3. Subsystem: `Wisdom-Trace` (Personalized Mastery)
The user-centric tracker.
- **Responsibility**: Map the Knowledge Graph specifically to a `user_id`.
- **Metrics**: Tracks `MasteryScore`, `Fragility` (forgetting curve), and `StruggleDensity`.
- **Personalization**: Injects user-specific "weakness nodes" into any learning path generated.

### 4. Subsystem: `Wisdom-Entity-Dictionary` (The Ontology)
The "Cerebro" sub-module for recognition.
- **Responsibility**: Recognize People (@), Tags (#), and Systems. Maintain attributes (e.g., @Jesus: Role=SRE, Level=Expert).
- **Standards**: Uses standard Markdown symbols for entity mapping during ingestion.

### 5. Subsystem: `Cortex-Substrate` (Memory as a Service)
The storage and retrieval backbone.
- **Responsibility**: Unified access to facts (SQL), relationships (Graph), and semantics (Vector).
- **Interface**: Decoupled "Introducer" (Writer) and "Extractor" (Reader) services.
- **Abstraction**: High-level API to query "Hechos" (Immutable truths) vs "Signals" (Changeable info).

## Next Steps for Debate
1.  **Transport Protocol**: Standardized on **gRPC** for request/response and **NATS JetStream** for events.
2.  **Entity Standards**: Finalize symbols: `@` for people, `#` for topics, `[[ ]]` for internal links, `!` for high-confidence facts.
3.  **Database Strategy**: **Google Cloud Spanner** for global consistency (The 'Enciclopedia') and **GCS** for large resource blobs (The 'Vault').
