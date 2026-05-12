# Technical Walkthrough: Wisdom Refactoring

## [2026-05-12] Thalamus Gateway Completion
*   **Go Implementation:** Built the Thalamus server in `pkg/thalamus/server.go`. It securely accesses `Cortex` via gRPC, extracts `Fact` payloads, and formats them into clean, LLM-ready markdown. Mapped LLM traces to `Cortex` as `Signal` nodes.
*   **Testing:** Validated logic using mock gRPC clients in `server_test.go`.
*   **Infrastructure:** Updated Terraform to provision `wisdom-thalamus` on Cloud Run. Linked `CORTEX_GRPC_URL` dynamically. Updated `wisdom-chat` to point to Thalamus. Added to CI/CD pipeline.

## [2026-05-12] Cerebellum Workers Initialization
*   Restored `TRACK_03_CEREBELLUM.md` from git history.
*   Drafted the `implementation_plan.md` and `task.md` for the Cerebellum service. The architecture is defined as a headless background worker subscribing to NATS JetStream.

## [2026-05-12] Cerebellum Workers Completion
*   **Cleanup:** Purged legacy LLM dependencies (`llm.go`, `llm_vertex.go`) to strictly enforce the "Zero LLMs in the core loop" rule.
*   **NATS & CloudEvents:** Integrated `nats.go` and `sdk-go/v2` (CloudEvents). `Cerebellum` now subscribes to `wisdom.knowledge.ingested` and correctly funnels events into `Cortex` via the `Memorize` gRPC endpoint.
*   **The REM Cycle:** Implemented the async `runREMCycle` ticker. 
    *   *Garbage Collection:* Executes a hard Postgres `DELETE` on all `Signal` nodes with an expired TTL.
    *   *Integrity Checker:* Scans the `Fact` nodes for exact URL duplication. If a conflict occurs, it sets `requires_human = true`, links the nodes with a `CONTRADICTS` edge, and fires a `wisdom.memory.conflict_detected` event back to NATS.
*   **CI/CD & DevOps:** Created `cmd/cerebellum/main.go`, `Dockerfile.cerebellum`, and updated both `cloudbuild.yaml` and `terraform/main.tf` to build and deploy the background worker.

## Next Steps
*   Start `TRACK_04_RESEARCH_MOC.md` to define the ingestion pathways (scrapers) that will feed NATS JetStream.
*   Review the Terraform architecture for NATS; currently using `demo.nats.io` as a placeholder, production requires deploying a managed broker or configuring Cloud Run to use Cloud Pub/Sub.
