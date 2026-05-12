# Wisdom Cognitive Runtime Refactoring: Researcher & MOC

## Objective
Build the `Researcher` and `Curriculum` services (Track 04). The Researcher acts as the deterministic data ingestion pipeline, scraping content and publishing it to NATS. The Curriculum module manages learning paths dynamically by formatting `Concept` nodes as Obsidian Maps of Content (MOCs) using markdown and `[[Wikilinks]]`.

## Architecture
*   **Language**: Go
*   **Components**: 
    *   `pkg/researcher`: Deterministic scraper and NATS publisher.
    *   `pkg/curriculum`: MOC generator and updater.
*   **Message Broker**: NATS JetStream (Publisher)
*   **Target Output**: CloudEvents JSON (Researcher) and Obsidian-compatible Markdown (Curriculum).

## Phased Approach

### Phase 1: Deterministic Researcher
1.  Scaffold `pkg/researcher`.
2.  Implement a basic HTTP scraper (e.g., fetching a webpage and converting its core content to Markdown).
3.  Implement the NATS Publisher.
4.  Construct the `wisdom.knowledge.ingested` CloudEvent payload according to `CONTRACTS.md` and publish it.

### Phase 2: Obsidian MOC Generator (Curriculum)
1.  Scaffold `pkg/curriculum`.
2.  Implement logic to create a new `Concept` node payload containing an initial MOC structure.
3.  Implement logic to *update* an existing MOC by parsing the Markdown payload and appending a new `[[Wikilink]]`.

### Phase 3: Integration
1.  Create `cmd/researcher/main.go` entry point.
2.  Create `Dockerfile.researcher`.
3.  Update `cloudbuild.yaml` and `terraform/main.tf` to deploy the Researcher service (typically as a Cloud Run Job or scheduled service, though we'll define it as a service for consistency).
