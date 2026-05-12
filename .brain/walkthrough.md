# Technical Walkthrough: Wisdom Refactoring

## [2026-05-12] Cerebellum Workers Completion
*   **Cleanup:** Purged legacy LLM dependencies to enforce the "Zero LLMs" rule.
*   **NATS & CloudEvents:** Integrated NATS JetStream. `Cerebellum` subscribes to `wisdom.knowledge.ingested` and funnels events into `Cortex`.
*   **The REM Cycle:** Implemented the async `runREMCycle` ticker for Garbage Collection (hard delete of expired Signals) and Integrity Checking (URL duplication detection, CONTRADICTS edge creation).
*   **CI/CD & DevOps:** Updated Cloud Build and Terraform.

## [2026-05-12] Researcher & MOC Initialization
*   Restored `TRACK_04_RESEARCH_MOC.md` from git history.
*   Drafted the `implementation_plan.md` and `task.md` for the Researcher (NATS publisher) and Curriculum (Obsidian MOC generator) packages.

## [2026-05-12] Researcher & MOC Completion
*   **Researcher Scraper:** Implemented `pkg/researcher/scraper.go` to deterministically convert raw HTML web pages into clean Markdown using the `html-to-markdown` package. 
*   **NATS Publisher:** Implemented `pkg/researcher/publisher.go` to construct standard CloudEvents containing the scraped Markdown and push them to the `wisdom.knowledge.ingested` NATS JetStream topic.
*   **Curriculum (Obsidian MOC):** Implemented `pkg/curriculum/moc.go` to generate and deterministically append `[[Wikilinks]]` to a Markdown string formatted as an Obsidian Map of Content. This avoids rigid SQL structures and enables fully flexible learning paths.
*   **CI/CD & DevOps:** Created `cmd/researcher/main.go` and `Dockerfile.researcher`. Updated `cloudbuild.yaml` to build the new image, and `terraform/main.tf` to configure the Researcher as a `google_cloud_run_v2_job` instead of a service, allowing on-demand execution.

## [2026-05-12] Infrastructure & CI/CD Refinement
*   **Fix (Docker Hub Compatibility):** Discovered that official `golang` Docker images dropped support for `bullseye` in Go 1.25. Upgraded all Dockerfiles and `cloudbuild.yaml` to use the valid `golang:1.25-bookworm` base image.
*   **Fix (Dependency Conflict):** Re-upgraded the local `go.mod` to `go 1.25.0` to satisfy the strict requirements of the `github.com/nats-io/nats.go@v1.52.0` package, completing the fix.
*   **Cleanup:** Purged the remaining legacy architecture packages (`pkg/mcp`, `pkg/sensory`, `pkg/api`, `pkg/kernel`) and their respective entry points (`cmd/wisdom-api`, `cmd/wisdom-mcp`). These files were referencing deleted structures from Cortex and violating the new distributed microservices contract.
*   **Researcher & MOC Completion:** Successfully implemented the deterministic ingestion pipeline and the Obsidian MOC generator for flexible learning paths.
*   **Status:** All 4 tracks (Cortex, Thalamus, Cerebellum, Researcher) are fully implemented, tested, and integrated into the CI/CD pipeline.

## Mission Accomplished
The Wisdom Cognitive Runtime has been successfully refactored into a high-performance, deterministic memory substrate. 
*   **Cortex:** Scalable Postgres graph substrate.
*   **Thalamus:** Context hydration gateway (Markdown-first).
*   **Cerebellum:** Async worker for cleanup and conflicts.
*   **Researcher:** Ingestion via CloudEvents.

## Next Steps
*   Execute `terraform apply` to provision the Cloud SQL instance and services.
*   Run the first `researcher-job` to hydrate the database with initial facts.

## [2026-05-12] Gap Clearance & Optimization
*   **Graph Retrieval:** Optimized `Recall` by implementing a batched `GetNodes` method. Now, a `Recall` request returns not just the edges but also the full `Node` objects for all direct neighbors in the graph. Fixed a slice-to-SQL conversion issue by integrating `github.com/lib/pq` for the `ANY($1)` clause.
*   **DevOps:** Created `scripts/setup_dev_env.ps1` and `scripts/setup_dev_env.sh` to automate the local installation of `protoc` and the necessary Go plugins.
*   **Infrastructure Hardening:** 
    *   Configured a placeholder for the Terraform GCS remote backend in `main.tf`.
    *   Restricted the `wisdom-cortex` Cloud Run service to internal access only. Now, only authenticated services (like Thalamus) using the `wisdom_sa` identity can invoke the database substrate. 

