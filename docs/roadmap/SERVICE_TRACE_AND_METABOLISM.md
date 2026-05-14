# Service: Wisdom-Mastery (Trace & Metabolism)

## Core Concept
Wisdom's true intellectual property lies in the `Trace` (Tracking) and `Metabolism` (Spaced Repetition/Scheduling) engines. 

*Architecture Decision:* **Trace and Metabolism are deployed as a single, highly cohesive Go microservice (`Wisdom-Mastery`).** They share the same database and process because Metabolism's interval calculations are intimately tied to Trace's mastery state. Splitting them would introduce unnecessary network latency.

While we delegate the study UI to Anki, **Wisdom remains the ultimate source of truth for the user's Mastery Level.**

## The Dual-Track Study System
We guarantee a system where our proprietary mastery engine and Anki can coexist.

### 1. Wisdom Native Study (Portal)
*   **Algorithm:** `Wisdom-Metabolism` uses a highly advanced, AI-driven Spaced Repetition System. Unlike standard SM-2 (Anki's default), Metabolism incorporates the LLM's understanding of *why* you failed a concept (e.g., "Failed because of vocabulary, not grammar").
*   **UI:** Hosted in the React `portal/` as a fallback or advanced study method.
*   **Flow:** User studies in Portal -> `Metabolism` calculates next interval -> `Trace` updates `MasteryScore` in `Cortex`.

### 2. Anki Coexistence (MCP Sync)
*   **Algorithm:** Anki uses its internal FSRS or SM-2 algorithms.
*   **UI:** The native Anki desktop/mobile app.
*   **Flow (The Sync Bridge):** 
    1. Wisdom creates cards in Anki via `anki-mcp-server`.
    2. User reviews cards in Anki.
    3. A background job in the `Integrations` microservice periodically queries Anki via MCP: *"Get review history for cards tagged `Wisdom`"*.
    4. Wisdom maps Anki's review logs (Again, Hard, Good, Easy) to our internal `MasteryScore`.
    5. `Trace` updates the `Cortex` database.

## The Mastery Guarantee
Regardless of whether the user studies a Chess tactic in Anki or on the Wisdom Portal, the `MasteryScore` is synchronized. 

When the `Chess_Expert` agent generates a new learning path, it queries the `Trace` service: *"What is the user's mastery of the Caro-Kann?"* Because `Trace` synced with Anki, the agent knows exactly what to teach next, ensuring zero redundant learning and a perfect User Experience.
