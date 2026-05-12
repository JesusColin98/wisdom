# Task Tracker: Thalamus Gateway

## Phase 1: Thalamus Definition & Setup
- [x] Create `thalamus.proto` defining `HydrateContext` and `AuditThought`.
- [x] Scaffold `server.go` for Thalamus on port 50052.

## Phase 2: Context Hydration
- [x] Implement Cortex gRPC client within Thalamus.
- [x] Implement `HydrateContext` logic (fetch and format Markdown).
- [x] Write unit tests for context formatting.

## Phase 3: Auditing
- [x] Implement `AuditThought` logic (save traces as Signals).

## Phase 4: Integration
- [x] Create `cmd/thalamus/main.go` entry point.
- [x] Create `Dockerfile.thalamus`.
- [x] Add `wisdom-thalamus` Cloud Run service to `terraform/main.tf` (Needs Cortex URL).
- [x] Add `wisdom-thalamus` build step to `cloudbuild.yaml`.
