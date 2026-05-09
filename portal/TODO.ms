# Project Gaps & TODOs

## Logic & Functionality
- [x] **Real Data Integration**: Replaced mock data in `GraphView.jsx` and `MetabolismView.jsx` with actual API calls to `/graph` and `/metabolism`.
- [x] **Responsiveness**: `GraphView` now listens for window resize events to update its dimensions.
- [x] **State Management**: Implemented `WisdomContext` using React Context API to manage global state (view, rigor, namespace, user).
- [x] **Loading/Error States**: Added global loading overlay and error toast notifications synced with the context.
- [x] **Port Standardization**: Standardized all services to listen on port 8080 for Cloud Run defaults.
- [x] **Chat Microservice**: Extracted agent logic into a separate FastAPI service to handle LLM orchestration.
- [x] **Real-time Connectivity**: Migrated chat to WebSockets for sub-100ms conversational latency.
- [x] **Google Search Integration**: Agent now has autonomous access to search for grounding.
- [x] **Gemini Live Prototype**: Implemented real-time video frame streaming from the portal to the Chat Agent.
- [ ] **Functional Wiring**: Point Portal `WS_BASE` and `API_BASE` to deployed `wisdom-chat-agent` URL instead of localhost.
- [ ] **Environment Parity**: Set `WISDOM_ENGINE_URL` in `wisdom-chat-agent` to the deployed `wisdom-mcp` URL.
- [ ] **Authentication**: Ensure OIDC tokens are correctly passed between Portal -> Chat Agent -> Wisdom Engine.

## Current Deployment & Functional Issues
- [ ] **MCP Transition**: Remove `nexusstate` from `~/.gemini/mcp_servers.json` and add `wisdom` pointing to `wisdom_bridge.py`.
- [ ] **Bridge Configuration**: Update `wisdom_bridge.py` `SERVICE_URL` with the newly deployed `wisdom-mcp` endpoint.
- [ ] **Deployment Permissions**: Resolve `iam.serviceaccounts.actAs` and Gaia ID issues for `cloud-run-build-sa`. (May require manual fix via `go/iam`).
- [ ] **Service Cleanup**: Previous `nexusstate-mcp` and `nexusstate-portal` deleted; ensure new services follow `wisdom-*` naming convention for clarity.
- [ ] **Gemini API Key**: Securely inject `GEMINI_API_KEY` into `wisdom-chat-agent` environment.

## Roadmap & Research
- [x] **Associative Memory Integration**: Leverage patterns from `//depot/google3/wireless/android/keystone/mcp/amem/skills/associative-memory/SKILL.md` to enhance Neural Atlas recall.
- [ ] **Multimodal Reasoning**: Enhance `chat_service` to process incoming JPEG frames using Gemini 1.5 Pro's multimodal capabilities.
- [ ] **Voice Synthesis**: Integrate text-to-speech for a full "Live" experience.
- [ ] **Edge-Side Reasoning**: Investigate using local Chrome AI (BlueDog sandbox patterns) for low-latency node classification.

