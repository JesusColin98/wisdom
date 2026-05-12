# Project Wisdom: Deployment, Security & Roadmap

## 🟢 Infrastructure & Architecture (COMPLETED)
- [x] **Unified Go Engine**: Consolidated React Portal into Go's `http.FileServer`.
- [x] **Decoupled Senses**: Separated `chat_service` (Python/WebSocket) for Live API streaming into its own container build via `cloudbuild.yaml`.
- [x] **Native Go MCP**: Implemented native Go MCP server in `pkg/mcp`.
- [x] **Secure Access (IAP)**: Configured Global External Load Balancer + IAP.

## 🏗️ Phase 2: Neural Substrate & Advanced Coaching (IN PROGRESS)
Goal: Deepen the graph traversal, visual intelligence, and cloud durability.

### 🧩 Persistence & Architecture
- [x] **Cloud Run Terraform**: Automated the deployment of `wisdom-unified` and `wisdom-chat` via Terraform in the `terraform/` directory. Included GCS FUSE setup for stateful SQLite.
- [ ] **Cross-Cloud Sync / Concurrency**: Evaluate SQLite concurrency locks over GCS FUSE. If scaling horizontally creates write contention, migrate Core Logic to Cloud Spanner or Firestore.

### 🧠 Agent Multimodality
- [x] **Live Audio UI**: Integrated real-time audio/video feedback loop in `ChatView.jsx`.
- [x] **Vision Processing**: Frames sent from Portal are correctly routed to the Gemini 2.0 pipeline in `wisdom-chat`.
- [ ] **WebSocket Auth**: Implement Firebase Auth verification for the voice proxy.

## 🎨 Phase 3: Frontend UX & Aesthetics (IN PROGRESS)
### 🧪 Portal Enhancements
- [x] **Hierarchy Explorer**: Implemented "Drill Down" / "Zoom Out" in `GraphView` using `/cortex/lineage`.
- [x] **Stratum Visuals**: Updated `GraphView` with glowing high-impact nodes and stratum labels.
- [x] **Hallucination Guard**: Added verification UI in `ChatView` for ungrounded text.
- [ ] **Telemetry Dashboard**: Add a small diagnostic panel to the portal to monitor WebSocket latency and buffer health.
