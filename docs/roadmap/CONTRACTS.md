# Contracts & Communication Boundaries

Wisdom's architecture relies on strict separation of concerns and a standardized data flow.

## 1. Authoritative Data Flow (Gap R3 Fix)
To prevent integration spaghetti, all creation and review cycles follow this single authoritative sequence:

### Knowledge Creation Path (AI-Generated)
`Expert Agent` -> `Integrations Service` -> `MCP Server (Obsidian/Anki/Logseq)`

### Mastery Feedback Path
`Anki Desktop` -> `AnkiConnect` -> `MCP Server` -> `Integrations Service (Polling)` -> `Trace Service` -> `Cortex DB`

### Markdown → Anki Conversion Path (User-Triggered)
`Portal UI` -> `Integrations Service (md_to_anki)` -> `obsidian-mcp-server (read note)` -> `Integrations (parse)` -> `Anki Export Queue` -> `AnkiConnect`
See `MD_TO_ANKI_PIPELINE.md` for the full spec.

### Note Polish Path (User-Triggered)
`Portal UI` -> `Integrations Service (polish_note)` -> `Gemini API` -> `Diff Response` -> `Portal UI (diff view)` -> `obsidian-mcp-server (write accepted hunks)`
See `INGESTION_STANDARDS.md §5` for the Gemini audit contract.

## 2. Internal Communication: gRPC & Pub/Sub
*   **Synchronous:** Internal microservices communicate via **gRPC (Protobuf)** with mTLS (e.g., Thalamus to Cortex).
*   **Asynchronous:** We use **GCP Pub/Sub** for event-driven messaging (e.g., triggering a background scrape job in Researcher, or broadcasting a memory consolidation event). This perfectly aligns with Cloud Run.

## 3. External & UX Communication: MCP
Interactions with the user's local tools (Obsidian, Anki) are mediated by **MCP**. 

## 4. Observability Communication: WebSockets
The `portal/` receives real-time system events (originating from Pub/Sub topics) via WebSockets routed through Thalamus, as defined in `PORTAL_SPEC.md`.

## 5. Cognitive Memory: Vertex AI API
Communication with the Memory Bank uses the ADK, scoped by `user_id` and `agent_name`.
