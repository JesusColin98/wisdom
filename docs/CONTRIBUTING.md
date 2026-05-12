# Contributing to Project Wisdom

## Engineering Standards

### 1. Language & Tooling
- **Language:** Go (1.25+).
- **Style:** Adhere strictly to Go idioms (see `google3/cloud/testing/cape/deli/service/pubsub/buganizer/processor/` for internal reference).
- **Linting:** Use `go fmt` and `go vet` before any submission.

### 2. Testing Policy
- **Requirement:** 100% unit test coverage for all `pkg/` modules.
- **Pattern:** Use **Table-Driven Tests** with `t.Run`.
- **Isolation:** Use SQLite in-memory databases for `pkg/cortex` tests to avoid filesystem side-effects.

### 3. Naming Conventions
- **Conciseness:** Prefer `Get()` over `GetNodeByID()`, `Record()` over `RecordMetabolicUsage()`.
- **Exporting:** Only export what is necessary for cross-package orchestration.

### 4. Adding a New Feature
1. **Plan Mode:** Draft a plan in `wisdom/plans/` and have it approved.
2. **Implementation:**
    - If adding a new **Tool**: Implement the `Tool` interface in `pkg/cerebellum/tools`.
    - If adding a new **Logic**: Implement in the relevant "biological" package.
3. **Observability:** Every new execution path MUST have an OTel span.
4. **Validation:** Every new API method MUST have a corresponding JSON Schema.

## Adding New Tools
To add a new capability to Wisdom:
1. Create a struct that implements `Tool` in `pkg/cerebellum/`.
2. Define a clear JSON Schema for parameters.
3. Register the tool in `cmd/wisdom/main.go`:
   ```go
   registry.Register(ToolDefinition{ID: "my_tool", ...}, &MyToolImpl{})
   ```
4. The engine will automatically handle validation, circuit breaking, and async execution.
