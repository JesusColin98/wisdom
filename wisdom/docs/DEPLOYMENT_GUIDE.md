# Project Wisdom: Deployment & Operations Guide

## 1. Local Development
To run the engine locally for development or testing:

```bash
cd wisdom
go build -o wisdom_engine cmd/wisdom/main.go
./wisdom_engine
```
The server will listen on `localhost:8080`. Access the Neural Atlas at `http://localhost:8080/ui/`.

## 2. Cloud Run Deployment (Recommended)
This is the production-ready deployment that includes the unified Frontend, Backend, and automated REM Cycle.

### Initial Setup
Ensure you have the following permissions:
- Cloud Run Admin
- Storage Admin (for Container Registry)
- Cloud Scheduler Admin
- Service Account User

### Execution Command
Run the deployment script from the project root:

```bash
# Usage: ./wisdom/scripts/deploy_cloud_run.sh [PROJECT_ID] [REGION]
./wisdom/scripts/deploy_cloud_run.sh jesus-mvp us-central1
```

### What this script automates:
1.  **Unified Build:** Compiles the React frontend and Go backend into a single multi-stage Docker image.
2.  **Infrastructure:** Deploys to Cloud Run with 1Gi RAM and 1 CPU.
3.  **Security:** Configures OIDC authentication.
4.  **Learning:** Sets up a Cloud Scheduler job to hit `/rem/all` every day at 03:00 AM.

## 3. Connecting Gemini CLI
To point your Gemini CLI instances to the production engine:

1.  Set the environment variable:
    ```bash
    export WISDOM_SERVICE_URL="https://wisdom-engine-your-hash.a.run.app"
    ```
2.  The `wisdom_bridge.py` will automatically handle OIDC token generation for secure communication.

## 4. Operational Monitoring
- **Health Check:** `GET /health`
- **Metabolic Rate:** `GET /metabolism` (Check TSR and token efficiency)
- **Neural Atlas:** `GET /ui/` (Visual graph exploration)
