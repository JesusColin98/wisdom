# Wisdom Cognitive Runtime: Master Overview

## The Hybrid Mastery Philosophy
Wisdom has evolved from a monolithic backend into a distributed **Cognitive Runtime**. The core philosophy is **Hybrid Mastery**: preserving our unique, proprietary backend intelligence (Go microservices for factual storage, mastery tracking, and autonomous research) while seamlessly integrating with established, validated UI ecosystems via the **Model Context Protocol (MCP)** and **Vertex AI Agent Platform Memory Bank**.

## The 5-Layer Architecture

### Layer 1: The Glass (UX & Observability)
We do not reinvent the wheel for knowledge management or flashcards.
*   **Wisdom Portal (React + Vite):** The primary user interface and our "Cognitive Cockpit". Contains the **Conversational Chat Interface** (the main entry point, wrapping the ADK Router), the **Note Editor** (Obsidian-compliant, with LIFT linter and "Polish with Gemini" rewrite), the **Vault Health Dashboard**, and the **Anki Export Review Panel**. See `FRONTEND_SPEC.md` and `CHATBOT_UI_SPEC.md`.
*   **Obsidian & Logseq (via MCP):** Secondary interfaces for browsing the Knowledge Graph. Wisdom generates 100% compliant local Markdown files with full YAML frontmatter, wikilinks, and PARA-compatible directory structure.
*   **Anki (via MCP):** The optional, validated UI for Spaced Repetition (SRS). Cards are pushed via the AI generation path or converted from existing Obsidian notes via the `md_to_anki` pipeline.

### Layer 2: Orchestration (The Router)
*   **Thalamus (ADK Gateway):** Built using the Agent Development Kit (ADK). This layer receives the user's prompt (e.g., "Analyze my chess game" or "What's the status of my Fibras?"), manages the session (`CreateSession`), and routes the request to the correct Domain Expert.

### Layer 3: Mix of Experts (MoE)
*   Highly decoupled, specialized AI agents handling specific domains.
*   Current Experts: **Finance (Fibras)**, **Chess**, **Language Learning**, **Technology**.
*   Each expert possesses unique prompts, logic, and tools, ensuring high-quality, domain-specific outputs without cross-domain hallucinations.

### Layer 4: Cognitive Memory (Vertex AI Memory Bank)
*   The LLM's long-term memory. Instead of storing raw chat logs, it uses **Reflective Memory Management**.
*   **Scoped Retrieval:** The Memory Bank guarantees isolation. The Chess Expert only retrieves memories with `scope: "Chess_Expert"`.
*   **Consolidation:** Uses `GenerateMemories` to asynchronously extract "Facts" (e.g., "User struggles with Spanish subjunctive") and overwrite outdated facts, preventing context window bloat.

### Layer 5: Factual Substrate (Go Microservices)
*   **Cortex:** Immutable data, massive graph relationships, and SQL storage for things that exceed the LLM context window.
*   **Researcher:** Autonomous crawler/scraper for content creation.
*   **Trace & Metabolism:** The proprietary Mastery engines that calculate the user's learning curve, acting as the ultimate source of truth, even if the user studies in Anki.
