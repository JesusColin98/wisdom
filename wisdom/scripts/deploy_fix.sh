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
    --config wisdom/cloudbuild.yaml \
    --service-account "projects/$PROJECT_ID/serviceAccounts/cloud-run-build-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --gcs-source-staging-dir "gs://wisdom-build-jesus-mvp/source" \
    --substitutions "_IMAGE_TAG=$IMAGE_TAG" \
    wisdom/

# 2. Deploy Engine to Cloud Run
# We use the existing nexusstate-sa which has established permissions.
# We mount gs://wisdom-cortex-jesus-mvp to /mnt/wisdom-cortex for persistence.
echo "🚀 Deploying Wisdom Engine to Cloud Run..."
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

ENGINE_URL=$(gcloud run services describe $SERVICE_NAME --platform managed --region $REGION --format 'value(status.url)')
WS_URL=$(echo $ENGINE_URL | sed 's/https/wss/')

echo "✅ Engine deployed at: $ENGINE_URL"

# 3. Build & Deploy Portal
PORTAL_SERVICE="wisdom-portal"
PORTAL_IMAGE="gcr.io/$PROJECT_ID/$PORTAL_SERVICE:latest"

echo "🏗️ Building Wisdom Portal with Cloud Build..."
gcloud builds submit \
    --config portal/cloudbuild.yaml \
    --service-account "projects/$PROJECT_ID/serviceAccounts/cloud-run-build-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --gcs-source-staging-dir "gs://wisdom-build-jesus-mvp/source" \
    --substitutions "_IMAGE_TAG=$PORTAL_IMAGE,_ENGINE_URL=$ENGINE_URL,_WS_URL=$WS_URL" \
    portal/

echo "🚀 Deploying Wisdom Portal to Cloud Run..."
gcloud run deploy $PORTAL_SERVICE \
    --image $PORTAL_IMAGE \
    --platform managed \
    --region $REGION \
    --allow-unauthenticated \
    --memory 256Mi \
    --cpu 1 \
    --service-account "nexusstate-sa@$PROJECT_ID.iam.gserviceaccount.com"

PORTAL_URL=$(gcloud run services describe $PORTAL_SERVICE --platform managed --region $REGION --format 'value(status.url)')

echo "🎉 All services deployed!"
echo "👉 Portal URL: $PORTAL_URL"
echo "👉 Engine URL: $ENGINE_URL"
