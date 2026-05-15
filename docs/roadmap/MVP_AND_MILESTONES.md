# MVP & Milestones

## Success Metrics
*   **Obsidian:** 100% parsing success.
*   **Anki:** Sync latency < 15 mins.
*   **Performance:** End-to-end latency < 1.2s (including voice processing).

## Phased Roadmap & Deliverables

### Phase 1: Substrate & Security (Weeks 1-2)
*   **Task:** Break the Go monolith into independent services. Establish gRPC mesh with mTLS + JWT implementation. Set up GCP Pub/Sub for async events.
*   **Deliverable (DoD):** A gRPC "Ping/Echo" test between `Thalamus` and `Cortex` succeeds with mutual TLS encryption verified in logs.

### Phase 2: The MCP Bridge (Weeks 3-4)
*   **Task:** Build `Integrations` Go microservice + Local MCP deployment.
*   **Deliverable (DoD):** A POST request to `Integrations` results in a valid Markdown file appearing in a local folder and a flashcard appearing in Anki desktop.

### Phase 3: Cognitive Routing & MoE (Weeks 5-6)
*   **Task:** Build the ADK Router (Python Microservice) + Domain Registration + Vertex Memory Bank integration.
*   **Deliverable (DoD):** Sending the prompt "How are my dividends?" routes to `Finance_Expert` and correctly retrieves `DIVIDEND_GOALS` from Memory Bank with < 1s classification latency.

### Phase 4: Mission Control Portal, Chat UI & Voice (Weeks 7-8)
*   **Task A:** React Portal — System Health View + Google STT Integration in Thalamus.
*   **Task B:** Chat UI (View 1) — WebSocket streaming, Agent Identity Indicator, Inline Action Cards. See `CHATBOT_UI_SPEC.md`.
*   **Task C:** Note Editor (View 2) — YAML wizard, LIFT linter, "Polish with Gemini" diff flow, PARA destination selector. See `FRONTEND_SPEC.md`.
*   **Task D:** Vault Health Dashboard (View 3) — Orphan notes list, LIFT metrics, D3 graph snapshot.
*   **Deliverable (DoD):** User speaks "Define React Hooks" → Chat routes to `Tech_Expert` → Inline Action Card appears → User clicks "Save" → note appears in Obsidian `00_Inbox/`. Portal shows audio waveform and "Scraping Status".

### Phase 5: Markdown → Anki Pipeline (Weeks 9-10)
*   **Task:** `md_to_anki` endpoint in `Integrations` service + Anki Export Review Panel (View 4) + "Convert to Anki" button in Note Editor.
*   **Deliverable (DoD):** User opens an existing Obsidian note in the Portal, clicks "Convert to Anki cards", reviews 3 generated cards, clicks "Send to Anki" → cards appear in Anki desktop within 15 seconds. Running the same conversion twice produces no duplicate cards.

### Phase 6: Post-MVP (Hardening)
*   **Task:** Implement local `pgvector` offline fallback for Vertex AI Memory Bank outages.

## Definition of Done (MVP)
User can speak a concept -> Researcher fetches context -> Python ADK Expert generates Obsidian/Anki content via Integrations -> Review stats sync back to Wisdom.
