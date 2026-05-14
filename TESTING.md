# Wisdom Cognitive Runtime — Manual Testing Guide

> Run these checks **in order** after `docker-compose -f docker-compose.dev.yml up`.
> Each section maps to a Phase of the refactoring. A ✅ means pass, ❌ means needs fixing.

---

## Prerequisites

```bash
# Start all services
cd c:\Users\jesus\wisdom\wisdom
docker-compose -f docker-compose.dev.yml up -d

# Verify all containers are running
docker-compose -f docker-compose.dev.yml ps
```

Expected: 8+ containers in `Up` state (cortex, thalamus, mastery, researcher, curriculum, integrations, entity, adk-router).

---

## Phase 1 — Go gRPC Mesh Health Checks

### 1.1 Cortex (port 50051)
```bash
curl http://localhost:50051/health
# Expected: {"status":"ok"} or gRPC SERVING response
```

### 1.2 Mastery (port 50053)
```bash
curl http://localhost:50053/health
```

### 1.3 Integrations (port 50056)
```bash
curl http://localhost:50056/health
```

### 1.4 All services respond
```bash
for port in 50051 50052 50053 50054 50055 50056 50057; do
  echo -n "Port $port: "
  curl -s --max-time 2 http://localhost:$port/health || echo "OFFLINE"
done
```

---

## Phase 2 — Integrations MCP Bridge

### 2.1 Create a note (Obsidian bridge)
```bash
curl -s -X POST http://localhost:50056/api/v1/integrations/note \
  -H "Content-Type: application/json" \
  -d '{
    "agent_name": "Tech_Expert",
    "user_id": "default",
    "target_path": "Tech/Test/hello-world.md",
    "content": "# Hello World\n\nThis is a test note.",
    "metadata": {
      "title": "Hello World",
      "tags": ["#test"],
      "mastery_score": 0.5
    }
  }'
```
**Expected:** `{"status": "SYNCED"}` or `{"status": "QUEUED"}` (if Obsidian is offline — this is OK).

### 2.2 Create a flashcard (Anki bridge)
```bash
curl -s -X POST http://localhost:50056/api/v1/integrations/card \
  -H "Content-Type: application/json" \
  -d '{
    "agent_name": "Tech_Expert",
    "user_id": "default",
    "deck_name": "Wisdom::Tech",
    "card_type": "BASIC",
    "front": "What is the time complexity of binary search?",
    "back": "O(log n)",
    "tags": ["Wisdom::Tech"]
  }'
```
**Expected:** `{"status": "SYNCED"}` or `{"status": "QUEUED"}`.

### 2.3 Check PENDING_SYNC queue (items queued when apps offline)
```bash
curl -s "http://localhost:50056/api/v1/integrations/queue?user_id=default" | python -m json.tool
```
**Expected:** JSON with `items` array (can be empty if Obsidian/Anki are running).

### 2.4 MCP Obsidian server (if running locally)
```bash
# Start the MCP server first:
# cd mcp-servers/obsidian && npm install && node src/index.js
curl -s http://localhost:3333/health
```
**Expected:** `{"status":"ok","service":"obsidian-mcp"}`

### 2.5 MCP Anki server (if running locally)
```bash
# cd mcp-servers/anki && npm install && node src/index.js
curl -s http://localhost:3334/health
```
**Expected:** `{"status":"ok","service":"anki-mcp"}`

---

## Phase 3 — ADK Router (port 8081)

### 3.1 Health check
```bash
curl -s http://localhost:8081/health
```
**Expected:** `{"status":"ok","service":"wisdom-adk-router"}`

### 3.2 List domain configuration
```bash
curl -s http://localhost:8081/domains | python -m json.tool
```
**Expected:** JSON with `domains` array containing CHESS, FINANCE, LANGUAGE, TECH, GENERAL.

### 3.3 Route a chess question
```bash
curl -s -X POST http://localhost:8081/route \
  -H "Content-Type: application/json" \
  -d '{
    "input": "How does the Caro-Kann defense work against e4?",
    "user_id": "default"
  }' | python -m json.tool
```
**Expected:**
```json
{
  "domain": "CHESS",
  "agent": "Chess_Expert",
  "confidence": ...,
  "response": "...",
  "elapsed_ms": ...
}
```

### 3.4 Route a finance question
```bash
curl -s -X POST http://localhost:8081/route \
  -H "Content-Type: application/json" \
  -d '{
    "input": "What is the dividend yield of Fibra Uno?",
    "user_id": "default"
  }' | python -m json.tool
```
**Expected:** `"domain": "FINANCE"`, `"agent": "Finance_Expert"`

### 3.5 Route a tech question
```bash
curl -s -X POST http://localhost:8081/route \
  -H "Content-Type: application/json" \
  -d '{
    "input": "Explain the time complexity of QuickSort",
    "user_id": "default"
  }' | python -m json.tool
```
**Expected:** `"domain": "TECH"`, `"agent": "Tech_Expert"`

