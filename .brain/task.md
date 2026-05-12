# Task Tracker: Researcher & MOC

## Phase 1: Deterministic Researcher
- [x] Scaffold `pkg/researcher/scraper.go` for HTTP/HTML-to-Markdown extraction.
- [x] Scaffold `pkg/researcher/publisher.go` for NATS CloudEvents publishing.
- [x] Ensure output matches `CONTRACTS.md`.

## Phase 2: Obsidian MOC Generator
- [x] Scaffold `pkg/curriculum/moc.go`.
- [x] Implement MOC generation (Header + Bullet list of Wikilinks).
- [x] Implement MOC append logic (inserting new `[[Wikilink]]` into existing markdown).

## Phase 3: Integration
- [x] Create `cmd/researcher/main.go` entry point.
- [x] Create `Dockerfile.researcher`.
- [x] Update `cloudbuild.yaml` with Researcher image.
- [x] Update `terraform/main.tf` to deploy the Researcher.
