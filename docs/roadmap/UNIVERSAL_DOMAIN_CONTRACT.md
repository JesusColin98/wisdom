# Universal Domain Contract

To ensure Wisdom remains highly modular and decoupled, external applications and domain experts must adhere to this strict Universal Domain Contract.

## 1. The Expert Agent Profile (ADK Schema)
External apps plug into Wisdom by registering a profile with the `Thalamus` gateway:
*   **`agent_name`**: Unique identifier (e.g., `"Finance_Expert"`).
*   **`intents`**: Array of natural language triggers.
*   **`custom_topics`**: Semantic boundaries for the Vertex AI Memory Bank.

## 2. Ingestion & Interaction
*   **Decoupled Ingestion:** Agents **NEVER** call MCP servers directly. They send standardized payloads to the `Integrations` service.
*   **Voice Interface:** The agent must process text transcripts provided by the `Routing` layer and format responses for TTS compatibility (no complex tables).

## 3. Registration & Lifecycle (Gap R1 Fix)
Wisdom maintains a `domains.json` registry file within the `Thalamus` service. Registration happens through two methods:

### 3.1 Static Registration
A JSON configuration file loaded at boot time.
```json
{
  "domains": [
    {
      "agent_name": "Chess_Expert",
      "intents": ["chess opening", "tactical analysis"],
      "custom_topics": ["ELO", "OPENINGS"],
      "grpc_endpoint": "localhost:50051"
    }
  ]
}
```

### 3.2 Dynamic Registration API
Thalamus exposes a protected endpoint for runtime registration:
*   **Endpoint:** `POST /v1/domains/register`
*   **Validation:** Thalamus checks for `agent_name` collisions. If two domains claim the same `intent`, Thalamus uses a **Priority Score** or prompts the user for clarification.
*   **Schema:** Registration must include the gRPC endpoint where the Expert Agent is hosted.

## 4. Ingestion Payload Contract (Gap R3 Fix)
Every domain expert uses this authoritative flow for knowledge creation:
`Expert Agent -> Integrations Service -> MCP Server`

### 4.1 Knowledge Graph Payload (Obsidian/Logseq)
```json
{
  "action": "create_note",
  "domain": "technology",
  "metadata": {
    "title": "React Server Components",
    "tags": ["#tech/react"],
    "aliases": ["RSC"]
  },
  "content": "...",
  "relationships": ["[[React Hooks]]"]
}
```
The `Integrations` service is responsible for determining whether to call `obsidian-mcp-server` or `mcp-logseq` based on user settings.
