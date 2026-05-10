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
if ! gcloud compute addresses describe $STATIC_IP_NAME --global --project $PROJECT_ID &>/dev/null; then
    echo "Creating Static IP..."
    gcloud compute addresses create $STATIC_IP_NAME --global --project $PROJECT_ID
fi
IP_ADDRESS=$(gcloud compute addresses describe $STATIC_IP_NAME --global --project $PROJECT_ID --format='value(address)')
DOMAIN="${IP_ADDRESS//./-}.nip.io"
echo "Target Domain: $DOMAIN"

# 1. Deploy Wisdom Engine (Internal-only logic handled via OIDC)
ENGINE_SERVICE="wisdom-engine"
ENGINE_IMAGE="gcr.io/$PROJECT_ID/$ENGINE_SERVICE:latest"

echo "🏗️ Building Engine..."
gcloud builds submit wisdom/     --config wisdom/cloudbuild.yaml     --service-account "$BUILD_SA"     --gcs-source-staging-dir "$STAGING_BUCKET"     --substitutions "_IMAGE_TAG=$ENGINE_IMAGE"

echo "🚀 Deploying Engine..."
gcloud run deploy $ENGINE_SERVICE     --image $ENGINE_IMAGE     --platform managed     --region $REGION     --no-allow-unauthenticated     --memory 1Gi     --cpu 1     --service-account "$RUNTIME_SA"     --add-volume=name=wisdom-cortex,type=cloud-storage,bucket=wisdom-cortex-jesus-mvp     --add-volume-mount=volume=wisdom-cortex,mount-path=/mnt/wisdom-cortex     --set-env-vars "GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=$REGION,WISDOM_DB_PATH=/mnt/wisdom-cortex/wisdom.db"     --ingress internal-and-cloud-load-balancing

ENGINE_URL=$(gcloud run services describe $ENGINE_SERVICE --platform managed --region $REGION --format 'value(status.url)')

# 2. Deploy Chat Agent
AGENT_SERVICE="wisdom-chat-agent"
AGENT_IMAGE="gcr.io/$PROJECT_ID/$AGENT_SERVICE:latest"

echo "🏗️ Building Chat Agent..."
gcloud builds submit chat_service/     --config chat_service/cloudbuild.yaml     --service-account "$BUILD_SA"     --gcs-source-staging-dir "$STAGING_BUCKET"     --substitutions "_IMAGE_TAG=$AGENT_IMAGE"

echo "🚀 Deploying Chat Agent..."
gcloud run deploy $AGENT_SERVICE     --image $AGENT_IMAGE     --platform managed     --region $REGION     --no-allow-unauthenticated     --service-account "$RUNTIME_SA"     --set-env-vars "WISDOM_ENGINE_URL=$ENGINE_URL"     --ingress internal-and-cloud-load-balancing

AGENT_URL=$(gcloud run services describe $AGENT_SERVICE --platform managed --region $REGION --format 'value(status.url)')

# 3. Deploy Portal
PORTAL_SERVICE="wisdom-portal"
PORTAL_IMAGE="gcr.io/$PROJECT_ID/$PORTAL_SERVICE:latest"

echo "🏗️ Building Portal..."
# Note: In a full GCLB setup, the Portal would call the Agent via the LB URL
# For now, we point it to the direct Agent URL (requires IAP or OIDC)
gcloud builds submit portal/     --config portal/cloudbuild.yaml     --service-account "$BUILD_SA"     --gcs-source-staging-dir "$STAGING_BUCKET"     --substitutions "_IMAGE_TAG=$PORTAL_IMAGE,_ENGINE_URL=$ENGINE_URL,_AGENT_URL=$AGENT_URL"

echo "🚀 Deploying Portal..."
gcloud run deploy $PORTAL_SERVICE     --image $PORTAL_IMAGE     --platform managed     --region $REGION     --no-allow-unauthenticated     --memory 256Mi     --cpu 1     --service-account "$RUNTIME_SA"     --ingress internal-and-cloud-load-balancing

