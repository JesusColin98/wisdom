# Wisdom Cognitive Runtime Refactoring: Thalamus Gateway

## Objective
Build the `Thalamus Gateway` (Track 02). Thalamus acts as a strict, deterministic gateway that feeds context to the external Gemini LLM and logs its chain of thought. It is completely isolated from the LLM execution itself, functioning purely as a data retriever and formatter to maximize the Token-to-Signal Ratio.

## Architecture

*   **Language**: Go (gRPC)
*   **Service Port**: 50052
*   **Protocol**: gRPC (`thalamus.proto`)
*   **Dependencies**: Acts as a gRPC client to `Cortex` (Port 50051).

## Phased Approach

### Phase 1: Thalamus Definition & Setup
1.  Define the `thalamus.proto` contract (`HydrateContext`, `AuditThought`).
2.  Scaffold the `thalamus` package in `pkg/thalamus`.
3.  Set up the gRPC Server on port `50052`.

### Phase 2: Context Hydration
1.  Implement a gRPC client to communicate with `Cortex`.
2.  Implement `HydrateContext`: Retrieve `Fact` nodes from `Cortex` and format them into dense Markdown.
3.  Write unit tests to verify the deterministic formatting logic.

### Phase 3: Auditing 
1.  Implement `AuditThought`: Receive reasoning traces and asynchronously write them to `Cortex` as `Signal` nodes.

### Phase 4: Integration
1.  Update Terraform to deploy `wisdom-thalamus` to Cloud Run.
2.  Update Cloud Build to build and push the Thalamus image.
