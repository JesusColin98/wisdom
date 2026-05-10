# Workspace Memory: chat_service
*Agent Instructions: Maintain this file as a high-signal audit trail. Perform surgical updates to minimize context usage.*

## 📋 Task Metadata
*   **Task Objective:** Decouple core engine logic from service layers and implement native Go MCP server.
*   **Target Bug:** b/___
*   **Active CL:** cl/___
*   **Current Phase:** Refactoring Completed
*   **Workspace State:** Fig

## 🛠️ Dynamic Tech Stack & Dependencies
*   **Core Deps:** Go (Wisdom Engine), Python (Chat Service), MCP Protocol (JSON-RPC), Traefik Yaegi (Interpreted Go Tools)
*   **Key Symbols/Files:** `pkg/kernel/kernel.go`, `pkg/mcp/server.go`, `cmd/wisdom-api`, `cmd/wisdom-mcp`
*   **Env/Build Flags:** WISDOM_DB_PATH, PORT

## 📂 Touched Artifacts
*   `../wisdom/pkg/kernel/kernel.go`
*   `../wisdom/pkg/mcp/protocol.go`
*   `../wisdom/pkg/mcp/server.go`
*   `../wisdom/pkg/mcp/server_test.go`
*   `../wisdom/cmd/wisdom-mcp/main.go`
*   `../wisdom/cmd/api-server/main.go`
*   `../wisdom/Dockerfile`
*   `../TODO.md`
*   `/usr/local/google/home/jesuscolin/brujula/chat_service/.gemini/gemini_memory.md`

## 🎯 Task Tracker
*   [x] **1. Phase 1:** Implement native Go MCP in `pkg/mcp`.
*   [x] **2. Phase 2:** Refactor engine to use shared `kernel` and rename to `wisdom-api`.
*   [x] **3. Phase 3:** Verify compilation and unit tests for both binaries.
*   [x] **4. Infrastructure:** Automated GCLB & IAP setup in `deploy_all.sh`.
*   [x] **5. Migration:** Configured local Gemini CLI to use the native Go MCP binary.

## 📝 Implementation Plan
1. Create `pkg/kernel` to encapsulate shared bootstrapping logic. (DONE)
2. Implement MCP protocol in `pkg/mcp` supporting `recall_wisdom`, `calculate_risk`, and legacy `chat`. (DONE)
3. Split entry points into `cmd/wisdom-api` and `cmd/wisdom-mcp`. (DONE)
4. Update Dockerfile to target `wisdom-api`. (DONE)
5. Update `deploy_all.sh` to configure Global Load Balancer and IAP. (DONE)
6. Switch local MCP config (`mcp_servers.json`) to native Go. (DONE)


## 🧪 Verification Strategy
*   **Automated Tests:** `go test ./pkg/mcp/...` passed.
*   **Build Verification:** `go build` successful for both `wisdom-api` and `wisdom-mcp`.

---
## ⏱️ Session Update Log
*Structure: [TIMESTAMP] | PHASE | ACTION -> OUTCOME | NEXT*

*   `2026-05-10 03:30:00` | Refactoring | Created `pkg/kernel` and `pkg/mcp`. | Split binaries.
*   `2026-05-10 03:45:00` | Completed | Refactoring finished. Binaries verified. | Presentation.


