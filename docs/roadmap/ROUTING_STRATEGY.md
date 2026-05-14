# Routing Strategy & Voice Integration

## 1. The Three-Stage Pipeline (Gap R5 Fix)

### Stage 0: Voice Preprocessing (The Ear)
*   **Engine:** `Google Cloud Speech-to-Text V2`.
*   **Responsibility:** Convert audio stream to text.
*   **Middleware:** A dedicated Go middleware in the `Thalamus` layer. 
*   **Sanitization:** Normalizes common spoken errors in programming syntax (e.g., converting "open bracket" to `{`) before passing to the ADK Router.

### Stage 1: Thalamus (The Deterministic Gateway)
*   Pure Go high-performance gateway.
*   Handles JWT Auth, rate limiting, and mTLS internal routing.

### Stage 2: ADK Router (The Cognitive Classifier)
*   **Technology:** Python Microservice. (Leveraging Python's first-class support for AI/Vertex SDKs while preserving Go for the factual backend).
*   LLM-powered (Gemini Flash).
*   Registers domains from `UNIVERSAL_DOMAIN_CONTRACT.md`.
*   Routes based on intent to specific Experts.

## 2. Authentication Model
*   **External:** Bearer JWT in the `Authorization` header.
*   **Internal:** Microservices use SPIFFE/Spire for mTLS identities.
*   **Memory Bank:** Authenticated via `service-account` credentials with Vertex AI scope.

## 3. Offline Fallback (Post-MVP)
Because Vertex AI Memory Bank is a managed cloud service, we must handle network partitions or service outages in the future.
1.  **Primary Path (MVP):** Expert Agent queries Vertex AI Memory Bank.
2.  **Fallback Path (Post-MVP):** If Vertex AI times out or returns 503, the ADK Router will fall back to querying a local `Cortex` database via `pgvector`.
3.  **Voice (MVP):** If Google STT is unavailable, the voice UI is disabled and the Portal prompts the user to switch to text.
