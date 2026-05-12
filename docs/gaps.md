# Analysis of Unused and Underutilized Components (Gaps)

During the deep architectural analysis of the project, several components were identified that are either disjointed from the unified production build or entirely redundant.

## 1. Underutilized/Detached Components

### `chat_service/` (The Multimodal Python Agent) -> **RESOLVED (DECOUPLED)**
- **Current State:** Contains a Python-based WebSocket proxy connecting to the **Gemini 2.0 Live API** (`wss://generativelanguage.googleapis.com/...`). It handles real-time audio (PCM) and streaming text for the frontend.
- **Why it was a Gap:** The root unified `Dockerfile` only built the Go backend.
- **Resolution:** Decoupled into its own independent Cloud Run build via `cloudbuild.yaml` as `wisdom-chat:latest`.

## 2. Unused Configuration Files

### `docker-compose.yml` -> **RESOLVED**
- **Current State:** Defined a multi-container setup with `wisdom-engine`, `wisdom-chat`, and `wisdom-portal`.
- **Resolution:** Updated to accurately reflect the 2-container architecture (`wisdom-engine` + `wisdom-chat`), removing the redundant standalone `portal` service.

## 3. Modularizing Wisdom (Memory as a Service)
- **Current State:** Wisdom tightly integrates graph processing with chat in its UI. 
- **The Gap:** To offer Wisdom as a pure "Memory-as-a-Service", the Go MCP Server (`wisdom-mcp`) should be the primary product surface for external LLMs. The frontend (`portal`) should serve as an agnostic visualizer of the `wisdom.db`.
- **Action:** Expose the MCP logic robustly so external tools (learning English, math, etc.) can hook into Wisdom's storage layer dynamically, and ensure the UI can visualize any arbitrary namespace.

---
## ✅ Resolved / Deleted Gaps

### `cerebellum_service/` -> **DELETED**
- **Issue:** Redundant Python microservice referencing Graph Mamba and GAT models, superseded by `pkg/cerebellum/` in Go.
- **Resolution:** Directory fully removed from the repository.

### `wisdom_bridge.py` -> **MOVED TO SCRIPTS**
- **Issue:** Obsolete Python script bridging Gemini CLI.
- **Resolution:** Moved to `scripts/tools/` for archiving, removing it from the execution path.
