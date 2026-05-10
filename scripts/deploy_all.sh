#!/bin/bash
# deploy_all.sh - Centralized Deployment for Wisdom Ecosystem
# Handles: Wisdom Engine (Go), Chat Agent (Python), and Portal (React)
# Plus: Global External Load Balancer (GCLB) and Identity-Aware Proxy (IAP)

set -e

# Constants
PROJECT_ID="jesus-mvp"
REGION="us-central1"
BUILD_SA="projects/$PROJECT_ID/serviceAccounts/cloud-run-build-sa@$PROJECT_ID.iam.gserviceaccount.com"
RUNTIME_SA="nexusstate-sa@$PROJECT_ID.iam.gserviceaccount.com"
STAGING_BUCKET="gs://wisdom-build-jesus-mvp/source"
STATIC_IP_NAME="wisdom-static-ip"
SSL_CERT_NAME="wisdom-portal-cert"
URL_MAP_NAME="wisdom-portal-map"
HTTPS_PROXY_NAME="wisdom-portal-proxy"
FW_RULE_NAME="wisdom-portal-fw"

echo "🌌 Starting Wisdom Ecosystem Deployment..."

# 0. Infrastructure Foundation (Static IP)
IP_ADDRESS=$(gcloud compute addresses describe $STATIC_IP_NAME --global --project $PROJECT_ID --format='value(address)')
DOMAIN="${IP_ADDRESS//./-}.nip.io"
echo "Target Domain: $DOMAIN"

# 1. Deploy Wisdom Engine
ENGINE_SERVICE="wisdom-engine"
ENGINE_IMAGE="gcr.io/$PROJECT_ID/$ENGINE_SERVICE:latest"

echo "🏗️ Building Engine..."
gcloud builds submit ../wisdom/ --config ../wisdom/cloudbuild.yaml --service-account "$BUILD_SA" --gcs-source-staging-dir "$STAGING_BUCKET" --substitutions "_IMAGE_TAG=$ENGINE_IMAGE"

echo "🚀 Deploying Engine..."
gcloud run deploy $ENGINE_SERVICE \
    --image $ENGINE_IMAGE \
    --platform managed \
    --region $REGION \
    --no-allow-unauthenticated \
    --memory 1Gi \
    --cpu 1 \
    --service-account "$RUNTIME_SA" \
    --add-volume=name=wisdom-cortex,type=cloud-storage,bucket=wisdom-cortex-jesus-mvp \
    --add-volume-mount=volume=wisdom-cortex,mount-path=/mnt/wisdom-cortex \
    --set-env-vars "GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=$REGION,WISDOM_DB_PATH=/mnt/wisdom-cortex/wisdom.db" \
    --ingress internal-and-cloud-load-balancing

ENGINE_URL=$(gcloud run services describe $ENGINE_SERVICE --platform managed --region $REGION --format 'value(status.url)')

# 2. Deploy Chat Agent
AGENT_SERVICE="wisdom-chat-agent"
AGENT_IMAGE="gcr.io/$PROJECT_ID/$AGENT_SERVICE:latest"

echo "🏗️ Building Chat Agent..."
gcloud builds submit . --config cloudbuild.yaml --service-account "$BUILD_SA" --gcs-source-staging-dir "$STAGING_BUCKET" --substitutions "_IMAGE_TAG=$AGENT_IMAGE"

echo "🚀 Deploying Chat Agent..."
gcloud run deploy $AGENT_SERVICE \
    --image $AGENT_IMAGE \
    --platform managed \
    --region $REGION \
    --no-allow-unauthenticated \
    --service-account "$RUNTIME_SA" \
    --set-env-vars "WISDOM_ENGINE_URL=$ENGINE_URL" \
    --ingress internal-and-cloud-load-balancing

AGENT_URL=$(gcloud run services describe $AGENT_SERVICE --platform managed --region $REGION --format 'value(status.url)')

# 3. Deploy Portal
PORTAL_SERVICE="wisdom-portal"
PORTAL_IMAGE="gcr.io/$PROJECT_ID/$PORTAL_SERVICE:latest"

