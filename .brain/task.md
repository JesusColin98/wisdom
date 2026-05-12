# Task Tracker: Cortex Substrate Refactoring

## Phase 1: Substrate Definition
- [x] Create `.brain` tracking directory.
- [x] Create `cortex.proto` with standard gRPC interface (`Memorize`, `Recall`, `QueryHechos`).
- [x] Create `schema_v2.sql` defining `nodes` and `edges` tables with ENUMs and JSONB.
- [x] Update `node.go` with Go structs matching the new schema.
- [ ] Ensure `protoc` compiles `cortex.proto` into Go code.

## Phase 2: Engine Implementation (PostgreSQL)
- [x] Refactor `PostgresEngine` in `postgres_engine.go` to connect and initialize using `schema_v2.sql`.
- [x] Implement `Memorize` logic (upsert node, handle TTL for Signals).
- [x] Implement `QueryHechos` logic (query JSONB metadata).
- [x] Implement `Recall` logic (fetch node + direct edges/neighbors).

## Phase 3: gRPC Server Setup
- [x] Scaffold `server.go` for the Cortex gRPC service.
- [x] Implement `grpc.health.v1.Health` standard.
- [x] Write basic unit tests for the gRPC endpoints.

## Phase 4: Validation
- [x] Verify `cortex` builds without errors.
- [x] Verify no LLM SDKs are imported in the `cortex` package.
