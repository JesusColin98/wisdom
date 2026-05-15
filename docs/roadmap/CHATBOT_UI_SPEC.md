# Chatbot UI Specification: The Conversational Interface

The Wisdom Chat UI is the **primary user entry point** to the entire Cognitive Runtime. It is a conversational interface that sits in front of the ADK Router, making the Mix of Experts system accessible without requiring the user to understand the underlying architecture.

> **Key Principle:** The user should feel they are talking to one intelligent system. The agent routing, memory retrieval, and tool calls must happen transparently behind the scenes.

---

## 1. Session Lifecycle

A conversation follows a strict lifecycle managed jointly by the Chat UI (frontend) and the `chat_service` backend:

```
INITIATED → ACTIVE → IDLE (>20min) → ABANDONED (>2h) → DISTILLED (REM)
```

### 1.1 Session Creation
- **Trigger:** User opens the Chat view or sends the first message.
- **Action:** Frontend calls `POST /v1/chat/session` on the `chat_service`.
- **Response:** Returns a `session_id` (UUID) stored in memory for the duration.
- **Persistence:** The `session_id` is stored in `SQLite` (Edge Cache, Tier 3 of Cortex) for low-latency access during the active conversation.

### 1.2 Message Streaming
- **Transport:** WebSocket connection to `WSS /v1/chat/stream/{session_id}`.
- **Protocol:** Server-Sent Events (SSE) as a fallback if WebSocket is unavailable.

### 1.3 Session Termination & REM
- **Explicit:** User clicks "End Session" → triggers `POST /v1/chat/session/{id}/close`.
- **Implicit:** After the `ABANDONED` timeout, the `REMService` auto-triggers distillation.
- **Outcome:** High-signal facts from the conversation are promoted to Cortex (Tier 1) and the Vertex AI Memory Bank; the SQLite cache is purged.

---

## 2. WebSocket Message Schema

All messages conform to a strict JSON envelope to allow the frontend to handle different event types.

### 2.1 Outbound (Client → Server): User Message
```json
{
  "type": "USER_MESSAGE",
  "session_id": "uuid-abc123",
  "payload": {
    "text": "Analyze my last chess game",
    "voice_transcript": false,
    "attachments": []
  }
}
```

### 2.2 Inbound (Server → Client): Streamed Response Chunk
```json
{
  "type": "AGENT_CHUNK",
  "session_id": "uuid-abc123",
  "payload": {
    "agent": "Chess_Expert",
    "chunk": "Based on your Caro-Kann game from Tuesday...",
    "is_final": false
  }
}
```

### 2.3 Inbound: Routing Event (Transparency)
```json
{
  "type": "ROUTING_EVENT",
  "payload": {
    "classified_intent": "chess_analysis",
    "routed_to": "Chess_Expert",
    "confidence": 0.97,
    "memory_retrieved": true
  }
}
```
*This event triggers the Agent Identity Indicator in the UI (see Section 3).*

### 2.4 Inbound: Tool Call Event (Inline Action Prompt)
```json
{
  "type": "TOOL_CALL",
  "payload": {
    "tool": "create_note",
    "preview": {
      "title": "Caro-Kann: My Weaknesses",
      "tags": ["#chess/openings", "#analysis"],
      "destination": "00_Inbox/",
      "content_preview": "## Key Mistakes\n- ..."
    },
    "action_required": true
  }
}
```
*Triggers an inline confirmation card in the chat (see Section 4).*

### 2.5 Inbound: Error Event
```json
{
  "type": "ERROR",
  "payload": {
    "code": "AGENT_TIMEOUT",
    "message": "Chess Expert did not respond within 10s. Retrying...",
    "retry": true
  }
}
```

---

## 3. Agent Identity Indicator

The UI must always show the **active agent** to build user trust and mental model clarity.

