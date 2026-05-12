# Master Plan: Wisdom Cognitive Runtime

## Context
Wisdom is transitioning to a **Memory-as-a-Service** architecture. Gemini acts as the brain (LLM); Wisdom is purely the memory substrate. The architecture is built on Go (gRPC), NATS JetStream (CloudEvents), and Postgres/Supabase (jsonb universal graph schema).

## Tracks
This project is divided into parallel, isolated tracks that can be executed by different agents/teams without stepping on each other's toes, thanks to locked contracts.

- **[TRACK 01: Cortex Substrate](./TRACK_01_CORTEX.md)** - Database and core retrieval gRPC API.
- **[TRACK 02: Thalamus Gateway](./TRACK_02_THALAMUS.md)** - Context hydration and CoT auditing (Zero LLM calls).
- **[TRACK 03: Cerebellum Workers](./TRACK_03_CEREBELLUM.md)** - REM cycle, TTL garbage collection, conflict resolution.
- **[TRACK 04: Researcher & Curriculum](./TRACK_04_RESEARCH_MOC.md)** - Deterministic scraping and Obsidian MOC (Map of Content) management.

## Execution Rules for Agents (Antigravity)
1. **No LLM in Core Loop**: Do not import or use LLM SDKs (OpenAI, Anthropic, Gemini) in any service except the external client that consumes Thalamus.
2. **Strict Contracts**: Follow `docs/roadmap/CONTRACTS.md` strictly for NATS events and gRPC definitions.
3. **Database Schema**: Do not create new SQL tables. Use the Universal Graph Schema (`nodes` and `edges`) defined in `docs/roadmap/DATABASE_STRATEGY.md`.