# Wisdom Cognitive Runtime Refactoring: Implementation Plan

## Objective
Refactor the `Cortex` substrate (Storage & Retrieval) of the Wisdom project to adopt a "Memory-as-a-Service" architecture. The new architecture relies on a flexible, two-table graph schema in PostgreSQL, using `jsonb` for dynamic attributes, entirely decoupled from LLM dependencies.

## Architecture

*   **Language**: Go (gRPC)
*   **Database**: PostgreSQL / Supabase
*   **Schema**: Universal Graph Schema (`nodes` and `edges`)
    *   `nodes`: UUID, Type (Fact, Signal, Concept, User), Payload (JSONB), Confidence, RequiresHuman, TTL.
    *   `edges`: Source UUID, Target UUID, Relation (THEORY_OF, CONTRADICTS, PREREQUISITE_OF, MASTERED_BY).
*   **Protocol**: gRPC (`cortex.proto`)
*   **Events**: NATS JetStream (CloudEvents) - *To be integrated in later tracks*

## Phased Approach

### Phase 1: Substrate Definition (Current Focus)
1.  Establish the Protobuf contract (`cortex.proto`) defining `Memorize`, `Recall`, and `QueryHechos`.
2.  Define the PostgreSQL schema adhering to the Universal Graph Schema.
3.  Implement the Go models representing Nodes and Edges.

### Phase 2: Engine Implementation
1.  Refactor `postgres_engine.go` to use the new two-table schema.
2.  Implement `PutNode`, `GetNode`, and edge creation logic.
3.  Ensure zero LLM dependencies in the storage layer.

### Phase 3: gRPC Server
1.  Implement the Cortex gRPC server fulfilling the protobuf interface.
2.  Wire the server to the `PostgresEngine`.
3.  Add health checks and validation.

### Phase 4: Integration & Cleanup
1.  Clean up legacy SQLite code and models if no longer needed by the Core Substrate.
2.  Prepare for NATS integration (Track 03).