- **Position:** Persistent pill/badge in the chat header, below the session title.
- **States:**
  - `🧠 Routing...` — while the ADK Router classifies the intent.
  - `♟️ Chess Expert` — once routed to the Chess Expert.
  - `💰 Finance Expert` — routed to Finance/Fibras Expert.
  - `📚 Learning Expert` — routed to General Learning.
  - `💻 Tech Expert` — routed to Technology Expert.
  - `🔄 Switching agents...` — when a mid-conversation intent change occurs.
- **Animation:** Subtle pulse animation while streaming is active.
- **Click behavior:** Clicking the badge opens a tooltip explaining what this expert handles and which memories it loaded.

---

## 4. Inline Action Cards (Confirmation UI)

When an agent wants to **create a note, generate Anki cards, or perform a tool call**, it must not do so silently. The chat renders an **Inline Action Card** that pauses execution until the user confirms.

### 4.1 "Save as Obsidian Note" Card
```
┌─────────────────────────────────────────────┐
│ 📝 New note ready to save                   │
│ Title: Caro-Kann: My Weaknesses             │
│ Destination: 00_Inbox/                      │
│ Tags: #chess/openings, #analysis            │
│ Links: [[Caro-Kann Defense]], [[Tactics MOC]]│
│                                             │
│ [Preview full note] [Save to Vault] [Discard]│
└─────────────────────────────────────────────┘
```

### 4.2 "Create Anki Cards" Card
```
┌─────────────────────────────────────────────┐
│ 🃏 3 Anki cards ready                       │
│ Type: Wisdom-Basic (×2), Wisdom-Cloze (×1) │
│ Deck: Chess::Openings::Caro-Kann            │
│                                             │
│ [Review cards] [Send to Anki] [Discard]     │
└─────────────────────────────────────────────┘
```

### 4.3 "Search Vault" Result Card
When the user asks something and the system finds a related existing note:
```
┌─────────────────────────────────────────────┐
│ 🔍 Found in your vault                      │
│ [[Caro-Kann Defense]] — 87% match           │
│ "The Caro-Kann is characterized by..."      │
│                                             │
│ [Open note] [Update with new info] [Ignore] │
└─────────────────────────────────────────────┘
```

---

## 5. Dual-Mode Interface (Text / Voice)

The Chat UI operates in two distinct modes controlled by a **mode toggle pill** (`Text | Voice`). Voice is the **default mode** — the interface is designed voice-first, with text as the secondary fallback.

> **Vault constraint:** The agent's knowledge is exclusively sourced from the user's Obsidian vault via the `recall_wisdom` tool. The agent always calls this tool before answering any question.

### 5.1 Mode State Machine

```
                 ┌─────────┐
  default ──────►│  VOICE  │◄──── user clicks "Voice"
                 └────┬────┘
                      │ user clicks "Text"
                      │ (mic auto-stops if was listening)
                      ▼
                 ┌─────────┐
                 │  TEXT   │◄──── user clicks "Text"
                 └─────────┘
```

- `mode` state: `'voice' | 'text'`
- Switching from **Voice → Text** calls `stopMic()` automatically
- Switching from **Text → Voice** does **not** auto-start the mic (user must tap)

### 5.2 Voice Mode Layout

```
┌──────────────────────────────────────┐
│  ChatHeader                          │
├──────────────────────────────────────┤
│                                      │
│         AuraOrb  (50vh)              │  ← full viewport height
│    reactive to AI audio frequency    │
│                                      │
│  ─── waveform bars (AI speaking) ──  │
├──────────────────────────────────────┤
│  ChatMessageList (scrollable)        │
├──────────────────────────────────────┤
│  [ Text | ● Voice ]  ← toggle pill   │
│                                      │
│          ◉  (68px hero mic)          │  ← pulsing indigo rings when listening
│       "Tap to speak" / "Listening…"  │
└──────────────────────────────────────┘
```

### 5.3 Text Mode Layout

```
┌──────────────────────────────────────┐
│  ChatHeader                          │
├──────────────────────────────────────┤
│  AuraOrb ambient strip (80px)        │  ← collapses, scale(0.28), opacity 40%
│  "AI Ready" / "Speaking…" label      │
├──────────────────────────────────────┤
│  ChatMessageList (flex-1, scroll)    │
├──────────────────────────────────────┤
│  [ ● Text | Voice ]  ← toggle pill   │
│  [🎤] [ Ask Wisdom about your vault… ] [→] │
└──────────────────────────────────────┘
```

