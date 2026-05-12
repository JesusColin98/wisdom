# Implementation Plan: Microservices Migration

This document outlines the phased transition from the current Go monolith to the distributed Cognitive Runtime.

## Phase 1: Substrate Hardening & gRPC Mesh
- **Goal**: Formalize the internal communication layer and decouple storage.
- **Task**: 
    - Define common Protobuf schemas in `pkg/proto`.
    - Separate `pkg/cortex` into a standalone gRPC service. 
    - Implement "Hechos" vs "Signals" metadata tags.
- **Verification**: Other services must use gRPC clients to communicate with the Cortex; direct SQL/local access is deprecated.

## Phase 2: Deterministic Researcher
- **Goal**: Launch the `Wisdom-Researcher` service.
- **Task**: Implement the Python-based Book-Vault and Go-based RSS Crawler.
- **Verification**: A request to `/investigate` returns a stream of Markdown signals with Obsidian metadata.

## Phase 3: Entity & Personalization (Entity + Trace)
- **Goal**: Move user state and recognition out of the core.
- **Task**: Implement the symbol recognition logic (@, #). Create the `Wisdom-Trace` service to calculate mastery scores.
- **Verification**: Generating a learning path for User A vs User B results in different recommended concepts.

## Phase 4: UI/UX Refactor
- **Goal**: Modernize the Portal to handle multiple backend sources.
- **Task**: Update `LearningView.jsx` to consume the new `Curriculum` and `Researcher` endpoints.
- **Verification**: Visualizing a "Big Topic" (e.g., Science) shows a recursive tree from Macro-Domain down to Atomic Concepts.