### 3.6 Route a language question
```bash
curl -s -X POST http://localhost:8081/route \
  -H "Content-Type: application/json" \
  -d '{
    "input": "How do I conjugate tener in the subjunctive?",
    "user_id": "default"
  }' | python -m json.tool
```
**Expected:** `"domain": "LANGUAGE"`, `"agent": "Language_Expert"`

---

## Phase 4 — Portal UI

### 4.1 Start the Portal locally
```bash
cd portal
npm install
npm run dev
# Open http://localhost:5173
```

### 4.2 Sidebar navigation — verify all 9 views load without crash
Click each item and confirm the view renders:
- [ ] Knowledge Graph
- [ ] Conversational
- [ ] **Study Session** ← new
- [ ] Spaced Repetition
- [ ] Note Repository
- [ ] Metabolic Audit
- [ ] **Mission Control** ← new
- [ ] **Researcher** ← new
- [ ] **Staging Area** ← new

### 4.3 Mission Control view
- [ ] Service cards render (8 services visible)
- [ ] Status shows "Checking" then resolves to Online/Offline
- [ ] "Refresh" button re-polls all services
- [ ] Routing Log section shows empty state (no crash)

### 4.4 Study Session (StudyView)
- [ ] Loads without error
- [ ] If cards due: flashcard shows with question, tapping reveals answer
- [ ] Grade buttons (Again / Hard / Good / Easy) are clickable
- [ ] Keyboard shortcuts work: `Space` reveals answer, `1-4` grades the card
- [ ] Mastery ring SVG renders correctly
- [ ] Session complete screen shows after last card

### 4.5 Staging Area
- [ ] Loads without crash (empty state OK)
- [ ] "Retry All" button exists and is disabled when queue is empty

### 4.6 Researcher Monitor
- [ ] Loads without crash
- [ ] Domain selector shows 5 options
- [ ] "Research" button is disabled when input is empty

---

## Phase 5 (Post-MVP) — pgvector Semantic Search

> Only test if you have pgvector installed and schema_v3 migrated.

### 5.1 Verify pgvector extension
```sql
-- Connect to your Cortex PostgreSQL instance:
psql $CORTEX_DB_CONN -c "SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';"
```
**Expected:** 1 row returned with `vector`.

### 5.2 Run schema V3 migration
```bash
psql $CORTEX_DB_CONN -f wisdom/pkg/cortex/schema_v3_pgvector.sql
```
**Expected:** No errors. `CREATE INDEX`, `CREATE TRIGGER` messages.

### 5.3 Test semantic search endpoint
```bash
curl -s -X POST http://localhost:50051/api/v1/cortex/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "chess opening theory",
    "limit": 5,
    "domain_filter": "CHESS"
  }' | python -m json.tool
```
**Expected (if embeddings exist):** JSON with `results` array ordered by similarity score.

### 5.4 Verify HNSW index exists
```sql
psql $CORTEX_DB_CONN -c "SELECT indexname FROM pg_indexes WHERE tablename='nodes' AND indexname='idx_nodes_embedding_hnsw';"
```
**Expected:** 1 row.

---

## Pub/Sub Integration (requires GCP credentials)

### 6.1 Simulate voice input via Pub/Sub push
```bash
# Encode a test payload
PAYLOAD=$(echo -n '{"type":"wisdom.voice.transcribed","text":"What is the Sicilian Defense?","user_id":"default","session_id":"test-001","confidence":0.97}' | base64)

curl -s -X POST http://localhost:8081/pubsub/voice-input \
  -H "Content-Type: application/json" \
  -d "{\"message\":{\"data\":\"$PAYLOAD\"},\"subscription\":\"adk-router-voice-input\"}"
```
**Expected:** HTTP 204 No Content (Pub/Sub ack).

### 6.2 Verify routing log received the event
Check the ADK Router logs:
```bash
docker logs wisdom-adk-router --tail 20
```
**Expected:** Log line with `"Routing decision"` JSON containing `domain`, `confidence`, `elapsed_ms`.

---

## Known Acceptable Behaviors

| Behavior | Why it's OK |
|---|---|
| Integrations returns `QUEUED` instead of `SYNCED` | Obsidian/Anki not running locally — queue works as designed |
| ADK Router takes 2-5s for first response | Gemini model cold start on first call |
| Routing confidence < 0.7 for ambiguous input | Router falls back to GENERAL — correct behavior |
| MCP servers show OFFLINE in Mission Control | They run on host, not in Docker — use localhost ports |
| pgvector tests fail | Requires PostgreSQL + pgvector extension installed locally |

---

## Quick Smoke Test (30 seconds)

```bash
# Run all health checks at once
echo "=== Go Services ===" && \
for port in 50051 50052 50053 50054 50055 50056 50057; do
  printf ":%d → " $port
  curl -sf --max-time 2 http://localhost:$port/health && echo "✓" || echo "✗"
done

echo "=== ADK Router ===" && \
curl -sf http://localhost:8081/health && echo "✓" || echo "✗"

echo "=== Route Test ===" && \
curl -sf -X POST http://localhost:8081/route \
  -H "Content-Type: application/json" \
  -d '{"input":"explain binary search","user_id":"default"}' \
  | python -m json.tool
```
