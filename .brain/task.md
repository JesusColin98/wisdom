# Task Tracker: Thalamus Gateway

## Phase 1: Thalamus Definition & Setup
- [ ] Create `thalamus.proto` defining `HydrateContext` and `AuditThought`.
- [ ] Scaffold `server.go` for Thalamus on port 50052.

## Phase 2: Context Hydration
- [ ] Implement Cortex gRPC client within Thalamus.
- [ ] Implement `HydrateContext` logic (fetch and format Markdown).
- [ ] Write unit tests for context formatting.

## Phase 3: Auditing
- [ ] Implement `AuditThought` logic (save traces as Signals).

## Phase 4: Integration
- [ ] Add `wisdom-thalamus` Cloud Run service to `terraform/main.tf`.
- [ ] Add `wisdom-thalamus` build step to `cloudbuild.yaml`.
- [ ] Create `Dockerfile.thalamus`.
