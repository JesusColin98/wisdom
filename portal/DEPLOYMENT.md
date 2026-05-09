# NexusState Multi-Service Deployment Guide

This document outlines the proven commands for building and deploying the NexusState ecosystem to Google Cloud Run.

## Project Constants
- **Project ID**: `jesus-mvp`
- **Region**: `us-central1`
- **Build Service Account**: `projects/jesus-mvp/serviceAccounts/cloud-run-build-sa@jesus-mvp.iam.gserviceaccount.com`

## 1. Wisdom Engine (Go Backend)
```bash
cd brujula/wisdom
gcloud run deploy wisdom-engine \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --project jesus-mvp \
  --build-service-account projects/jesus-mvp/serviceAccounts/cloud-run-build-sa@jesus-mvp.iam.gserviceaccount.com
```

## 2. Chat Agent Service (Python Orchestrator)
```bash
cd brujula/chat_service
gcloud run deploy wisdom-chat-agent \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --project jesus-mvp \
  --env-vars-file .env.yaml \
  --build-service-account projects/jesus-mvp/serviceAccounts/cloud-run-build-sa@jesus-mvp.iam.gserviceaccount.com
```

## 3. NexusPortal (React Frontend)
```bash
cd brujula/portal
gcloud run deploy nexusstate-portal \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --project jesus-mvp \
  --build-service-account projects/jesus-mvp/serviceAccounts/cloud-run-build-sa@jesus-mvp.iam.gserviceaccount.com
```

### Note on Connectivity
Ensure the `WISDOM_ENGINE_URL` in the Chat Agent Service points to the deployed URL of the Wisdom Engine.
