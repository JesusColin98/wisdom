# Wisdom Deployment & Infrastructure Guide

## Cloud Run Cutover (The "One-Click" Strategy)

Wisdom is designed to be a drop-in replacement for the legacy `nexusstate` service. Follow these steps to deploy and cleanup.

### 1. Build and Push Container
```bash
cd wisdom
# Build for Google Cloud Artifact Registry
gcloud builds submit --tag gcr.io/[PROJECT_ID]/wisdom-engine .
```

### 2. Deploy to Cloud Run
Deploy using the existing Service Account to maintain permissions. **Note: No keep-alive required.**
```bash
gcloud run deploy wisdom-mcp \
  --image gcr.io/[PROJECT_ID]/wisdom-engine \
  --service-account nexusstate-sa@[PROJECT_ID].iam.gserviceaccount.com \
  --port 8080 \
  --region us-central1 \
  --allow-unauthenticated \
  --update-env-vars WISDOM_DB_PATH=/mnt/storage/wisdom.db
```

### 3. Persistent Storage (GCS Fuse)
To ensure your notes and graph data survive container restarts:
1. Create a GCS bucket: `gcloud storage buckets create gs://wisdom-cortex-[PROJECT_ID]`
2. Update Cloud Run to mount this bucket as a volume at `/mnt/storage`.

---

## 🧹 Infrastructure Cleanup (Cost Savings)

Once Wisdom is verified at `https://wisdom-mcp-...`, delete these legacy resources:

### 1. Delete Keep-Alive Cloud Function
Legacy Python had slow cold starts. Go is fast enough that this is wasted money.
```bash
gcloud functions delete nexusstate-keep-alive --region us-central1
```

### 2. Deprovision Legacy Redis (Memorystore)
Wisdom's SQLite substrate in Go is faster than legacy Python + Redis.
```bash
gcloud redis instances delete [REDIS_INSTANCE_ID] --region us-central1
```

### 3. Deprovision Legacy Spanner
The relational schema has been ported to SQLite.
```bash
gcloud spanner instances delete [SPANNER_INSTANCE_ID]
```

---

## 🛡️ Permission Audit
The `nexusstate-sa` requires the following roles:
- `roles/logging.logWriter` (OTel & Slog)
- `roles/secretmanager.secretAccessor` (LLM Keys)
- `roles/storage.objectAdmin` (GCS Fuse Persistence)
- `roles/aiplatform.user` (LLM access)

---

## 🏗️ Future-Proofing
To add a new tool or feature, see `docs/CONTRIBUTING.md`. The system handles concurrency, circuit breaking, and resource tracking automatically.
