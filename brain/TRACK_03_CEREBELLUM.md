# TRACK 03: Cerebellum Workers

## Objective
Implement the asynchronous background workers for garbage collection, memory consolidation (REM Cycle), and conflict resolution.

## Tasks

### 1. NATS JetStream Setup
- [ ] Initialize NATS JetStream connection.
- [ ] Subscribe to the `wisdom.knowledge.ingested` topic (CloudEvents standard).
- [ ] Upon receiving an ingested event, validate it and forward it to `Cortex` via gRPC for storage.

### 2. The REM Cycle (Cron Job)
- [ ] Implement a Go ticker or cron job that runs periodically (e.g., daily at 3 AM).
- [ ] **Garbage Collection**: Query `Cortex` for all `Signal` nodes where `ttl < NOW()`. Hard delete them.
- [ ] **Consolidation**: Identify `Signal` nodes with high reference counts (queried frequently). Update their type to `Fact` and remove their TTL.

### 3. Conflict Resolution Engine
- [ ] Create an integrity checker that scans `Fact` nodes for overlapping or contradicting payloads.
- [ ] If a conflict is found:
  - Create a `CONTRADICTS` edge in the `edges` table between the two nodes.
  - Set `requires_human = true` on the losing node (lower `confidence`).
  - Publish a `wisdom.memory.conflict_detected` NATS event.

## Acceptance Criteria
- Cerebellum operates asynchronously without exposing external synchronous APIs.
- NATS events are strictly typed as CloudEvents JSON.
- REM cycle successfully deletes expired TTL nodes from the Postgres database.