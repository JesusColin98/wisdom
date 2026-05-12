# Memory as a Service (MaaS) Integration Guide

Wisdom is designed as a **Knowledge Runtime**. It provides a unified substrate for both episodic memory (conversations, files) and structured learning (curriculums, mastery levels). This document outlines how external systems can leverage Wisdom as a service.

## 1. Architectural Philosophy: Unified Substrate
We maintain a single Knowledge Graph (**Cortex**) for all types of information. 
- **Memories** (Notes, Chats) are nodes.
- **Learning Concepts** (Chess Theory, English Grammar) are also nodes.
- **User Mastery** is expressed via links (`MASTERED_BY`, `STRUGGLES_WITH`).

By keeping these together, external systems can ask complex questions like: 
*"Show me concepts in the 'Chess' learning path that relate to my meeting notes from yesterday."*

## 2. Connection Patterns

### Option A: Model Context Protocol (MCP)
The preferred way for LLMs (like Gemini, Claude, or custom agents) to connect.
- **Protocol**: JSON-RPC over Standard I/O or SSE.
- **Capabilities**:
    - `recall`: Semantic and graph-based retrieval.
    - `memorize`: Store new nodes/memories.
    - `plan_learning`: Trigger the Proactive Learning Engine for a topic.
- **Implementation**: See `/wisdom/pkg/mcp`.

### Option B: REST API
Ideal for traditional applications, dashboards, or data pipelines.
- **Base URL**: `https://wisdom-engine-[ID].a.run.app`
- **Endpoints**:
    - `POST /chat`: High-level grounded conversation.
    - `POST /cortex/recall`: Low-level context retrieval.
    - `GET /cortex/weaknesses`: Retrieve nodes the user struggles with.
    - `POST /learning/generate`: Create a new learning path from a topic or resource.
- **Specification**: See `openapi.yaml` in the root directory.

## 3. Extending Wisdom
To add new logic without bloating the core:
1. **New Intent**: Add a new classifier intent in `pkg/thalamus/classifier.go`.
2. **Logic Engine**: Implement a new struct in `pkg/thalamus/` (e.g., `LearningEngine`).
3. **API Hook**: Register the endpoint in `cmd/wisdom-api/main.go`.

## 4. Security & Multi-tenancy
- All requests must include a `user_id` or LDAP header.
- The `NamespaceID` in the Cortex ensures isolation between different projects or systems sharing the same instance.
