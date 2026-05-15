# Domain Experts: The MoE Architecture

Wisdom utilizes a **Mix of Experts (MoE)** architecture, orchestrated by the Agent Development Kit (ADK). This decouples business logic, allowing us to support diverse, complex domains while ensuring high-quality output and strict memory isolation via the Vertex AI Memory Bank.

## The Routing Mechanism (Thalamus)
When a user inputs a query, the `Thalamus` router evaluates the intent and dispatches the execution to one of the isolated Expert Agents. Each agent has its own `system_instruction` and specific toolset.

## Supported Domains

### 1. Finance Expert (Fibras & Investments)
*   **Responsibility:** Analyze portfolios, track real estate investment trusts (Fibras), dividend yields, and risk tolerance.
*   **Memory Bank Scope:** `agent_name: "Finance_Expert"`.
*   **Consolidation Topics:** `ASSET_ALLOCATION`, `RISK_PROFILE`, `DIVIDEND_GOALS`.
*   **Tools:** Can query `Cortex` for historical stock/Fibras data scraped by the `Researcher`.

### 2. Chess Expert
*   **Responsibility:** Analyze game PGNs, build opening repertoires, and identify tactical weaknesses.
*   **Memory Bank Scope:** `agent_name: "Chess_Expert"`.
*   **Consolidation Topics:** `CURRENT_ELO`, `OPENING_REPERTOIRE`, `TACTICAL_BLUNDERS`.
*   **Integration:** Generates Obsidian Markdown files with visual board representations (using FEN notation plugins) and exports tactical puzzles to Anki.

### 3. Language Learning Expert
*   **Responsibility:** Teach vocabulary, grammar rules, and cultural context.
*   **Memory Bank Scope:** `agent_name: "Language_Expert"`.
*   **Consolidation Topics:** `TARGET_LANGUAGES`, `CEFR_LEVEL`, `STRUGGLED_GRAMMAR`.
*   **Integration:** Generates highly structured Anki cards with audio (TTS) and cloze deletions.

### 4. Technology & General Learning Expert
*   **Responsibility:** Teach software engineering, system architecture, and general concepts.
*   **Memory Bank Scope:** `agent_name: "Tech_Expert"`.
*   **Integration:** Heavy reliance on Obsidian Markdown, generating interconnected Knowledge Graphs using `[[Wikilinks]]` for deep, conceptual exploration.

### 5. Dynamic Experts (The Expansion Layer)
*   **Responsibility:** Any domain registered at runtime (e.g., Sales, Philosophy, Music).
*   **Architecture:** Uses the `DynamicExpert` class to instantiate agents from a JSON configuration.
*   **Persistence:** Configurations are stored in `Cortex` as `DomainConfig` facts, ensuring persistence across service restarts.
*   **Capabilities:** All dynamic experts inherit the standard **Autonomous Research** tool and **Memory Consolidation** pipeline.

## Autonomous Research Capability
All experts can now invoke the `Researcher` microservice. When an expert detects a knowledge gap or the user requests a deep-dive, the expert:
1.  Triggers `_tool_research_topic(topic)`.
2.  The `Researcher` service crawls the web and generates a `ResearchReport`.
3.  The report is ingested into the user's `Knowledge Graph` (Obsidian) and `Memory Bank`.

## Memory Isolation Guarantee
Because each Expert uses a hardcoded `agent_name` scope when calling `RetrieveMemories` and `GenerateMemories` from the Vertex AI Memory Bank, **data never overlaps**. The Finance Agent will never hallucinate a chess move into your investment strategy, and the LLM context window remains incredibly efficient.
