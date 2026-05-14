# Wisdom MCP Servers — Local Setup Guide

These two Node.js servers run on your **local machine** and bridge the Wisdom
Cognitive Runtime with your local Obsidian vault and Anki decks.

---

## Prerequisites

| Requirement | Notes |
|---|---|
| **Node.js ≥ 20** | `node --version` |
| **Obsidian** | With the [Local REST API](https://github.com/coddingtonbear/obsidian-local-rest-api) plugin |
| **Anki Desktop** | With the [AnkiConnect](https://ankiweb.net/shared/info/2055492159) add-on (code: `2055492159`) |

---

## 1. Obsidian MCP Server (`localhost:3333`)

### Setup

```bash
# 1. Install Obsidian Local REST API plugin:
#    Obsidian → Settings → Community Plugins → Browse → "Local REST API"
#    Then enable it and generate an API key.

# 2. Install dependencies
cd mcp-servers/obsidian
npm install

# 3. Configure environment variables
cp .env.example .env
# Edit .env and set:
#   OBSIDIAN_API_KEY=<your-api-key-from-obsidian-plugin>
#   WISDOM_VAULT_ROOT=Wisdom/   # Vault subfolder for Wisdom notes (optional)

# 4. Start the server
npm run dev       # Development (ts-node)
npm run build && npm start  # Production
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `OBSIDIAN_API_URL` | `https://localhost:27123` | Obsidian Local REST API URL |
| `OBSIDIAN_API_KEY` | *(required)* | API key from the Obsidian plugin |
| `WISDOM_VAULT_ROOT` | `""` | Subfolder prefix for Wisdom notes, e.g. `Wisdom/` |
| `PORT` | `3333` | HTTP port for the Integrations service bridge |

### Tools Exposed

| Tool | Description |
|---|---|
| `create_note` | Creates a note with YAML frontmatter (title, tags, mastery_score) |
| `read_note` | Reads a note by vault-relative path |
| `update_note` | Overwrites a note's content |
| `search_vault` | Full-text search across the vault |

---

## 2. Anki MCP Server (`localhost:3334`)

### Setup

```bash
# 1. Install AnkiConnect add-on in Anki:
#    Anki → Tools → Add-ons → Get Add-ons → Code: 2055492160
#    Restart Anki.

# 2. (Important) Configure AnkiConnect CORS in Anki:
#    Tools → Add-ons → AnkiConnect → Config
#    Add to webCorsOriginList: ["http://localhost", "http://localhost:3334"]

# 3. Install dependencies
cd mcp-servers/anki
npm install

# 4. Start the server (Anki desktop must be running)
npm run dev
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ANKICONNECT_URL` | `http://localhost:8765` | AnkiConnect URL |
| `PORT` | `3334` | HTTP port for the Integrations service bridge |

### Tools Exposed

| Tool | Description |
|---|---|
| `add_note` | Creates a flashcard using the `Wisdom-Basic` or `Wisdom-Cloze` model |
| `find_notes` | Searches cards using AnkiConnect query syntax |
| `get_reviews` | Fetches review history for all `tag:Wisdom` cards (used by the sync loop) |

### Note Types Created Automatically

On first use, the Anki server creates two custom note types in your collection:

- **Wisdom-Basic** — Front/Back cards with styled CSS
- **Wisdom-Cloze** — Cloze deletion cards with Extra field

---

## 3. Running Both Servers Together

```bash
# From the mcp-servers directory, run both concurrently:
npm install -g concurrently
concurrently \
  "cd obsidian && npm run dev" \
  "cd anki && npm run dev"
```

Or add to your `.env` / startup script and run them as background services.

---

## 4. Gemini CLI Integration

Add both servers to your `~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "wisdom-obsidian": {
      "command": "node",
      "args": ["C:/Users/jesus/wisdom/wisdom/mcp-servers/obsidian/dist/index.js"],
      "env": {
        "OBSIDIAN_API_KEY": "<your-key>",
        "WISDOM_VAULT_ROOT": "Wisdom/"
      }
    },
    "wisdom-anki": {
      "command": "node",
      "args": ["C:/Users/jesus/wisdom/wisdom/mcp-servers/anki/dist/index.js"]
    }
  }
}
```

> **Note:** After adding to Gemini CLI, restart the terminal session running `gemini`.

---

## 5. Testing the Connection

```bash
# Test Obsidian MCP is online:
curl -X POST http://localhost:3333/tools/search_vault \
  -H "Content-Type: application/json" \
  -d '{"action":"search_vault","query":"chess opening"}'

# Test Anki MCP is online (Anki must be open):
curl -X POST http://localhost:3334/tools/get_reviews \
  -H "Content-Type: application/json" \
  -d '{"action":"get_reviews","query":"tag:Wisdom"}'
```
