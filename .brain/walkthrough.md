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
*   Run the database initialization script (`schema_v2.sql`) on a local PostgreSQL instance.
*   Once `protoc` is available in the build pipeline, remove the stubs in `v1/stubs.go` and generate the real protobuf bindings.
*   Proceed to `TRACK_02_THALAMUS.md` to build the Gateway layer.
