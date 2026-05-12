# TRACK 02: Thalamus Gateway

## Objective
Build the strict, deterministic gateway that feeds context to the external Gemini LLM and logs its chain of thought.

## Tasks

### 1. Thalamus gRPC Service Definition
- [ ] Define `Thalamus` service in `thalamus.proto`.
- [ ] Create gRPC server running on port `50052`.

### 2. Context Hydration (HydrateContext)
- [ ] Connect Thalamus as a gRPC client to the `Cortex` service (Track 01).
- [ ] Implement logic: Accept a `Query` (string + token budget).
- [ ] Query `Cortex` using exact text match, regex, or basic vector similarity to find relevant Nodes.
- [ ] Format the retrieved nodes into a dense, highly optimized Markdown string (maximizing Token-to-Signal Ratio).
- [ ] Return the `ContextPayload` without invoking any LLMs.

### 3. Auditing (AuditThought)
- [ ] Implement an endpoint that receives the streaming output / reasoning traces from Gemini.
- [ ] Save these logs asynchronously (e.g., via NATS or writing directly to a `Signal` node in Cortex) for future analysis.

## Acceptance Criteria
- `HydrateContext` returns formatted Markdown in under 50ms.
- The service acts purely as a data formatter and router.
- No LLM API keys are required to run this service.