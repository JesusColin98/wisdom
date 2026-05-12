# Technical Implementation Details: Project Wisdom v2.0

## 🧠 Cognitive Substrate (Cortex)

### Spaced Repetition (SM-2 Algorithm)
- **Schema:** Added `repetition_count`, `easiness_factor`, and `next_review_at` to the `nodes` table.
- **Logic:** Implemented the **SM-2 algorithm** in `pkg/thalamus/scheduler.go`.
    - Successful reviews increase the interval and update the Easiness Factor based on a 0-5 grade.
    - Blackouts (grade < 3) reset the repetition cycle.
- **Endpoints:**
    - `GET /cortex/due`: Retrieves nodes pending review in a specific namespace.
    - `POST /cortex/review`: Records a review and schedules the next session.

### Dynamic Topic Clustering
- **Mechanism:** Automatic reorganization of "latent knowledge" from `ns-general` into domain-specific namespaces.
- **Process:**
    1. **Centroid Identification:** Nodes in `ns-general` are grouped using embedding similarity (threshold > 0.85).
    2. **LLM Synthesis:** The content of a cluster is sent to Gemini to generate a descriptive name (e.g., "Chess Openings") and a detailed description.
    3. **Neurogenesis:** A new namespace is created, and nodes are migrated with full provenance tracking.
- **Trigger:** Integrated into the end of the **REM Cycle** (Rapid Evidence Mapping).

### RPForest Partitioning (Advanced ANN)
- **Algorithm:** Enhanced Random Projection Forest using **Centroid-based Splitting**.
- **Accuracy:** Instead of random hyperplanes, the splitting plane is derived from the difference vector between two random points in a node, significantly increasing the quality of the approximate nearest neighbor search.

### Universal Knowledge Ingestion
- **Document Ingestor:** Leverages **Multimodal Gemini** to parse a wide array of formats:
    - **PDF & Images:** Processed via `genai.Blob` with rich technical extraction prompts.
    - **Plain Text:** Ingested as direct semantic context for extraction.
    - **Office Suite (.docx, .pptx, .xlsx):** Supported through Gemini's native multimodal understanding.
- **Codebase Mapper:** Automatically analyzes repository structure, extracting symbols and dependencies to build an architectural knowledge graph in `ns-engineering`.
- **Live Vision Bridge:** WebSocket support for binary frames, allowing Gemini to process visual context from user sessions in real-time.

## 🌐 Microservice Architecture

### Microservice Decoupling
- **Wisdom Engine (Backend):** Go-based core handling graph storage, vector search, and REM cycles.
- **Wisdom Portal (Frontend):** React/Vite SPA serving the Neural Atlas and Chat interface.
- **Communication:**
    - **REST API:** Standard data fetching.
    - **WebSockets:** Real-time chat and system notifications (e.g., when knowledge is consolidated).
- **Environment Management:** Uses `VITE_ENGINE_URL` and `VITE_WS_URL` to point to the backend service.

### Deployment & Persistence
- **Cloud Run:** Both services are deployed as independent Cloud Run instances.
- **Persistence:** SQLite database is stored in a GCS bucket (`gs://wisdom-cortex-jesus-mvp`) and mounted to the Engine via **GCS Fuse**. This ensures knowledge survives service restarts without the complexity of a full Spanner instance for the MVP.
- **Build Pipeline:** Uses `gcloud builds submit` with user-managed service accounts to bypass purged default project SAs.

## 🛠️ Tooling & Integration
- **WebSocket Chat:** Supports real-time "thinking" state and context-grounded responses.
- **Provenance:** Every node tracks its `author`, `source_type` (e.g., MANUAL, REM_CYCLE), and `source_ref` (e.g., session ID).
- **Temporal Logic:** Nodes support `valid_from` and `valid_until` for historical truth tracking.
