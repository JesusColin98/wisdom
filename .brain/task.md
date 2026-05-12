# Task Tracker: Cerebellum Workers

## Phase 1: NATS Setup & Ingestion
- [ ] Add CloudEvents and NATS Go dependencies to `go.mod`.
- [ ] Scaffold `pkg/cerebellum/worker.go`.
- [ ] Implement NATS JetStream subscriber.
- [ ] Implement CloudEvent parsing and forwarding to Cortex via gRPC.

## Phase 2: The REM Cycle
- [ ] Implement cron/ticker logic for periodic execution.
- [ ] Implement TTL Garbage Collection (Direct Postgres `DELETE`).
- [ ] Implement Signal to Fact promotion (Consolidation).

## Phase 3: Integrity Checker
- [ ] Implement conflict detection logic.
- [ ] Implement `CONTRADICTS` edge creation and `requires_human` flag update.
- [ ] Implement NATS publisher for conflict detection events.

## Phase 4: Integration
- [ ] Create `cmd/cerebellum/main.go` entry point.
- [ ] Create `Dockerfile.cerebellum`.
- [ ] Update `cloudbuild.yaml` with Cerebellum image.
- [ ] Update `terraform/main.tf` to deploy the worker.
