# Subsystem: Wisdom-Cerebellum (Asynchronous Motor Tasks)

The Cerebellum handles all heavy, asynchronous processing that must not block the main memory retrieval loops. It acts as the background worker for the Cognitive Runtime.

## The REM Cycle (Garbage Collection & Consolidation)
The REM cycle is a scheduled cron-like job orchestrated by the Cerebellum. It has two main phases:

1. **Garbage Collection (Forgetting)**:
   - Scans all `Signal` nodes (ephemeral memory).
   - If a Signal's TTL has expired and it has a `reference_count` of 0, it is **Hard Deleted**.
2. **Consolidation (Learning)**:
   - If an ephemeral Signal has a high `reference_count` or was flagged during an Audit, its type is promoted to `Fact`.

## Conflict Resolution Engine
When two facts contradict:
- The Cerebellum creates a `CONTRADICTS` edge between them.
- It calculates the `confidence_weight` based on provenance (e.g., an academic paper has higher weight than a blog post).
- The node with the higher weight is returned in default queries.
- The conflict is flagged with a `REQUIRES_HUMAN_RESOLUTION` boolean for user intervention.

## Interface (Events)
The Cerebellum primarily listens to NATS events (e.g., `MEMORY.ingested`) rather than serving synchronous gRPC requests.
