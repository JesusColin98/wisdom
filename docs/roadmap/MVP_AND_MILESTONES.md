# MVP & Milestones

## Success Metrics
*   **Obsidian:** 100% parsing success.
*   **Anki:** Sync latency < 15 mins.
*   **Performance:** End-to-end latency < 1.2s (including voice processing).

## Phased Roadmap & Deliverables

### Phase 1: Substrate & Security (DONE)
*   **Status**: COMPLETED. Go microservices decoupled, gRPC mesh established with mTLS.

### Phase 2: The MCP Bridge (DONE)
*   **Status**: COMPLETED. Integrations service handles Obsidian/Anki synchronization.

### Phase 3: Cognitive Routing & MoE (DONE)
*   **Status**: COMPLETED. ADK Router implemented with Gemini Flash. Baseline experts (Chess, Finance, Tech, Language) operational.

### Phase 4: Mission Control Portal, Chat UI & Voice (CURRENT)
*   **Task A**: React Portal — System Health View + Google STT Integration in Thalamus.
*   **Task B**: Chat UI — WebSocket streaming, Agent Identity Indicator, Inline Action Cards.
*   **Task C**: Dynamic Scaling — **Expert Registry UI** allows adding new domains without code. (COMPLETED)
*   **Task D**: Autonomous Research — Experts can trigger the `Researcher` service tool-wise. (COMPLETED)
*   **Deliverable (DoD)**: Register a domain via UI → Ask a question → Expert uses Researcher tool to fetch context → Result saved to Obsidian.

### Phase 5: Markdown → Anki Pipeline (Weeks 9-10)
*   **Task:** `md_to_anki` endpoint in `Integrations` service + Anki Export Review Panel (View 4) + "Convert to Anki" button in Note Editor.
*   **Deliverable (DoD):** User opens an existing Obsidian note in the Portal, clicks "Convert to Anki cards", reviews 3 generated cards, clicks "Send to Anki" → cards appear in Anki desktop within 15 seconds. Running the same conversion twice produces no duplicate cards.

### Phase 6: Post-MVP (Hardening)
*   **Task:** Implement local `pgvector` offline fallback for Vertex AI Memory Bank outages.

## Definition of Done (MVP)
User can speak a concept -> Researcher fetches context -> Python ADK Expert generates Obsidian/Anki content via Integrations -> Review stats sync back to Wisdom.
