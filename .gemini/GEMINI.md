# Workspace Policy: brujula (Wisdom Engine Integration)

## Project Background
The 'brujula' project has migrated from NexusState to the **Wisdom Engine**. Wisdom is a high-performance Go-based cognitive engine providing a unified interface for SRE tasks, including a Neural Atlas for relational intelligence and automated REM cycles for knowledge consolidation.

## 1. Cognitive Architecture
- **Wisdom Engine:** The primary backend for all cognitive and state management.
- **Neural Atlas:** Accessible at `http://localhost:8080/ui/` (local) or via the Cloud Run service URL.
- **Bridge:** Managed via `wisdom_bridge.py`.

## 2. Operating Procedures
- **Tool Usage:** Use the tools provided by the Wisdom MCP bridge.
- **Memory Consolidation:** Proactively trigger REM cycles via the `/rem` endpoint when significant learning occurs.
- **Language Policy:** All internal documentation and responses MUST be in **English**.

## 3. Deployment & Recovery
- **Local:** `cd wisdom && go build -o wisdom_engine cmd/wisdom/main.go && ./wisdom_engine`
- **Auto-Recovery:** The `wisdom_bridge.py` will attempt to restart the local engine if it is not responding on `localhost:8080`.
- **Cloud Run:** Use `./wisdom/scripts/deploy_cloud_run.sh` to move to a serverless architecture.

## 4. Verification & Health
- **Engine Status:** `curl http://localhost:8080/health`
- **Metabolism:** `curl http://localhost:8080/metabolism`
- **Manual Start:** If the bridge fails to auto-start, run: `nohup ./wisdom/wisdom_engine > wisdom_engine.log 2>&1 &`


