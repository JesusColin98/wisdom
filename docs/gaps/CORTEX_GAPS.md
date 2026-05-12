# Cortex Substrate Gaps and Next Steps

This document outlines the gaps, future technical debt, and missing features identified during the refactoring of the Cortex Substrate (Track 01) to the Universal Graph Schema.

## 1. Protobuf Compilation Dependencies
- **Issue**: The current local environment lacks the `protoc` compiler. We temporarily used `v1/stubs.go` to mock the generated interfaces to allow `go build` to pass.
- **Resolution**: Once the Cloud Build pipeline runs, it will compile the actual protobufs. However, for local development, `protoc` and the `protoc-gen-go` / `protoc-gen-go-grpc` plugins must be installed on developers' machines.
- **Action Item**: Add `protoc` installation to a `scripts/setup_dev_env.sh` script.

## 2. Advanced JSONB Querying (GIN Operators)
- **Issue**: The current implementation of `QueryFacts` uses a basic `@>` containment operator for filtering via JSONB payload.
- **Resolution**: While functional for exact metadata matches, a full graph database often needs deeper JSON path searches, regex matching inside JSON, or integration with `pgvector` for semantic search on unstructured parts of the payload.
- **Action Item**: Expand the `FactRequest` protobuf definition to support more complex query structures beyond a simple `map<string, string>`.

## 3. Node Neighbor Fetching in `Recall`
- **Issue**: The `Recall` function in `server.go` currently fetches the center node and its adjacent edges (In/Out), but it returns an empty list for the actual `Neighbors` (the instantiated `Node` structs at the end of those edges) to save an additional DB roundtrip.
- **Resolution**: Implement a batched `SELECT ... FROM nodes WHERE id IN (...)` to populate the `Neighbors` array in the `CognitionResponse`.
- **Action Item**: Add batched neighbor fetching to `postgres_engine.go` inside the `Recall` method.

## 4. Terraform Provider State
- **Issue**: Terraform state management is not explicitly defined to use a remote backend (like a GCS bucket). Currently, state would be local, which is bad for teamwork.
- **Resolution**: Configure a `backend "gcs"` block in `main.tf`.

## 5. Security & Authentication
- **Issue**: The `wisdom-cortex` gRPC service in Cloud Run is currently marked as `allUsers` (public) in Terraform for easy testing.
- **Resolution**: gRPC services exposing raw database substrates should be heavily protected. It should only be invokable by other services in the GCP project (like Thalamus) using IAM, or by clients presenting a valid JWT.
- **Action Item**: Change `roles/run.invoker` to a specific internal service account in `main.tf` before pushing to production.
