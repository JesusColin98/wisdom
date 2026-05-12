# TRACK 04: Researcher & Curriculum MOC

## Objective
Build the deterministic data scrapers and the Obsidian Map of Content (MOC) generator for flexible learning paths.

## Tasks

### 1. Deterministic Researcher (Ingestion)
- [ ] Create scraper modules for defined domains (e.g., RSS feeds, standard web scraping, PDF parsing).
- [ ] Extract raw content and strictly format it into Markdown.
- [ ] Publish the extracted data to NATS JetStream on the `wisdom.knowledge.ingested` subject using the CloudEvents JSON format defined in `CONTRACTS.md`.

### 2. Obsidian MOC (Map of Content) Generator
- [ ] Implement the `Curriculum` logic to handle Learning Paths.
- [ ] Instead of creating rigid database rows, generate a Node of type `Concept`.
- [ ] The `payload` of this node must be pure Markdown formatted as an Obsidian MOC (e.g., using bullet points and `[[Wikilinks]]`).
- [ ] Implement dynamic updating: When new concepts are ingested for a domain, append a new `[[Wikilink]]` to the existing MOC node's payload rather than rebuilding a SQL schema.

## Acceptance Criteria
- Data is extracted entirely deterministically (no LLM summarizing at the edge).
- NATS events exactly match the `CONTRACTS.md` JSON schema.
- Learning paths can be exported as `.md` files that open natively in Obsidian with valid internal linking.