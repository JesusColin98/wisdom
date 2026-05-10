# Project Wisdom: Deployment, Security & Roadmap

## 🟢 Infrastructure & Connectivity (COMPLETED)
- [x] **Centralized Deployment Script**: Created `brujula/scripts/deploy_all.sh` to sync Engine, Agent, and Portal.
- [x] **Fix 403 Forbidden**: Implemented OIDC Authentication flow and secure IAM bindings.
- [x] **Secure Access (IAP)**: Configured Global External Load Balancer + IAP. UI now redirects to Google Login.
- [x] **Native Go MCP**: Implemented native Go MCP server in `pkg/mcp`, removing local Python overhead.
- [x] **CLI Integration**: Local Gemini CLI configured to use `wisdom-mcp` directly via `wisdom.db`.

## 🏗️ Phase 2: Neural Substrate & Advanced Coaching
Goal: Deepen the graph traversal and visual intelligence.

### 🧩 Knowledge Graph (Thalamus)
- [ ] **Recursive Prerequisite Discovery**: Logic to traverse deep dependency chains in the graph.
- [ ] **Cross-Cloud Sync**: Sync local `wisdom.db` with Cloud Spanner or Firestore for persistence.
- [ ] **Tool Interpretation**: Enhance `Cerebellum` to support more complex JSON-schema for tools.

### 🧠 Agent Multimodality
- [ ] **Vision Loop**: Implement JPEG frame processing in `chat_service` using Gemini 1.5 Pro.
- [ ] **Live Session**: Enable real-time audio/video feedback loop in the Portal.

## 🎨 Phase 3: Frontend UX & Aesthetics
### 🧪 Portal Enhancements
- [ ] **Stratum Visuals**: Update `GraphView` to color nodes by Stratum (`HOT` vs `COLD`).
- [ ] **Hierarchy Explorer**: Add "Drill Down" / "Zoom Out" using `/cortex/lineage`.
- [ ] **Dopamine Glow**: Visual feedback for high-impact knowledge nodes.
- [ ] **Hallucination Guard**: Highlight ungrounded text in `ChatView` with red underlines.

## 🔍 Findings Log
- **2026-05-10**: **IAP SUCCESS**: IAP is now enforcing Google SSO on the Portal. No more 403 on LB URL.
- **2026-05-10**: **MCP NATIVE**: Verified `wisdom-mcp` is functional via stdio and JSON-RPC 2.0.
- **2026-05-10**: **BOOTSTRAP**: Created `pkg/kernel` to unify initialization between API and MCP.
- **2026-05-09**: **SECURITY**: Switched to per-identity bindings (`roles/run.invoker`) for SAs and Users.
