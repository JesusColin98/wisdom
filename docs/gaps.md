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

## 4. Current Architectural Gaps (In Progress)

### Redundant MCP Tool Definitions
- **Issue**: `scripts/tools/wisdom_bridge.py` and `chat_service/main.py` both define tool sets for Wisdom.
- **Risk**: Inconsistent tool behavior across different access methods.
- **Action**: Consolidate into a single configuration-driven tool registry.

### SQLite over GCS FUSE (Concurrency)
- **Issue**: High-latency writes and lock contention in multi-instance environments.
- **Risk**: `database is locked` errors during heavy REM cycles or concurrent user updates.
- **Action**: Monitor and prepare migration to Firestore/Cloud Spanner if horizontal scaling is required.

### Authentication for WebSockets
- **Issue**: `wisdom-chat` lacks explicit token verification for the incoming WebSocket connection from the Portal.
- **Risk**: Unauthorized access to the Gemini Live session if the URL is discovered.
- **Action**: Integrate Firebase Auth SDK in the Python proxy.

### Telemetry & Spans
- **Issue**: No cross-service tracing between the Python proxy and Go backend.
- **Action**: Implement OpenTelemetry tracing to measure latency in the multimodal loop.
