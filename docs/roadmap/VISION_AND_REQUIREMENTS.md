# Wisdom Cognitive Runtime: Vision & Requirements

## Overview
Wisdom is evolving into a comprehensive **Cognitive Runtime**. The core philosophy is a "Hybrid Mastery" approach: we preserve our unique, proprietary backend engines (Cortex, Thalamus, Researcher, Metabolism, Trace) and our custom **Portal Dashboard**, while integrating with existing validated platforms (Obsidian, Anki, Logseq) via the **Model Context Protocol (MCP)**.

To maximize efficiency and intelligence, Wisdom utilizes a **Layered Mix of Experts (MoE) Architecture** orchestrated by the **Agent Development Kit (ADK)** and the **Vertex AI Memory Bank**. This ensures domain-specific agents (e.g., Finance, Chess, General Learning) handle requests independently, querying only the memories relevant to their domain.

## Core Directives
1. **Preserve Unique Value:** Maintain and enhance our unique Go microservices (especially our routing and backend logic) and the React Portal.
2. **Mission Control Portal:** The `portal/` acts as the observability hub for the entire system.
3. **Dual-Track Study System:** Wisdom implements its own Spaced Repetition/Mastery engine that coexists with Anki.
4. **Mix of Experts (MoE) via ADK:** Decouple logic by creating specialized agents for specific domains, orchestrated by a central router (`Thalamus`).
5. **Scoped Memory Bank:** Use Vertex AI Memory Bank to maintain isolated memory scopes per domain, preventing context bloat and cross-domain hallucinations.

## User Stories

1. **Mission Control Observability (Portal):**
   *As a user, I want to use the Portal to see the real-time status of the Researcher, monitor microservice health, and manage the "Knowledge Staging Area."*

2. **Domain-Specific Expertise (MoE):**
   *As a user, when I ask about my financial portfolio (Fibras) or my Chess openings, I want the system to route my request to a dedicated expert agent. That expert should only load memories relevant to its domain (e.g., the Chess agent remembers my ELO, but doesn't care about my stock investments).*

3. **Proprietary Mastery Engine vs. Anki Coexistence:**
   *As a user, I want to choose between studying in Anki or in Wisdom's custom UI. Wisdom should track my mastery regardless of the UI I choose.*

4. **Seamless Knowledge Management (Obsidian/Logseq):**
   *As a user, I want my knowledge organized as local Markdown files that integrate automatically with Obsidian/Logseq.*

## Technical Requirements & Architecture Layers

### Layer 1: Experience & UI (The "Glass")
- **Obsidian / Logseq:** Connected via MCP for reading/writing Markdown knowledge graphs.
- **Anki:** Connected via MCP for executing spaced repetition sessions.
- **Wisdom Portal:** The React dashboard serving as Mission Control.

### Layer 2: Orchestration (The "Thalamus" Router)
- Built using **ADK (Agent Development Kit)**.
- **Routing Logic:** Analyzes the incoming user intent and routes it to the correct Expert Agent.
- Maintains the master `CreateSession` ID to track the entire conversation flow.

### Layer 3: Mix of Experts (The "Agents")
Decoupled, modular agents that handle specific business logic.
- **Finance/Fibras Expert:** Analyzes portfolio metrics, tracks dividend goals, and understands risk tolerance.
- **Chess Expert:** Analyzes game mechanics, ELO, opening repertoires, and strategic weaknesses.
- **General Learning Expert:** Handles generic knowledge extraction, reading comprehension, and study scheduling.

### Layer 4: Cognitive Memory (Vertex AI Memory Bank)
- **Scoped Retrieval:** Each Expert Agent triggers `RetrieveMemories` with a specific scope (e.g., `agent_name: "Chess_Expert"`). This saves tokens and sharpens context.
- **Topic-Based Consolidation:** `GenerateMemories` is configured with domain-specific Custom Topics. 
  - *Example:* The Fibras expert uses a Custom Topic for `ASSET_ALLOCATION_PREFERENCES`.
- **Identity Isolation:** All memories are strictly tied to the `user_id`.

### Layer 5: Factual Substrate (The "Cortex")
- Our Go-based internal storage for immutable facts, crawled data, and global relationships that are larger than what should fit in an LLM's context window.
- Connected via **gRPC** to the Expert Agents.
