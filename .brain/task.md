# Task Tracker: Researcher & MOC

## Phase 1: Deterministic Researcher
- [x] Scaffold `pkg/researcher/scraper.go` for HTTP/HTML-to-Markdown extraction.
- [x] Scaffold `pkg/researcher/publisher.go` for NATS CloudEvents publishing.
- [x] Ensure output matches `CONTRACTS.md`.

## Phase 2: Obsidian MOC Generator
- [x] Scaffold `pkg/curriculum/moc.go`.
- [x] Implement MOC generation (Header + Bullet list of Wikilinks).
- [x] Implement MOC append logic (inserting new `[[Wikilink]]` into existing markdown).

## Phase 5: Gap Clearance
- [x] Implement batched node fetching for `Recall` in `postgres_engine.go`.
- [x] Update `server.go` to return full neighbor nodes in `Recall`.
- [x] Create `scripts/setup_dev_env.ps1` (Windows friendly) and `sh` versions.
- [x] Add `backend "gcs"` placeholder to `terraform/main.tf`.
- [x] Remove `allUsers` invoker role for Cortex and use service-to-service IAM.

