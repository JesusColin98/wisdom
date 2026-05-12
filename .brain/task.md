# Task Tracker: Researcher & MOC

## Phase 1: Deterministic Researcher
- [ ] Scaffold `pkg/researcher/scraper.go` for HTTP/HTML-to-Markdown extraction.
- [ ] Scaffold `pkg/researcher/publisher.go` for NATS CloudEvents publishing.
- [ ] Ensure output matches `CONTRACTS.md`.

## Phase 2: Obsidian MOC Generator
- [ ] Scaffold `pkg/curriculum/moc.go`.
- [ ] Implement MOC generation (Header + Bullet list of Wikilinks).
- [ ] Implement MOC append logic (inserting new `[[Wikilink]]` into existing markdown).

## Phase 3: Integration
- [ ] Create `cmd/researcher/main.go` entry point.
- [ ] Create `Dockerfile.researcher`.
- [ ] Update `cloudbuild.yaml` with Researcher image.
- [ ] Update `terraform/main.tf` to deploy the Researcher.
