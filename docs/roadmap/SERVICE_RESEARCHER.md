# Subsystem: Wisdom-Researcher (Investigation Service)

This service is the primary "ingestor of the world." It prioritizes deterministic acquisition over LLM generation to ensure factual groundedness.

## Use Cases
- **Topic Deep Dive**: Given "Modern Chess Openings," find and download relevant PDFs, blog posts, and latest tournament news.
- **Resource Extraction**: Extract clean Markdown text from a provided URL or PDF file.
- **News Monitoring**: Subscribe to a topic and receive daily updates from RSS/News APIs.

## Architecture & Modules

### 1. The Resource Analyst (Ephemeral Librarian)
- **Policy**: Wisdom **DOES NOT** persist raw PDFs or large files long-term.
- **Process**:
    1.  **Reference Storage**: Store the `source_url` (Public or Google Drive URI).
    2.  **Temporary Ingestion**: Download the file to a transient GCS sandbox.
    3.  **Knowledge Extraction**: Use LLM/OCR to convert key insights into structured Markdown nodes.
    4.  **Immediate Purge**: Delete the source file from the sandbox after extraction.
- **Output**: Structured knowledge in Cortex with a pointer to your original URI for future human reference.

### 2. Blog-Crawler (The Expert Scout)
- **Tooling**: Go-based RSS/Atom aggregator.
- **Process**: Crawl known expert blogs -> Filter by topic -> Extract content using clean-text algorithms (e.g., Boilerpipe).

### 3. News-Stream (The Trend Sentinel)
- **Tooling**: Integration with NewsAPI, Google News, or specialized industry feeds.
- **Process**: Real-time ingest of high-signal headlines -> Link to existing graph nodes.

## Service Interface (gRPC/REST)

```protobuf
service Researcher {
  rpc InvestigateTopic(TopicRequest) returns (stream ResearchSignal);
  rpc ExtractResource(ResourceRequest) returns (ResourceContent);
}

message TopicRequest {
  string topic = 1;
  repeated string sources = 2;
  int32 metabolic_limit = 3; // Limit research based on token/compute cost
}

message ResearchSignal {
  string source_url = 1;
  string content_md = 2;
  float confidence = 3;
  float tsr_estimate = 4; // Token-to-Signal Ratio estimate
  map<string, string> metadata = 5; 
}
```

## Integration with Cerebellum (GNN)
While the Researcher is deterministic, its output is pushed to the **Cerebellum** for:
- **Topology Mapping**: The GNN identifies where this new research fits in the global graph.
- **Link Prediction**: Auto-generating `ASSOCIATED_WITH` links between researched facts.

## Edge Scenarios
- **Paywalled Content**: The Researcher returns a `SIGNAL_UNAVAILABLE` with a request for user credentials.
- **Low Signal (TSR < 0.2)**: The service aborts research to preserve the metabolic budget.
