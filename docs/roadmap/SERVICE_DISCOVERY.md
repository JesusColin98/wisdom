# Subsystem: Wisdom-Discovery (The Cognitive Mesh)

Wisdom uses **NATS** as the central nervous system for both Service Discovery and Asynchronous Event Broadcasting.

## 1. Discovery as a Service (DaaS)
For a dynamic environment like Cloud Run/GKE, I recommend **NATS Service Discovery** over static IPs.

- **Mechanism**: Each microservice (Researcher, Cortex, Trace) "advertises" its capabilities on a NATS subject.
- **Service API**: We use `nats.go`'s `micro` package.
- **Benefit**: The Thalamus doesn't need to know the IP of the Researcher. It simply requests `RESEARCH.investigate` and NATS routes it to the fastest available instance.

## 2. Asynchronous Event Bus (NATS JetStream)
Certain cognitive tasks are long-running and shouldn't block the gRPC thread.

| Event Subject | Publisher | Subscriber | Action |
| :--- | :--- | :--- | :--- |
| `KNOWLEDGE.ingested` | Researcher | Cortex | Trigger indexing & embedding. |
| `LEARNING.path_created` | Curriculum | Trace | Initialize user progress markers. |
| `USER.struggle_detected` | Chat | Curriculum | Re-calculate learning priority. |

## 3. Storage Abstraction (Blob-Pointer Pattern)
To handle the "Enciclopedia" of books and large documents:
1.  **GCS (The Body)**: Raw data (PDFs, Audio) is stored in Google Cloud Storage.
2.  **Spanner (The Mind)**: Stores the metadata, extracted facts, and a URI link to the GCS object.
3.  **Discovery**: When the `Researcher` finds a new book, it uploads to GCS and broadcasts a `KNOWLEDGE.ingested` message with the URI.

## 4. Implementation Choice: NATS JetStream
Why JetStream?
- **Persistence**: If the `Cortex` is down, the message from the `Researcher` isn't lost.
- **Discovery**: Built-in support for service health checks.
- **Speed**: Memory-first architecture that matches our "Neural" performance goals.
