# Wisdom Cognitive Runtime Refactoring: Cerebellum Workers

## Objective
Build the `Cerebellum` (Track 03) worker service. Cerebellum operates purely asynchronously in the background. It is responsible for event ingestion via NATS JetStream, database maintenance through the REM Cycle (Garbage Collection and Consolidation), and knowledge graph integrity checking (Conflict Resolution).

## Architecture

*   **Language**: Go
*   **Mode**: Background Worker (No inbound HTTP/gRPC traffic, connects to message broker and DB).
*   **Message Broker**: NATS JetStream
*   **Event Standard**: CloudEvents
*   **Dependencies**: Connects directly to the PostgreSQL Database (or via Cortex gRPC depending on specific operations, but direct DB access for bulk TTL deletion is more efficient). We will use direct DB access for REM cycle, and gRPC for ingesting new nodes.

## Phased Approach

### Phase 1: NATS Setup & Ingestion
1.  Configure the NATS connection and JetStream context.
2.  Implement a subscriber for the `wisdom.knowledge.ingested` topic.
3.  Implement an event handler that validates the CloudEvent payload and forwards it to `Cortex` via the gRPC `Memorize` endpoint.

### Phase 2: The REM Cycle (Cron Job)
1.  Implement a periodic runner (ticker).
2.  **Garbage Collection**: Directly query the PostgreSQL database to `DELETE FROM nodes WHERE type = 'Signal' AND ttl < NOW()`.
3.  **Consolidation**: Implement logic to promote heavily referenced `Signals` to `Facts`.

### Phase 3: Integrity Checker
1.  Implement a routine to scan for duplicate or contradicting facts.
2.  If found, interact with the DB to create a `CONTRADICTS` edge and set `requires_human = true`.
3.  Publish the `wisdom.memory.conflict_detected` event back to NATS.

### Phase 4: Integration
1.  Add `wisdom-cerebellum` build step to `cloudbuild.yaml`.
2.  Create `Dockerfile.cerebellum`.
3.  Update Terraform to provision a NATS instance (or use a managed one) and deploy the Cerebellum worker.
