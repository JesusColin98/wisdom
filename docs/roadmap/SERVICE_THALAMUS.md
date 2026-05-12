# Subsystem: Wisdom-Thalamus (Memory Gateway & Auditor)

## Philosophy: "Wisdom is Memory, Gemini is the Brain"
The Thalamus is **not** an active LLM agent. It does not waste tokens generating responses. It is a high-performance **Gateway, Context Injector, and Auditor**. Its job is to provide Gemini (or any external LLM) with the highest Token-to-Signal Ratio (TSR) context possible, and to passively record Gemini's Chain of Thought.

## Use Cases
1. **Context Hydration**: An external LLM requests context for a query. Thalamus searches Cortex, packages the data into a dense, token-efficient Markdown block, and returns it.
2. **Chain of Thought (CoT) Auditing**: As Gemini reasons, it streams its steps. Thalamus logs this CoT for future analysis and learning pattern extraction.
3. **Intent Routing**: Deterministic routing of requests (e.g., "Get me my chess weaknesses") directly to the `Trace` or `Curriculum` service.

## Architecture
- **Strictly Deterministic**: Written in Go. Uses exact matching, vector similarity, and graph traversals. Zero LLM calls initiated internally unless explicitly requested for a specific batch transformation.
- **Auditor Pattern**: Acts as a middleware interceptor.

## Interface (gRPC)
```protobuf
service Thalamus {
  // Inject context without thinking
  rpc HydrateContext(Query) returns (ContextPayload);
  // Passively audit Gemini's reasoning
  rpc AuditThought(ChainOfThought) returns (Ack);
}

message Query {
  string raw_text = 1;
  int32 token_budget = 2;
}

message ContextPayload {
  string markdown_context = 1;
  repeated string entity_tags = 2;
}
```
