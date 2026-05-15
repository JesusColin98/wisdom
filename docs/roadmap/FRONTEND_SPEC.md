# Frontend Specification: Wisdom Portal (Full)

The Wisdom Portal is a **React + Vite** SPA deployed to Cloud Run. It is the user's **Cognitive Cockpit** — it teaches best practices (LIFT, atomic notes), enforces quality standards (Obsidian compliance), and surfaces system intelligence visually.

> This document supersedes the observability-only scope of `PORTAL_SPEC.md`.

---

## 1. Design System

- **Framework:** React 18 + Vite
- **Styling:** Vanilla CSS with CSS custom properties
- **Icons:** Lucide React
- **Fonts:** `Inter` (UI), `JetBrains Mono` (code/Markdown previews)
- **Real-time:** Native WebSocket API

### Color Tokens
```css
--color-bg-primary: #0d1117;
--color-bg-surface: #161b22;
--color-bg-elevated: #21262d;
--color-accent-primary: #7c3aed;   /* Violet — AI actions */
--color-accent-secondary: #0ea5e9; /* Sky blue — system actions */
--color-success: #22c55e;
--color-warning: #f59e0b;
--color-danger: #ef4444;
--color-text-primary: #f0f6fc;
--color-text-muted: #8b949e;
```

### Micro-animations
- All interactive elements: `transition: all 0.15s ease`
- Page transitions: fade-in over `200ms`
- Streaming text: character-by-character with cursor blink
- Agent badge: pulse animation while streaming

---

## 2. View Inventory (6 Core Views)

### View 1: Chat (Default Landing)
See `CHATBOT_UI_SPEC.md` for the full spec.

- **Route:** `/chat` — first view after login
- **Layout:** Two-column desktop (chat + context panel); single-column mobile
- **Context Panel:** Recently modified vault notes + pending Anki cards

---

### View 2: Note Editor (Guided Creation Wizard)

Route: `/notes` (new) | `/notes/:id` (existing)

#### 2.1 YAML Frontmatter Panel
Rendered as form inputs (not raw YAML):
```yaml
---
id: <auto: YYYYMMDDHHMMSS>
title: ""
tipo: permanente       # permanente | literatura | efimera
estado: borrador       # borrador | en-progreso | terminado
tema: ""
fase: ""               # Parent MOC: [[MOC Name]]
tags: []
mastery_score: 0.0
date: <auto: today>
---
```
- `fase` field: searchable dropdown of existing MOC notes in vault.

#### 2.2 Markdown Editor
- Split pane: Editor (left) + Obsidian-style preview (right)
- `[[` triggers fuzzy-search wikilink autocomplete
- `^` at end of paragraph auto-generates a unique block ID
- Word count + reading time in footer

#### 2.3 LIFT Compliance Linter (Pre-save)

| Rule | Check | Action on Failure |
|---|---|---|
| Atomic size | Body < 500 words | Warning: suggest splitting |
| Parent MOC | `fase` not empty | Error: required |
| 3-link rule | ≥1 wikilink in body | Error: add at least one `[[link]]` |
| Path depth | Destination ≤ 2 folder levels | Error: use MOCs instead |
| YAML validity | All required fields present | Error: list missing fields |
| Unique ID | `id` present and unique | Auto-fix: generate one |

#### 2.4 "Polish with Gemini" Button (✨)

Trigger: User clicks ✨ in the editor toolbar.

**Gemini audits for:**
- Prose redundancy → collapse into one statement
- Long paragraphs (>4 sentences same topic) → convert to bullets or table
- Missing heading hierarchy → add structure
- Heading order violations → reorder
- Wikilink gaps → suggest `[[links]]` for terms matching vault notes
- Note size → suggest split if >500 words
- LIFT compliance

**UI flow:**
1. Editor dims → "Analyzing with Gemini..."
2. **Diff view** replaces editor: green = additions, red = removals, per hunk
3. Each hunk: `[Accept]` / `[Reject]` buttons. "Accept All" / "Reject All" at top
4. Accepted changes written back to editor; user saves normally

