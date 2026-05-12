# Ecosystem Contracts: Protobuf & CloudEvents

To allow independent teams (or agents) to build components in parallel, all inter-service communication MUST adhere to these locked contracts.

## 1. Asynchronous Events (NATS JetStream)
We use the [CloudEvents](https://cloudevents.io/) specification for all NATS messages.

### Subject: `wisdom.knowledge.ingested`
Fired by `Researcher` when new data is acquired. `Cortex` and `Cerebellum` listen to this.
```json
{
  "specversion": "1.0",
  "type": "wisdom.knowledge.ingested",
  "source": "/researcher/blog-crawler",
  "id": "A234-1234-1234",
  "time": "2026-05-12T00:00:00Z",
  "datacontenttype": "application/json",
  "data": {
    "title": "New Chess Strategies",
    "markdown_content": "# Intro\n...",
    "source_url": "https://example.com/chess",
    "suggested_tags": ["#chess", "#strategy"]
  }
}
```

### Subject: `wisdom.memory.conflict_detected`
Fired by `Cerebellum` when a contradiction is found. Frontend/Portal listens to prompt the user.
```json
{
  "specversion": "1.0",
  "type": "wisdom.memory.conflict_detected",
  "source": "/cerebellum/integrity-checker",
  "data": {
    "winning_node_id": "uuid-1",
    "losing_node_id": "uuid-2",
    "confidence_delta": 0.6
  }
}
```

## 2. Synchronous APIs (gRPC)
*(See individual SERVICE_*.md files for specific rpc definitions)*.
All gRPC services MUST implement the standard Health checking protocol: `grpc.health.v1.Health`.
