# Portal Specification: Mission Control

The Wisdom Portal is the primary observability and **conversational interaction** layer for the user. It is a React + Vite application that communicates with the backend via REST and WebSockets.

> **See also:** `FRONTEND_SPEC.md` for the complete view inventory, design system, navigation structure, Obsidian PKM onboarding flow, and mobile responsiveness specification.

## 1. Core Views

### 1.0 Chat (Primary — Default Landing View)
*   **Purpose:** Conversational entry point to the entire Cognitive Runtime. The user talks to the ADK Router via this interface.
*   **Data Source:** `WSS /v1/chat/stream/{session_id}` (via `chat_service`).
*   **Content:** Streaming chat UI with Agent Identity Indicator, Inline Action Cards (Save note, Create Anki card, Search vault), and voice input.
*   **See:** `CHATBOT_UI_SPEC.md` for the full specification.

### 1.1 Dashboard (System Health)
*   **Purpose:** Monitor backend microservices.
*   **Data Source:** `GET /v1/system/health` (Polled or WebSocket).
*   **Content:** Status indicators for `Thalamus`, `Cortex`, `Researcher`, `Integrations`, `chat_service`, and `Vertex AI Memory Bank`.

### 1.2 Researcher Monitor (Scrape Queue)
*   **Purpose:** Visualize autonomous data gathering.
*   **Data Source:** `WS /v1/researcher/stream`.
*   **Content:** 
    *   Currently crawling URL/PDF.
    *   Progress bar (0-100%).
    *   Error log for rate-limited sites.

### 1.3 Knowledge Staging Area
*   **Purpose:** View content pending sync to MCP.
*   **Data Source:** `GET /v1/integrations/queue`.
*   **Content:** List of Obsidian notes and Anki cards that failed to sync because local apps were closed. Includes a "Retry All" button.

### 1.4 Wisdom Study UI (Proprietary SRS)
*   **Purpose:** Advanced AI-driven study sessions.
*   **Data Source:** `GET /v1/metabolism/due`.
*   **Content:** Flashcard interface with AI-augmented feedback explaining *why* an answer was correct or incorrect.

## 2. WebSocket Event Schemas (Gap R6 Fix)

### Event: `SCRAPE_PROGRESS`
```json
{
  "event": "SCRAPE_PROGRESS",
  "payload": {
    "job_id": "uuid-123",
    "source": "https://react.dev/hooks",
    "status": "CRAWLING",
    "progress": 45,
    "eta_seconds": 12
  }
}
```

### Event: `MEMORY_CONSOLIDATION`
```json
{
  "event": "MEMORY_CONSOLIDATION",
  "payload": {
    "session_id": "uuid-456",
    "facts_extracted": 3,
    "topics": ["CHESS_OPENINGS"]
  }
}
```

## 3. API Contract
The Portal interacts with two backends:
*   **`Thalamus`:** System health, researcher stream, staging area, SRS study. Auth: JWT via Cookie.
*   **`chat_service`:** All conversational messages and session management. Auth: JWT Bearer token.
*   **Transport:** HTTPS/REST for CRUD; WebSockets for real-time events (streaming chat + scrape progress).

## 4. Extended Specifications
*   **Full view inventory + design system:** See `FRONTEND_SPEC.md`.
*   **Chat session lifecycle + message schemas:** See `CHATBOT_UI_SPEC.md`.
*   **Markdown → Anki conversion:** See `MD_TO_ANKI_PIPELINE.md`.