**API Contract:**
```
POST /v1/notes/polish
Body:    { "content": "...", "metadata": { "title": "...", "tags": [...] } }
Response: { "hunks": [{ "original": "...", "rewritten": "...", "reason": "..." }] }
```

#### 2.5 PARA Destination Selector
PARA-guided dropdown for choosing the destination folder:

| PARA Bucket | Path | Auto-suggested for |
|---|---|---|
| Inbox | `00_Inbox/` | Default for all new notes |
| Projects | `10_Projects/<project>` | `tipo: efimera` with deadline |
| Areas | `20_Areas/<area>` | Ongoing responsibilities |
| Resources | `30_Resources/<domain>/` | `tipo: literatura` or `permanente` |
| Archive | `40_Archive/` | Completed/obsolete |
| Templates | `99_Templates/` | `tipo: template` |

---

### View 3: Vault Health Dashboard

Route: `/vault`
Data source: `GET /v1/vault/health` (polled every 5 min or WebSocket push)

#### Metrics Panel

| Metric | Description |
|---|---|
| Total notes | Count of all `.md` files |
| Orphan rate | % notes with 0 incoming wikilinks |
| MOC coverage | % notes with a `fase` link |
| 3-link compliance | % notes with ≥3 wikilinks |
| LIFT violations | Count of notes failing any linter rule |
| Avg mastery score | Mean `mastery_score` across vault |
| Notes in Inbox | Count still in `00_Inbox/` |

#### Orphan Notes List
- Title + last modified date
- `[Open in Editor]` | `[Suggest MOC]` (Gemini suggests best parent MOC)

#### Graph Snapshot
- Miniature force-directed graph (D3.js) showing connectivity clusters
- Click node → opens note in editor

---

### View 4: Anki Export Review Panel

Route: `/anki`
Data source: `GET /v1/integrations/anki/queue`

#### Card Preview List
Each queued card shown as a flip-card:
- Front face (question/cloze)
- Back face (answer + `[[Source Note]]` wikilink)
- Deck + tags

#### Per-card Actions
`[Edit]` | `[Change type: Basic↔Cloze]` | `[Discard]` | `[Send now]`

#### Batch Actions
`[Send all to Anki]` | `[Export as .apkg]`

---

### View 5: System Health (Mission Control)
Inherited from `PORTAL_SPEC.md`. No changes.

Status indicators for: `Thalamus`, `Cortex`, `Researcher`, `Integrations`, `chat_service`, `Vertex AI Memory Bank`.

---

### View 6: Wisdom Study UI (Proprietary SRS)
Inherited from `PORTAL_SPEC.md`. No changes.

Data source: `GET /v1/metabolism/due`

---

## 3. Obsidian PKM Onboarding Flow

First-time users see a guided sequence before accessing any view:

1. **Connect Vault:** Point to Obsidian vault directory; verify `Local REST API` plugin
2. **Philosophy intro:** Interactive tutorial on Atomic Notes, LIFT, PARA, 3-link rule (skippable)
3. **Vault baseline:** Run LIFT linter on existing vault → show health report
4. **First note together:** Guided creation using the Note Editor

---

## 4. Navigation Structure

```
/chat          ← Default
/notes         ← New note
/notes/:id     ← Edit note
/vault         ← Vault Health
/anki          ← Anki Queue
/system        ← System Health
/study         ← SRS Study
```

- **Desktop:** Left sidebar — icon + label (collapsible)
- **Mobile:** Bottom tab bar — 4 primary icons (Chat, Notes, Vault, Study)

---

## 5. Mobile Responsiveness

| Element | Mobile (≤768px) | Desktop (>768px) |
|---|---|---|
| Navigation | Bottom tab bar | Left sidebar |
| Chat layout | Full-width | Two-column |
| Note editor | Single-pane (no preview) | Split editor + preview |
| Agent indicator | Icon only | Full label |
| Vault health graph | Hidden | Visible |
| YAML form | Collapsible accordion | Always visible |
