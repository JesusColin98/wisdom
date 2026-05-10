# Workspace Memory: chat_service
*Agent Instructions: Maintain this file as a high-signal audit trail. Perform surgical updates to minimize context usage.*

## 📋 Task Metadata
*   **Task Objective:** Debug frontend JSON parsing error in Wisdom Portal after IAP deployment.
*   **Target Bug:** b/___
*   **Active CL:** cl/___
*   **Current Phase:** Debugging Frontend
*   **Workspace State:** Fig

## 🛠️ Dynamic Tech Stack & Dependencies
*   **Core Deps:** Go (Wisdom Engine), Python (Chat Service), React (Portal), GCP IAP, GCLB
*   **Key Symbols/Files:** `portal/src/components/GraphView.jsx`, `chat_service/main.py`, `scripts/deploy_all.sh`
*   **Env/Build Flags:** WISDOM_DB_PATH, PORT

## 📂 Touched Artifacts
*   `../wisdom/pkg/api/server.go` (Added /whoami)
*   `../scripts/deploy_all.sh` (Configuring Path-based routing)
*   `deploy_all.sh.new` (Temporary deployment script)
*   `/usr/local/google/home/jesuscolin/brujula/chat_service/.gemini/gemini_memory.md`
*   `/usr/local/google/home/jesuscolin/brujula/chat_service/deploy_all.sh.new`

## 🎯 Task Tracker
*   [x] **1. Phase 1:** Implement native Go MCP in `pkg/mcp`.
*   [x] **2. Phase 2:** Refactor engine to use shared `kernel` and rename to `wisdom-api`.
*   [x] **3. Phase 3:** Verify compilation and unit tests for both binaries.
*   [x] **4. Infrastructure:** Automated GCLB & IAP setup in `deploy_all.sh`.
*   [x] **5. Migration:** Configured local Gemini CLI to use the native Go MCP binary.
*   [ ] **6. Phase 4:** Debug "Unexpected token '<'" JSON error in frontend.

## 📝 Implementation Plan
1. Identify API calls in the portal source code.
2. Verify if API requests are being redirected to IAP login page (returning HTML).
3. Adjust routing or authentication (JWT assertion) if necessary.


## 🧪 Verification Strategy
*   **Browser Inspection:** Check Network tab for 302 redirects to accounts.google.com on API calls.
*   **Logs:** Check Cloud Run and GCLB logs for rejected/redirected requests.

---
## ⏱️ Session Update Log
*Structure: [TIMESTAMP] | PHASE | ACTION -> OUTCOME | NEXT*

*   `2026-05-10 03:30:00` | Refactoring | Created `pkg/kernel` and `pkg/mcp`. | Split binaries.
*   `2026-05-10 03:45:00` | Completed | Refactoring finished. Binaries verified. | Presentation.
*   `2026-05-10 04:00:00` | Debugging | Investigating "Unexpected token '<'" error. | Search API calls in portal.


