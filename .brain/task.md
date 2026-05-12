# Task Tracker: Cerebellum Workers

## Phase 1: NATS Setup & Ingestion
- [x] Add CloudEvents and NATS Go dependencies to `go.mod`.
- [x] Scaffold `pkg/cerebellum/worker.go`.
- [x] Implement NATS JetStream subscriber.
- [x] Implement CloudEvent parsing and forwarding to Cortex via gRPC.

## Phase 2: The REM Cycle
- [x] Implement cron/ticker logic for periodic execution.
- [x] Implement TTL Garbage Collection (Direct Postgres `DELETE`).
- [x] Implement Signal to Fact promotion (Consolidation).

## Phase 3: Integrity Checker
- [x] Implement conflict detection logic.
- [x] Implement `CONTRADICTS` edge creation and `requires_human` flag update.
- [x] Implement NATS publisher for conflict detection events.

## Phase 4: Integration
- [x] Create `cmd/cerebellum/main.go` entry point.
- [x] Create `Dockerfile.cerebellum`.
- [x] Update `cloudbuild.yaml` with Cerebellum image.
- [x] Update `terraform/main.tf` to deploy the worker.