# 4. GCLB & IAP Orchestration
echo "⚙️  Configuring Load Balancer & IAP..."

# Create NEGs
for SVC in $ENGINE_SERVICE $AGENT_SERVICE $PORTAL_SERVICE; do
    if ! gcloud compute network-endpoint-groups describe "$SVC-neg" --region=$REGION --project $PROJECT_ID &>/dev/null; then
        gcloud compute network-endpoint-groups create "$SVC-neg"             --region=$REGION --network-endpoint-type=serverless             --cloud-run-service=$SVC --project $PROJECT_ID
    fi
done

# Create Backend Services
for SVC in $ENGINE_SERVICE $AGENT_SERVICE $PORTAL_SERVICE; do
    if ! gcloud compute backend-services describe "$SVC-backend" --global --project $PROJECT_ID &>/dev/null; then
        gcloud compute backend-services create "$SVC-backend"             --load-balancing-scheme=EXTERNAL_MANAGED --global --project $PROJECT_ID
        gcloud compute backend-services add-backend "$SVC-backend"             --global --network-endpoint-group="$SVC-neg"             --network-endpoint-group-region=$REGION --project $PROJECT_ID
    fi
done

# SSL Certificate
if ! gcloud compute ssl-certificates describe $SSL_CERT_NAME --global --project $PROJECT_ID &>/dev/null; then
    gcloud compute ssl-certificates create $SSL_CERT_NAME --domains=$DOMAIN --global --project $PROJECT_ID
fi

# URL Map
if ! gcloud compute url-maps describe $URL_MAP_NAME --global --project $PROJECT_ID &>/dev/null; then
    gcloud compute url-maps create $URL_MAP_NAME --default-service="$PORTAL_SERVICE-backend" --global --project $PROJECT_ID
fi

# Target Proxy
if ! gcloud compute target-https-proxies describe $HTTPS_PROXY_NAME --global --project $PROJECT_ID &>/dev/null; then
    gcloud compute target-https-proxies create $HTTPS_PROXY_NAME --url-map=$URL_MAP_NAME --ssl-certificates=$SSL_CERT_NAME --global --project $PROJECT_ID
fi

# Forwarding Rule
if ! gcloud compute forwarding-rules describe $FW_RULE_NAME --global --project $PROJECT_ID &>/dev/null; then
    gcloud compute forwarding-rules create $FW_RULE_NAME         --address=$STATIC_IP_NAME --target-https-proxy=$HTTPS_PROXY_NAME --global --ports=443 --project $PROJECT_ID
fi

# 5. Secure IAM configuration
echo "🔒 Finalizing IAM bindings..."
# Allow Agent to call Engine
gcloud run services add-iam-policy-binding $ENGINE_SERVICE     --member="serviceAccount:$RUNTIME_SA"     --role="roles/run.invoker"     --region $REGION --quiet

# Allow user to call Agent and Portal via IAP/Direct
CURRENT_USER=$(gcloud config get-value account)
for SVC in $AGENT_SERVICE $PORTAL_SERVICE; do
    gcloud run services add-iam-policy-binding $SVC         --member="user:$CURRENT_USER"         --role="roles/run.invoker"         --region $REGION --quiet
    
    # Also grant IAP access
    gcloud iap web add-iam-policy-binding         --resource-type=backend-services         --service="$SVC-backend"         --member="user:$CURRENT_USER"         --role="roles/iap.httpsResourceAccessor" --project $PROJECT_ID --quiet
done

echo "🎉 Ecosystem Deployed & Secured!"
echo "-----------------------------------"
echo "Public UI (IAP): https://$DOMAIN"
echo "Direct Portal:   $(gcloud run services describe $PORTAL_SERVICE --region $REGION --format 'value(status.url)')"
echo "Direct Agent:    $AGENT_URL"
echo "Direct Engine:   $ENGINE_URL"
echo "-----------------------------------"