echo "🏗️ Building Portal..."
# We use empty strings for API URLs to force relative paths in the frontend
# This ensures that API calls are routed through the GCLB on the same domain
gcloud builds submit ../portal/ --config ../portal/cloudbuild.yaml --service-account "$BUILD_SA" --gcs-source-staging-dir "$STAGING_BUCKET" --substitutions "_IMAGE_TAG=$PORTAL_IMAGE,_ENGINE_URL=,_AGENT_URL=,_WS_URL="

echo "🚀 Deploying Portal..."
gcloud run deploy $PORTAL_SERVICE \
    --image $PORTAL_IMAGE \
    --platform managed \
    --region $REGION \
    --no-allow-unauthenticated \
    --memory 256Mi \
    --cpu 1 \
    --service-account "$RUNTIME_SA" \
    --ingress internal-and-cloud-load-balancing

# 4. GCLB & IAP Orchestration
echo "⚙️  Configuring Load Balancer & IAP..."

# Create NEGs
for SVC in $ENGINE_SERVICE $AGENT_SERVICE $PORTAL_SERVICE; do
    if ! gcloud compute network-endpoint-groups describe "$SVC-neg" --region=$REGION --project $PROJECT_ID &>/dev/null; then
        gcloud compute network-endpoint-groups create "$SVC-neg" \
            --region=$REGION --network-endpoint-type=serverless \
            --cloud-run-service=$SVC --project $PROJECT_ID
    fi
done

# Create Backend Services
for SVC in $ENGINE_SERVICE $AGENT_SERVICE $PORTAL_SERVICE; do
    if ! gcloud compute backend-services describe "$SVC-backend" --global --project $PROJECT_ID &>/dev/null; then
        gcloud compute backend-services create "$SVC-backend" \
            --load-balancing-scheme=EXTERNAL_MANAGED --global --project $PROJECT_ID
        gcloud compute backend-services add-backend "$SVC-backend" \
            --global --network-endpoint-group="$SVC-neg" \
            --network-endpoint-group-region=$REGION --project $PROJECT_ID
    fi
done

# URL Map refactoring
echo "Updating URL Map routing rules..."
gcloud compute url-maps add-path-matcher $URL_MAP_NAME \
    --default-service="$PORTAL_SERVICE-backend" \
    --path-matcher-name="wisdom-matcher" \
    --backend-service-path-rules="/cortex/*=$ENGINE_SERVICE-backend,/metabolism=$ENGINE_SERVICE-backend,/validate=$ENGINE_SERVICE-backend,/whoami=$ENGINE_SERVICE-backend,/ws=$ENGINE_SERVICE-backend,/chat=$AGENT_SERVICE-backend,/ws/chat=$AGENT_SERVICE-backend" \
    --global --project $PROJECT_ID --quiet || \
gcloud compute url-maps update-path-matcher $URL_MAP_NAME \
    --default-service="$PORTAL_SERVICE-backend" \
    --path-matcher-name="wisdom-matcher" \
    --backend-service-path-rules="/cortex/*=$ENGINE_SERVICE-backend,/metabolism=$ENGINE_SERVICE-backend,/validate=$ENGINE_SERVICE-backend,/whoami=$ENGINE_SERVICE-backend,/ws=$ENGINE_SERVICE-backend,/chat=$AGENT_SERVICE-backend,/ws/chat=$AGENT_SERVICE-backend" \
    --global --project $PROJECT_ID --quiet

# 5. Secure IAM configuration
echo "🔒 Finalizing IAM bindings..."
CURRENT_USER=$(gcloud config get-value account)
for SVC in $ENGINE_SERVICE $AGENT_SERVICE $PORTAL_SERVICE; do
    gcloud iap web add-iam-policy-binding \
        --resource-type=backend-services \
        --service="$SVC-backend" \
        --member="user:$CURRENT_USER" \
        --role="roles/iap.httpsResourceAccessor" --project $PROJECT_ID --quiet
done

echo "🎉 Ecosystem Deployed & Secured with Path-based Routing!"
echo "Public UI: https://$DOMAIN"
