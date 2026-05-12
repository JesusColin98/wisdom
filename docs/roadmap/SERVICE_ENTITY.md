# Subsystem: Wisdom-Entity-Dictionary (Ontology Service)

The Entity Dictionary is the specialized component responsible for identifying and contextualizing the "Who," "What," and "How" within the knowledge substrate.

## Use Cases
- **Entity Extraction**: Scan ingested Markdown for symbols (@, #) and map them to the global registry.
- **Attribute Profiling**: Maintain a list of attributes for an entity (e.g., @ElonMusk: profession=Engineer, interests=[Mars, AI]).
- **Relationship Recognition**: Identify how two entities are connected (e.g., @Jesus WORKS_AT @Google).

## Standardization: Obsidian + Markdown Symbols
...

| Symbol | Meaning | Example |
| :--- | :--- | :--- |
| `@` | Person / Org | `Met with @SamMorgan today.` |
| `#` | Topic / Category | `Researching #LLM performance.` |
| `[[ ]]` | Node Link | `See [[Backpropagation]] for more.` |
| `[::]` | Attribute | `[level:: Expert]` |
| `^` | Block ID | `This is a specific fact ^block123` |

## Interface

```protobuf
service EntityDictionary {
  rpc ResolveEntity(EntityRequest) returns (EntityProfile);
  rpc TagContent(ContentRequest) returns (TaggedContent);
}

message EntityRequest {
  string symbol_text = 1;
  string user_id = 2;
  
  enum Scope {
    PRIVATE = 0;
    TEAM = 1;
    GLOBAL = 2;
  }
  Scope visibility = 3;
}
```

## Edge Scenarios
- **Granular Referencing**: When a block ID (`^`) is used, the Dictionary creates a sub-node in Cortex linked via `PART_OF` to the parent document.
- **Ambiguous @Recognition**: If `@Jesus` could refer to multiple users, the service returns a `RESOLUTION_REQUIRED` signal to the Thalamus.
