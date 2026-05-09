#!/bin/bash
# deploy_fix.sh - Unified Cloud Run Deployment for Project Wisdom
# This script uses 'gcloud builds submit' instead of local 'docker build'
# to avoid environment limitations and handles common permission issues.

set -e

PROJECT_ID=${1:-$(gcloud config get-value project)}
REGION=${2:-"us-central1"}
SERVICE_NAME="wisdom-engine"
IMAGE_TAG="gcr.io/$PROJECT_ID/$SERVICE_NAME:latest"

echo "☁️ Starting Unified Cloud Run Deployment for Project Wisdom..."

# 1. Build & Push using Cloud Build (Multi-stage Dockerfile handled remotely)
echo "🏗️ Building Unified Docker image with Cloud Build..."
# We use a custom service account and an explicit staging bucket to avoid dependency on the broken default setup.
gcloud builds submit \
    --config brujula/wisdom/cloudbuild.yaml \
    --service-account "projects/$PROJECT_ID/serviceAccounts/cloud-run-build-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --gcs-source-staging-dir "gs://wisdom-build-jesus-mvp/source" \
    --substitutions "_IMAGE_TAG=$IMAGE_TAG" \
    brujula/wisdom/

# 2. Deploy to Cloud Run
# We use the existing nexusstate-sa which has established permissions.
# We mount gs://wisdom-cortex-jesus-mvp to /mnt/wisdom-cortex for persistence.
echo "🚀 Deploying to Cloud Run..."
gcloud run deploy $SERVICE_NAME \
    --image $IMAGE_TAG \
    --platform managed \
    --region $REGION \
    --allow-unauthenticated \
    --memory 1Gi \
    --cpu 1 \
    --service-account "nexusstate-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --add-volume=name=wisdom-cortex,type=cloud-storage,bucket=wisdom-cortex-jesus-mvp \
    --add-volume-mount=volume=wisdom-cortex,mount-path=/mnt/wisdom-cortex \
    --set-env-vars "GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=$REGION,WISDOM_DB_PATH=/mnt/wisdom-cortex/wisdom.db"

SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --platform managed --region $REGION --format 'value(status.url)')

echo "✅ Service deployed at: $SERVICE_URL"
echo "👉 Update your local environment: export WISDOM_SERVICE_URL=$SERVICE_URL"
