# Workspace Policy: chat_service (jesuscolin)
**Active Extensions:** bqml
**System Instructions:** Strictly adhere to the memory and execution protocols below.

## 1. Memory Architecture (Three-Tier Model)
This workspace utilizes a tiered memory system to ensure context efficiency and prevent drift.

*   **Tier 0: Workspace Policy & Context (`.gemini/GEMINI.md`)**
    *   **Purpose:** Core instructions, architectural mandates, and persistent background context.
    *   **Context Mandate:** You MUST proactively update the `Workspace Background Context` section of this file as you discover high-level project goals or persistent architectural intent.
*   **Tier 1: Local Workspace Memory (`.gemini/gemini_memory.md`)**
    *   **Purpose:** Active task tracking, technical strategy, touched artifacts, and implementation audit logs.
    *   **Initialization Mandate:** You MUST proactively update the `Task Objective`, `Target Bug`, and `Active CL` fields in this file as soon as the user defines the goal of the session.
    *   **Surgical Mandate:** ALWAYS use `replace` or `append` (via `run_shell_command`). You are STRICTLY FORBIDDEN from using `write_file` to update `gemini_memory.md`.
    *   **Fast Update Tool:** Use the memory sync tool for instant updates: `python3 cloud/helix/tools/gemini_cli/extensions/bqml/utils/memory_sync_tool.py "Status" "Phase"`. This tool bypasses the slow Gemini CLI orchestration layer and ensures global dashboard synchronization.
    *   **New Task Protocol:** When beginning a completely new, unrelated task in this workspace, you MUST archive the old memory file (e.g., `mv .gemini/gemini_memory.md .gemini/archive/old_task.md`) and re-initialize it from the template.
*   **Tier 2: Global Context Memory (`~/.gemini/gemini_memory.md`)**
    *   **Purpose:** Cross-workspace high-level dashboard. Status updates are automatically synced here and to daily logs.

## 2. Gated Technical Protocols
<PROTOCOL:PLAN>
1. **Research Phase:** Systematically map relevant files and symbols. Reproduce failure states or verify feature requirements empirically before proposing changes.
2. **Strategy Phase:** Draft a technical approach in the `Implementation Plan` section of the local memory file.
3. **Approval:** Present the plan and wait for explicit user confirmation before initiating any source code modifications.
</PROTOCOL:PLAN>

<PROTOCOL:IMPLEMENT>
1. **Validation-First:** Write verification logic (e.g., failing tests or reproduction scripts) before applying the fix/feature.
2. **Artifact Tracking:** Proactively list all modified, created, or deleted files in the `## 📂 Touched Artifacts` section of the local memory file.
3. **Surgical Update:** Apply changes to the minimum required lines. Maintain local stylistic consistency and idiomatic quality.
4. **Closure:** Execute project-specific validation (e.g., `blaze test`, `hg lint`, `pytype`) and document the outcome in the Session Log.
</PROTOCOL:IMPLEMENT>

## 3. Workspace Background Context
*The agent MUST update this section proactively to document the high-level project goal, background context, and any persistent architectural intent for the workspace.*

*   **Project Background:** Project Wisdom is a cognitive SRE engine designed to provide grounded knowledge retrieval and analysis using a Neural-Socratic loop. This workspace (`chat_service`) hosts the Python-based agent that interacts with users and utilizes the Go-based Wisdom Engine as its knowledge substrate via MCP tools.

## 4. Technology Stack & Core Dependencies
*The agent MUST keep this section updated in the active workspace file as it discovers the architecture.*

*   **Primary Languages:** Python, Go
*   **Build/Package System:** pip, Go modules
*   **Critical Context Paths:**
    *   `../wisdom/pkg/thalamus/...`
    *   `../wisdom/pkg/api/...`
*   **Core Dependencies:** FastAPI, requests, Google Generative AI (Gemini), uvicorn


## 5. Engineering Integrity & Safety
*   **Context Efficiency:** Minimize turns by utilizing parallel tool calls and reading only necessary file ranges.
*   **Security:** NEVER log or commit secrets, API keys, or internal credentials.
*   **Safety:** Briefly explain the technical rationale BEFORE executing any file modification or shell command.

## 6. Memory Guard Enforcement (CRITICAL)
*   **Start of Session:** You MUST read `.gemini/gemini_memory.md` at the beginning of EVERY session to synchronize your context.
*   **Initialization Mandate:** You MUST proactively initialize the `Task Objective`, `Target Bug`, and `Active CL` fields in `.gemini/gemini_memory.md` as soon as the user defines the session goal. Failure to do so within 3 turns will result in a hard block.
*   **Context Mandate:** You MUST proactively initialize the `Project Background` field in `.gemini/GEMINI.md` with the high-level project context. Failure to do so within 5 turns will result in a hard block.
*   **Task Updates:** You MUST update the `Status` and `Last Updated` fields in the `Task Tracker` and append a concise entry to the `Session Update Log` for every significant milestone or task completion.
*   **Thresholds:** A hard block will trigger after 10 turns or 30 tool calls without a memory update. Update memory to unblock.
