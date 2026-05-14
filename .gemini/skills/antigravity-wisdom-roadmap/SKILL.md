---
name: antigravity-wisdom-roadmap
description: Orchestrates the architectural changes for the Wisdom Cognitive Runtime, implementing a Mission Control Portal, Dual-Track Study, and a Mix of Experts (MoE) using ADK and Memory Bank.
version: 1.0.3
---

# Wisdom Cognitive Runtime Orchestrator

<instructions>
Use this skill when implementing or modifying Wisdom's architecture. The goal is a **Hybrid Cognitive Runtime** with a decoupled, layered **Mix of Experts (MoE)** architecture using the **Agent Development Kit (ADK)** and **Vertex AI Memory Bank**.

## Core Directives

1. **Layered Mix of Experts (MoE) Architecture:**
   - **Orchestration:** Use a central Router (Thalamus) built with ADK to intercept user requests and route them to specific Domain Experts.
   - **Domain Experts:** Create decoupled agents for specific domains (e.g., `Finance/Fibras Agent`, `Chess Agent`, `General Learning Agent`).
   - **Independence:** Each expert must be a modular unit that can be updated or swapped without affecting the rest of the system.

2. **Scoped Memory Management (Vertex AI Memory Bank):**
   - **Scope Isolation:** Ensure that each Expert Agent only retrieves memories relevant to its domain to save tokens and improve accuracy (e.g., the Chess Agent uses the scope `agent_name: "Chess_Expert"`).
   - **Custom Topics:** Define domain-specific extraction topics for `GenerateMemories` (e.g., `FINANCIAL_GOALS`, `CHESS_WEAKNESSES`).
   - **Flow:** Maintain continuous event tracking (`AppendEvent`) per session.

3. **Mission Control Portal:**
   - **PRESERVE:** The `portal/` React frontend. It is the observability hub for the entire system (Scraping status, Resource health, Memory routing logs).

4. **Dual-Track Study System (Proprietary vs. Anki):**
   - Implement Wisdom's custom mastery calculation in the backend (`Metabolism/Trace`).
   - Provide an optional sync bridge to Anki via `anki-mcp-server`. 

5. **Knowledge Management (Obsidian/Logseq):**
   - Use `obsidian-mcp-server` and `mcp-logseq` for note management and graph visualization. Produce strict, high-quality Markdown.

## Execution Workflow
1. **Agent Integration:** When adding a new domain (like Fibras or Chess), define its Agent profile and its specific Memory Topics in the ADK configuration first.
2. **Context Efficiency:** Never load global memory for a domain-specific task. Always use similarity search and scoped retrieval in the Memory Bank.
3. **Microservices:** Implement the complex data parsing and long-running tasks in the Go backend (gRPC), leaving the ADK Agents lightweight and focused on cognitive reasoning and tool execution.
</instructions>

<available_resources>
- Vertex AI Agent Platform Memory Bank Documentation
- Agent Development Kit (ADK) Documentation
- obsidian-mcp-server Documentation
- anki-mcp-server Documentation
</available_resources>
