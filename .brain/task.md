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
- [ ] Implement batched node fetching for `Recall` in `postgres_engine.go`.
- [ ] Update `server.go` to return full neighbor nodes in `Recall`.
- [ ] Create `scripts/setup_dev_env.ps1` (Windows friendly) and `sh` versions.
- [ ] Add `backend "gcs"` to `terraform/main.tf`.
- [ ] Remove `allUsers` invoker role and use service-to-service IAM.