### 5.4 AuraOrb Transition

The AuraOrb viewport uses a CSS `cubic-bezier(0.34, 1.56, 0.64, 1)` spring transition (no Framer Motion dependency) to animate between:

| Property    | Voice mode | Text mode |
|---|---|---|
| `height`    | `50vh`     | `80px`    |
| `min-height`| `260px`    | `80px`    |
| Orb `scale` | `scale(1)` | `scale(0.28)` |
| Orb `opacity` | `1`      | `0.4`     |
| Transition duration | `450ms` | `450ms` |

### 5.5 Mic Pipeline

1. `navigator.mediaDevices.getUserMedia({ audio: { sampleRate: 16000, channelCount: 1 } })`
2. `AudioWorkletNode('pcm-processor')` loaded from `/worklets/pcm-processor.js`
3. Raw PCM frames → `WebSocket.send(ArrayBuffer)` → `chat_service /ws/chat`
4. AI audio response arrives as binary `Blob` → decoded as PCM Int16 → `AudioBufferSourceNode` connected to `AnalyserNode`
5. `AnalyserNode` ref is shared with `ChatAuraOrb` for frequency-reactive animation

### 5.6 Component File Map

| Component | File | Responsibility |
|---|---|---|
| `ChatHeader` | `chat/ChatHeader.jsx` | Title, connection status, namespace badge |
| `ChatAuraOrb` | `chat/ChatAuraOrb.jsx` | Reactive orb animation (indigo theme) |
| `ChatMessageList` | `chat/ChatMessageList.jsx` | Scrollable message thread |
| `ChatControlPanel` | `chat/ChatControlPanel.jsx` | Mode pill + voice/text input |
| `ChatView` | `ChatView.jsx` | Orchestrator — WebSocket, audio, state |

### 5.7 Degradation

- If microphone permission is denied → mic button shows error state; text mode is auto-activated
- If WebSocket drops → UI shows "Cortex Disconnected" in header; auto-reconnect not implemented (user must refresh)
- If `recall_wisdom` fails → agent responds with a fallback message indicating Cortex is unreachable

---

## 6. Chat History & Continuity

- **Within session:** Full chat history visible in the current scroll view.
- **Across sessions:** The last 5 sessions are listed in a collapsible sidebar. Clicking a past session shows its transcript (read-only after REM distillation).
- **Memory continuity:** The user can ask "What did we discuss last time about my chess game?" — the ADK Router retrieves facts from the Vertex AI Memory Bank (scoped to `Chess_Expert`) and injects them into the new session.

---

## 7. Mobile Layout Requirements

The Chat UI must be **fully functional on mobile browsers**.

- **Layout:** Single-column. Chat history takes 80% of viewport height.
- **Input:** Sticky bottom bar with text input + microphone button.
- **Inline cards:** Stack vertically with full-width action buttons.
- **Agent indicator:** Compact pill (icon only, no text) on mobile. Full label on desktop.
- **Voice:** Available on mobile as the primary input method.
- **Responsive breakpoints:** `≤768px` = mobile layout, `>768px` = desktop layout.

---

## 8. API Contract with `chat_service`

The Chat UI communicates **only** with the `chat_service` Python microservice. It never calls the ADK Router or Cortex directly.

| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/v1/chat/session` | Create a new session |
| `WS` | `/v1/chat/stream/{session_id}` | Bidirectional streaming |
| `POST` | `/v1/chat/session/{id}/close` | Explicitly end a session |
| `GET` | `/v1/chat/sessions` | List past sessions (last 5) |
| `GET` | `/v1/chat/session/{id}/transcript` | Get distilled transcript |

- **Auth:** JWT via `Authorization: Bearer <token>` header.
- **CORS:** Restricted to the Portal's origin only.
