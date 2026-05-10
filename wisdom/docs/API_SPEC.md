# API Specification: Project Wisdom

This document defines the interface for consuming Wisdom features via the Thalamus layer.

## 1. REST Endpoints

### 1.1 Configuration (`/config`)
- **GET:** Returns current `WisdomConfig`.
- **POST:** Updates runtime parameters (Hops, Uncertainty Threshold, Token Budget).

### 1.2 Lineage & Hierarchy (`/cortex/lineage`)
- **GET:** `?node_id=XYZ&direction=UP|DOWN`
- **Logic:** Returns recursive ancestral or descendant nodes for Tree-RAG visualization.

### 1.3 Context Recall (`/thalamus/recall`)
- **POST:** Fetches prioritized knowledge nodes based on intent.
- **Payload:**
  ```json
  {
    "user_id": "string",
    "query": "string",
    "seeds": ["node_id_1", "alias_2"],
    "uncertainty": 0.8
  }
  ```

## 2. WebSocket: Dopamine Loop (Metabolism)

The frontend should subscribe to `/ws/metabolism` to receive real-time TSR (Token-to-Signal Ratio) events.

### 2.1 TSR Alert
- **Message:**
  ```json
  {
    "type": "TSR_UPDATE",
    "tsr": 0.85,
    "efficiency": "HIGH",
    "message": "Metabolic efficiency optimized. Depth reduced for budget conservation."
  }
  ```

## 3. WebSocket: Hallucination Guard

When a chatbot generates text, it can validate chunks in real-time via the WebSocket connection.

### 3.1 Validation Event
- **Message:** `{"type": "VALIDATE", "text": "The Sam Morgan role history..."}`
- **Response:** `{"type": "VALIDATION_RESULT", "is_grounded": true, "warnings": []}`
