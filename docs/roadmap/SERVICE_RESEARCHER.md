# Service: Wisdom-Researcher

## Core Concept
The `Wisdom-Researcher` is a standalone Go microservice responsible for autonomous, deterministic data gathering. It acts as the "eyes and hands" of the Expert Agents.

## Capabilities & Modules

### 1. NotebookLM Export Ingestion (Structured Knowledge)
*   **Function:** Process rich knowledge dumps exported from NotebookLM.
*   **Mechanism:** Parses NotebookLM zip archives, normalizes citations, and breaks content into atomic blocks for the `Cortex` substrate.

### 2. Autonomous Crawler (Web & RSS)
*   **Function:** Subscribes to technical blogs, financial news, and feeds.
*   **Mechanism:** Periodically fetches XML/HTML, strips boilerplate, and stores raw text in `Cortex`.

### 3. Book-Vault (Deep Research)
*   **Function:** Interacts with DDL sources (Anna's Archive, Z-Library) or local PDF directories.
*   **Mechanism:** Uses OCR and PDF text extraction, chunking books into semantic sections.

## Content Creation Pipeline
1.  Expert calls the `Wisdom-Researcher` via gRPC.
2.  The `Researcher` gathers raw content from preferred sources.
3.  The `Researcher` sanitizes the data and returns raw Markdown/Facts to the Expert.
4.  The Expert (Logic Layer) then determines how to format this for Obsidian/Anki.
