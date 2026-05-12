# Subsystem: Cortex-Substrate (Storage & Retrieval)

The Cortex is the unified persistence layer for Wisdom, now evolved into a global, horizontally scalable **Knowledge Substrate**.

## Storage Tiering: Facts vs Signals

To optimize for cost and scale, Wisdom implements a multi-engine storage strategy.

### Tier 1: Relational Facts (The 'Enciclopedia')
- **Engine**: Cloud SQL (Postgres) or Cloud Spanner (if global scale is required).
- **Content**: Immutable truths, entity definitions, and cross-referenced categories (e.g., "Chess Theory", "History of Rome").
- **Structure**: Strongly typed tables ensuring referential integrity.

### Tier 2: Non-Structured Signals (Memory Mesh)
- **Engine**: **Google Cloud Firestore**.
- **Content**: Ephemeral user data, chat transcripts, and high-frequency "Signals" (e.g., current focus, temporary notes).
- **Cost-Efficiency**: Firestore is the optimal choice for a single user (near-zero cost initial scale) and handles unstructured JSON naturally.
- **Flexibility**: All Signal nodes are stored as extensible JSON documents, allowing the schema to evolve without migrations.

### Tier 3: Edge Cache (Bucle Neural)
- **Engine**: **SQLite**.
- **Usage**: Only for the **Chat Service**.
- **Purpose**: Store high-frequency "trash" data and transient context to minimize network latency during active conversations.
- **Cleanup Strategy**: Managed via the **REM Cycle**. Logic includes an `abandoned_session` timeout (e.g., 2 hours of inactivity) and a daily summary task.

### Tier 4: Reference Layer (Original Sources)
- **Content**: URI Pointers (Google Drive, HTTPS, arXiv).
- **Logic**: No raw storage of source PDFs. Cortex stores the extracted *Knowledge* and the *Link* to the original source.
- **Ingestion Buffer**: A TTL-restricted GCS bucket for files currently being scanned. Files are auto-deleted after 24h or upon extraction success.

## Interface (gRPC MaaS)
...

```protobuf
service Cortex {
  rpc Memorize(IngestRequest) returns (NodeID);
  rpc Recall(RecallRequest) returns (CognitionResponse);
  rpc QueryHechos(FactRequest) returns (FactList);
}

message IngestRequest {
  string content = 1;
  string author = 2;
  map<string, string> metadata = 3;
  
  enum Stratum {
    HOT = 0;  // SQLite / Superficial
    COLD = 1; // Graph / Deep
  }
  Stratum target_stratum = 4;
  
  bool is_immutable = 5; // True = Fact, False = Signal
  
  enum Consistency {
    EVENTUAL = 0;
    STRONG = 1;
  }
  Consistency sync_requirement = 6;
}
```

## Edge Scenarios
- **Conflicting Facts**: When two "Hechos" contradict, Cortex creates a `CONTRADICTS` link and triggers a `Neural-Socratic` task for the user to resolve.
- **Signal Decay**: Signals with low `ImpactScore` are auto-pruned after 30 days of inactivity.
