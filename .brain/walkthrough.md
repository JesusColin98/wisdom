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

## Next Steps
*   All core infrastructure and backend services (Cortex, Thalamus, Cerebellum, Researcher) are now implemented under the deterministic ruleset.
*   Next actions should focus on deploying to GCP, testing the entire event-driven flow (from Cloud Run Job -> NATS -> Cerebellum -> Postgres), and ensuring the Python Voice Proxy can successfully connect to the Thalamus Gateway.
