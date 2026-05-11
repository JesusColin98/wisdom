# Project Wisdom: Deployment, Security & Roadmap

## 🟢 Infrastructure & Connectivity (COMPLETED)
- [x] **Centralized Deployment Script**: Created `brujula/scripts/deploy_all.sh` to sync Engine, Agent, and Portal.
- [x] **Fix 403 Forbidden**: Implemented OIDC Authentication flow and secure IAM bindings.
- [x] **Secure Access (IAP)**: Configured Global External Load Balancer + IAP. UI now redirects to Google Login.
- [x] **Native Go MCP**: Implemented native Go MCP server in `pkg/mcp`, removing local Python overhead.
- [x] **CLI Integration**: Local Gemini CLI configured to use `wisdom-mcp` directly via `wisdom.db`.

## 🏗️ Phase 2: Neural Substrate & Advanced Coaching (IN PROGRESS)
Goal: Deepen the graph traversal and visual intelligence.

### 🧩 Knowledge Graph (Thalamus)
- [x] **Recursive Prerequisite Discovery**: Implemented `GetPrerequisites` in `HierarchyManager` using recursive CTEs.
- [ ] **Cross-Cloud Sync**: Sync local `wisdom.db` with Cloud Spanner or Firestore for persistence.
- [ ] **Tool Interpretation**: Enhance `Cerebellum` to support more complex JSON-schema for tools.

### 🧠 Agent Multimodality
- [ ] **Vision Loop**: Implement JPEG frame processing in `chat_service` using Gemini 1.5 Pro.
- [ ] **Live Session**: Enable real-time audio/video feedback loop in the Portal.

## 🎨 Phase 3: Frontend UX & Aesthetics (IN PROGRESS)
### 🧪 Portal Enhancements
- [x] **Stratum Visuals**: Updated `GraphView` to color nodes by Entity Class and show Stratum status (HOT/COLD).
- [ ] **Hierarchy Explorer**: Add "Drill Down" / "Zoom Out" using `/cortex/lineage`.
- [x] **Dopamine Glow**: Visual feedback for high-impact knowledge nodes (>0.8 impact).
- [x] **Hallucination Guard**: Added verification UI in `ChatView` with wavy red underlines for ungrounded text.

## 🔍 Findings Log
- **2026-05-11**: **BUILD FIXED**: Resolved go.sum mismatch and fixed compilation errors in pkg/cortex. Consolidatated Cortex methods to delegate to the engine interface. Verified via Docker build and package tests.
- **2026-05-10**: **IAP SUCCESS**: IAP is now enforcing Google SSO. Note: Avoid loading in frames to prevent OAuth redirect blocks.
- **2026-05-10**: **RECURSIVE PREREQS**: Added recursive graph traversal to identify missing deep dependencies.
- **2026-05-10**: **VISUAL ATLAS**: Enhanced `GraphView` with glowing high-impact nodes and stratum labels.
- **2026-05-10**: **MCP NATIVE**: Verified `wisdom-mcp` is functional via stdio and JSON-RPC 2.0.
- **2026-05-10**: **JSON ERROR FIX**: Resolved "Unexpected token '<'" by implementing path-based routing in GCLB and adding a /whoami endpoint to the engine. All services now share the same domain.
