#!/bin/bash
set -e

# Project Wisdom: Cloud Run Deployment & Automation Script
# Usage: ./deploy_cloud_run.sh [PROJECT_ID] [REGION]

PROJECT_ID=${1:-$(gcloud config get-value project)}
REGION=${2:-"us-central1"}
SERVICE_NAME="wisdom-engine"
IMAGE_TAG="gcr.io/$PROJECT_ID/$SERVICE_NAME:latest"

echo "☁️ Starting Unified Cloud Run Deployment for Project Wisdom..."

# 1. Build & Push (Using root context for Frontend + Backend)
echo "🏗️ Building Unified Docker image..."
docker build -t $IMAGE_TAG .
echo "📤 Pushing image to GCR..."
docker push $IMAGE_TAG

# 2. Deploy to Cloud Run
echo "🚀 Deploying to Cloud Run..."
gcloud run deploy $SERVICE_NAME \
    --image $IMAGE_TAG \
    --platform managed \
    --region $REGION \
    --allow-unauthenticated=false \
    --memory 1Gi \
    --cpu 1 \
    --service-account "nexusstate-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --set-env-vars "GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=$REGION"

SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --platform managed --region $REGION --format 'value(status.url)')

echo "✅ Service deployed at: $SERVICE_URL"

# 3. Setup Automated REM Cycle (Cloud Scheduler)
echo "⏰ Configuring Automated REM Cycle (Daily at 03:00)..."
gcloud scheduler jobs create http daily-rem-cycle \
    --schedule="0 3 * * *" \
    --uri="$SERVICE_URL/rem/all" \
    --http-method=POST \
    --oidc-service-account-email="nexusstate-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --oidc-token-audience="$SERVICE_URL" \
    --location=$REGION \
    --description="Consolidates session logs into the Cortex daily." \
    --quiet || echo "ℹ️ Job already exists or failed to create."

echo "🎉 Deployment complete."
echo "👉 Update your local environment: export WISDOM_SERVICE_URL=$SERVICE_URL"
