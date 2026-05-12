# TRACK 01: Cortex Substrate

## Objective
Implement the foundational persistence layer of Wisdom using a flexible, two-table graph schema in PostgreSQL.

## Tasks

### 1. Database Initialization
- [ ] Connect to PostgreSQL (or Supabase).
- [ ] Create `nodes` table:
  - `id` (UUID, Primary Key)
  - `type` (ENUM: Fact, Signal, Concept, User)
  - `payload` (JSONB)
  - `confidence` (FLOAT)
  - `requires_human` (BOOLEAN, default false)
  - `ttl` (TIMESTAMP, nullable)
- [ ] Create `edges` table:
  - `source_id` (UUID, Foreign Key -> nodes)
  - `target_id` (UUID, Foreign Key -> nodes)
  - `relation` (VARCHAR/ENUM)
- [ ] Create indexes on `payload` (GIN) for fast JSON querying.

### 2. Cortex gRPC Service
- [ ] Define `Cortex` service in `cortex.proto`.
- [ ] Implement `Memorize(IngestRequest)`: Upserts nodes into the database. If it's a Signal, set the TTL.
- [ ] Implement `QueryHechos(FactRequest)`: Retrieves `Fact` nodes based on metadata inside the JSONB `payload`.
- [ ] Implement `Recall(RecallRequest)`: Retrieves a node and its direct neighbors via `edges`.

## Acceptance Criteria
- Database queries use native Postgres JSONB operations.
- `cortex.proto` compiles and the Go gRPC server runs on port `50051`.
- Zero LLM dependencies in this codebase.