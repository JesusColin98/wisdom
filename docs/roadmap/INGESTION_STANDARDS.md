# Ingestion & Interoperability: Ecosystem Standards

Wisdom adopts a "Multi-Source, Unified Format" strategy to ensure modularity with tools like Obsidian and Logseq while enabling complex ingestion from the web and chat.

## 1. Supported Ecosystems (Interoperability)

| Tool | Interop Level | Wisdom Strategy |
| :--- | :--- | :--- |
| **Obsidian** | High (Native) | Directly index Obsidian vaults. Respect [[Links]] and YAML metadata. |
| **Logseq** | Medium (Format) | Parse block-level properties (`key:: value`) and bullet hierarchies. |
| **Anytype** | Low (Export) | Import via Markdown export. Support Anytype-style Relations mapping. |
| **Tana** | Low (API/Paste) | Support "Tana Paste" formatted input for structured list ingestion. |

## 2. Ingestion Pipelines

### A. Local Ingestion (The Vault)
- **Mechanism**: File-system watcher on configured directories.
- **Protocol**: Converts `.md`, `.canvas`, and `.pdf` into `Cortex` nodes.
- **Modularity**: Users can "Mount" an Obsidian vault as a `Thought Space` (Namespace) in Wisdom.

### B. Internet Ingestion (The Researcher)
- **Mechanism**: Stream research results as Markdown signals.
- **Metadata Standard**: Every web signal must include:
    - `source_url`: URL of the page.
    - `ingested_at`: ISO timestamp.
    - `content_hash`: For de-duplication.
    - `tags`: Identified topics.

### C. Conversational Ingestion (The REM Cycle)
- **Mechanism**: During `/chat`, ephemeral nodes are created in the `Hippocampus`.
- **Promotion**: During the REM cycle, the `REMService` analyzes these nodes. If they contain high-signal entities (@, #) or significant facts, they are promoted to the permanent `Cortex`.

## 3. MCP Integration Strategy
Wisdom will expose an **MCP Server** that allows agents (Gemini, Claude) to:
1.  **Read**: `recall(query)` - Search the unified graph.
2.  **Write**: `memorize(content)` - Ingest new facts into the specific user's namespace.
3.  **Learn**: `plan_learning(topic)` - Trigger the `LearningEngine` to output a structured path.

## 4. TTL (Time-To-Live) Strategy for Signals
To prevent "Graph Rot," non-immutable nodes (Signals) implement TTL:
- **Low Impact**: 7 days.
- **Medium Impact**: 30 days.
- **High Impact / Fact**: Permanent.
- **Auto-Renewal**: Every time a node is successfully recalled or upvoted, its TTL is reset.
