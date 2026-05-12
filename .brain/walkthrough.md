# Technical Walkthrough: Wisdom Refactoring

## [2026-05-12] Initialization
*   Activated Antigravity workflow.
*   Initialized `.brain` directory.
*   Drafted `implementation_plan.md` based on `wisdom/brain/MASTER_PLAN.md` and `wisdom/brain/TRACK_01_CORTEX.md`.
*   Drafted initial `task.md`.
*   **Substrate Definition:** Created `cortex.proto` defining the gRPC interface (`IngestRequest`, `RecallRequest`, etc.).
*   **Database Schema:** Created `schema_v2.sql` containing the minimal two-table graph schema (`nodes` and `edges`) using `jsonb` and enums.
*   **Go Models:** Updated `wisdom/wisdom/pkg/cortex/node.go` to match the new schema exactly.

## Next Steps
*   Run the database initialization script (`schema_v2.sql`) on a local PostgreSQL instance (or wait for the Cloud SQL instance to be provisioned).
*   Run `terraform apply` in the `terraform/` directory.
*   Push code to GitHub to trigger the newly updated Cloud Build pipeline.
*   Proceed to `TRACK_02_THALAMUS.md` to build the Gateway layer.

## [2026-05-12] Infrastructure & CI/CD Setup
*   **Terraform:** Replaced the GCS FUSE SQLite persistence with a Cloud SQL PostgreSQL instance (`db-f1-micro`) in `terraform/main.tf`. Updated the Cloud Run deployment from `wisdom-unified` to the new gRPC `wisdom-cortex`.
*   **Entry Point:** Scaffolded `wisdom/cmd/cortex/main.go` to serve the gRPC application over TCP (port 50051) using the `DB_CONN_STRING` provided by Secret Manager.
*   **Docker:** Created `wisdom/Dockerfile.cortex` which installs `protobuf-compiler` and the Go plugins during the builder stage. This circumvents local environment issues by ensuring code generation happens deterministically in CI.
*   **Cloud Build:** Updated the root `cloudbuild.yaml` to run `protoc` directly, build the `wisdom-cortex` image, build the `wisdom-chat` image, and push them to Artifact Registry. Removed the obsolete nested `wisdom/cloudbuild.yaml`.
