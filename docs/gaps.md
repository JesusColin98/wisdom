# Analysis of Unused and Underutilized Components (Gaps)

During the deep architectural analysis of the project, several components were identified that are either disjointed from the unified production build or entirely redundant.

## 1. Underutilized/Detached Components

### `chat_service/` (The Multimodal Python Agent)
- **Current State:** Contains a Python-based WebSocket proxy connecting to the **Gemini 2.0 Live API** (`wss://generativelanguage.googleapis.com/...`). It handles real-time audio (PCM) and streaming text for the frontend.
- **Why it's a Gap:** The root unified `Dockerfile` only builds the Go backend (`wisdom_engine`) and the React frontend. It completely ignores `chat_service/`. Consequently, if deployed using the unified Docker image, the voice/live capabilities in the UI will fail to connect.
- **Action Required:** Decide whether to (A) port the Gemini 2.0 Live API proxy logic directly into the Go backend's `pkg/api/websocket.go` to maintain a single deployment artifact, or (B) update the `cloudbuild.yaml` and deployment architecture to deploy `chat_service` as a sidecar/separate Cloud Run service so the UI can use the Live Voice features.

### `cerebellum_service/`
- **Current State:** Contains a placeholder Python FastAPI server referencing Graph Mamba and GAT models for "Tier 2" graph reasoning.
- **Why it's a Gap:** `pkg/cerebellum/` in Go has natively taken over the core tool orchestration and schemas. `cerebellum_service` is unused and unlinked.
- **Action:** Safely delete the `cerebellum_service` directory unless Tier 2 Graph Mamba testing is actively happening in Python.

### `wisdom_bridge.py`
- **Current State:** A Python-based script designed to bridge Gemini CLI to the Wisdom Ecosystem over JSON-RPC (MCP).
- **Why it's a Gap:** The documentation explicitly states: "Native Go MCP: Implemented native Go MCP server in pkg/mcp, removing local Python overhead." There is already a `wisdom-mcp` binary built from Go.
- **Action:** Safely delete `wisdom_bridge.py`.

## 2. Unused Configuration Files

### `docker-compose.yml`
- **Current State:** Defines a multi-container setup with `wisdom-engine`, `wisdom-chat` (Python), and `wisdom-portal`.
- **Why it's a Gap:** The project has pivoted to a unified Go binary serving the static UI. The `docker-compose.yml` reflects the older microservice architecture and might confuse developers regarding the production deployment strategy.
- **Action:** Update `docker-compose.yml` to reflect the desired architecture (e.g., whether `chat_service` should remain separate or everything unified).

## 3. Modularizing Wisdom (Memory as a Service)
- **Current State:** Wisdom tightly integrates graph processing with chat in its UI. 
- **The Gap:** To offer Wisdom as a pure "Memory-as-a-Service", the Go MCP Server (`wisdom-mcp`) should be the primary product surface for external LLMs. The frontend (`portal`) should serve as an agnostic visualizer of the `wisdom.db`.
- **Action:** Expose the MCP logic robustly so external tools (learning English, math, etc.) can hook into Wisdom's storage layer dynamically, and ensure the UI can visualize any arbitrary namespace.
