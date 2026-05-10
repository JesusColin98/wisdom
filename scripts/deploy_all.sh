#!/bin/bash
# deploy_all.sh - Robust & Idempotent Deployment for Wisdom Ecosystem
# Handles: Wisdom Engine (Go), Chat Agent (Python), and Portal (React)
# Plus: Infrastructure creation and Load Balancer orchestration

set -e

# --- Configuration & Discovery ---
echo "🔍 Discovering environment..."
PROJECT_ID=${1:-$(gcloud config get-value project 2>/dev/null)}

if [ -z "$PROJECT_ID" ]; then
    echo "❌ Error: Project ID not found. Use: ./deploy_all.sh [PROJECT_ID]"
    exit 1
fi

CURRENT_ACCOUNT=$(gcloud config get-value account 2>/dev/null)
if [ -z "$CURRENT_ACCOUNT" ]; then
    echo "❌ Error: No active gcloud account found. Please run 'gcloud auth login' first."
    exit 1
fi

echo "Using Project: $PROJECT_ID"
echo "Using Account: $CURRENT_ACCOUNT"

REGION="us-central1"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
echo "Project: $PROJECT_ID ($PROJECT_NUMBER)"
echo "Region: $REGION"

# Resource Names
RUNTIME_SA_NAME="nexusstate-sa"
RUNTIME_SA="$RUNTIME_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com"
CORTEX_BUCKET="wisdom-cortex-$PROJECT_ID"
STATIC_IP_NAME="wisdom-static-ip"
SSL_CERT_NAME="wisdom-portal-cert"
URL_MAP_NAME="wisdom-portal-map"
HTTPS_PROXY_NAME="wisdom-portal-proxy"
FW_RULE_NAME="wisdom-portal-fw"

echo "🌌 Starting Wisdom Ecosystem Deployment..."

# --- 0. Service Activation ---
echo "⚙️  Enabling required services..."
gcloud services enable \
    run.googleapis.com \
    compute.googleapis.com \
    iam.googleapis.com \
    cloudbuild.googleapis.com \
    iap.googleapis.com \
    secretmanager.googleapis.com \
    artifactregistry.googleapis.com \
    --project "$PROJECT_ID"

# --- 1. Infrastructure Foundation ---

# GCS Bucket for SQLite
if ! gcloud storage buckets describe "gs://$CORTEX_BUCKET" --project "$PROJECT_ID" &>/dev/null; then
    echo "📦 Creating storage bucket: $CORTEX_BUCKET"
    gcloud storage buckets create "gs://$CORTEX_BUCKET" --location=$REGION --project "$PROJECT_ID"
else
    echo "✅ Storage bucket already exists."
fi

# Runtime Service Account
if ! gcloud iam service-accounts describe "$RUNTIME_SA" --project "$PROJECT_ID" &>/dev/null; then
    echo "👤 Creating runtime service account: $RUNTIME_SA_NAME"
    gcloud iam service-accounts create "$RUNTIME_SA_NAME" \
        --display-name="Wisdom Runtime Service Account" --project "$PROJECT_ID"
else
    echo "✅ Runtime service account already exists."
fi

# IAM Permissions for Service Account
echo "🔐 Configuring IAM permissions..."
gcloud storage buckets add-iam-policy-binding "gs://$CORTEX_BUCKET" \
    --member="serviceAccount:$RUNTIME_SA" \
    --role="roles/storage.objectAdmin" --project "$PROJECT_ID" --quiet

# Add Secret Manager access (Best Practice from ARCHITECTURE.md)
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$RUNTIME_SA" \
    --role="roles/secretmanager.secretAccessor" --condition=None --project "$PROJECT_ID" --quiet

# Artifact Registry Repository
REPO_NAME="wisdom-repo"
if ! gcloud artifacts repositories describe "$REPO_NAME" --location=$REGION --project "$PROJECT_ID" &>/dev/null; then
    echo "📦 Creating Artifact Registry repository: $REPO_NAME"
    gcloud artifacts repositories create "$REPO_NAME" \
        --repository-format=docker --location=$REGION --project "$PROJECT_ID"
else
    echo "✅ Artifact Registry repository already exists."
fi

# Static IP
if ! gcloud compute addresses describe "$STATIC_IP_NAME" --global --project "$PROJECT_ID" &>/dev/null; then
    echo "🌐 Reserving global static IP: $STATIC_IP_NAME"
    gcloud compute addresses create "$STATIC_IP_NAME" --global --project "$PROJECT_ID"
fi
IP_ADDRESS=$(gcloud compute addresses describe "$STATIC_IP_NAME" --global --project "$PROJECT_ID" --format='value(address)')
DOMAIN="${IP_ADDRESS//./-}.nip.io"
echo "Target Domain: $DOMAIN"

# --- 2. Build & Deploy Services ---

# Paths (relative to script location)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Helper for Build & Push
build_and_push() {
    local name=$1
    local context_dir=$2
    local config=$3
    local subs=$4
    local image="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/$name:latest"

    echo "🏗️  Building $name..."
    gcloud builds submit "$context_dir" \
        --config "$config" \
        --substitutions "_IMAGE_TAG=$image,$subs" \
        --project "$PROJECT_ID"
    
    echo "$image"
}

# 1. Wisdom Engine
ENGINE_SERVICE="wisdom-engine"
ENGINE_IMAGE=$(build_and_push "$ENGINE_SERVICE" "$ROOT_DIR/wisdom" "$ROOT_DIR/wisdom/cloudbuild.yaml" "")

echo "🚀 Deploying Engine..."
gcloud run deploy $ENGINE_SERVICE \
    --image "$ENGINE_IMAGE" \
    --platform managed \
    --region $REGION \
    --no-allow-unauthenticated \
    --memory 1Gi \
    --cpu 1 \
    --service-account "$RUNTIME_SA" \
    --add-volume=name=wisdom-cortex,type=cloud-storage,bucket="$CORTEX_BUCKET" \
    --add-volume-mount=volume=wisdom-cortex,mount-path=/mnt/wisdom-cortex \
    --set-env-vars "GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=$REGION,WISDOM_DB_PATH=/mnt/wisdom-cortex/wisdom.db" \
    --ingress internal-and-cloud-load-balancing \
    --project "$PROJECT_ID"

ENGINE_URL=$(gcloud run services describe $ENGINE_SERVICE --platform managed --region $REGION --format 'value(status.url)' --project "$PROJECT_ID")

# 2. Chat Agent
AGENT_SERVICE="wisdom-chat-agent"
AGENT_IMAGE=$(build_and_push "$AGENT_SERVICE" "$ROOT_DIR/chat_service" "$ROOT_DIR/chat_service/cloudbuild.yaml" "")

echo "🚀 Deploying Chat Agent..."
gcloud run deploy $AGENT_SERVICE \
    --image "$AGENT_IMAGE" \
    --platform managed \
    --region $REGION \
    --no-allow-unauthenticated \
    --service-account "$RUNTIME_SA" \
    --set-env-vars "WISDOM_ENGINE_URL=$ENGINE_URL" \
    --ingress internal-and-cloud-load-balancing \
    --project "$PROJECT_ID"

AGENT_URL=$(gcloud run services describe $AGENT_SERVICE --platform managed --region $REGION --format 'value(status.url)' --project "$PROJECT_ID")

# 3. Portal
PORTAL_SERVICE="wisdom-portal"
PORTAL_IMAGE=$(build_and_push "$PORTAL_SERVICE" "$ROOT_DIR/portal" "$ROOT_DIR/portal/cloudbuild.yaml" "_ENGINE_URL=$ENGINE_URL,_AGENT_URL=$AGENT_URL,_WS_URL=")

echo "🚀 Deploying Portal..."
gcloud run deploy $PORTAL_SERVICE \
    --image "$PORTAL_IMAGE" \
    --platform managed \
    --region $REGION \
    --no-allow-unauthenticated \
    --memory 256Mi \
    --cpu 1 \
    --service-account "$RUNTIME_SA" \
    --ingress internal-and-cloud-load-balancing \
    --project "$PROJECT_ID"

# --- 3. GCLB & IAP Orchestration ---
echo "⚙️  Configuring Load Balancer & IAP..."

# Create NEGs
for SVC in $ENGINE_SERVICE $AGENT_SERVICE $PORTAL_SERVICE; do
    if ! gcloud compute network-endpoint-groups describe "$SVC-neg" --region=$REGION --project "$PROJECT_ID" &>/dev/null; then
        gcloud compute network-endpoint-groups create "$SVC-neg" \
            --region=$REGION --network-endpoint-type=serverless \
            --cloud-run-service=$SVC --project "$PROJECT_ID"
    fi
done

# Create Backend Services
for SVC in $ENGINE_SERVICE $AGENT_SERVICE $PORTAL_SERVICE; do
    if ! gcloud compute backend-services describe "$SVC-backend" --global --project "$PROJECT_ID" &>/dev/null; then
        gcloud compute backend-services create "$SVC-backend" \
            --load-balancing-scheme=EXTERNAL_MANAGED --global --project "$PROJECT_ID"
        gcloud compute backend-services add-backend "$SVC-backend" \
            --global --network-endpoint-group="$SVC-neg" \
            --network-endpoint-group-region=$REGION --project "$PROJECT_ID"
    fi
done

# Create/Update URL Map
if ! gcloud compute url-maps describe $URL_MAP_NAME --global --project "$PROJECT_ID" &>/dev/null; then
    echo "🗺️ Creating URL Map..."
    gcloud compute url-maps create $URL_MAP_NAME \
        --default-service="$PORTAL_SERVICE-backend" \
        --global --project "$PROJECT_ID"
fi

echo "🗺️ Updating URL Map routing rules..."
gcloud compute url-maps add-path-matcher $URL_MAP_NAME \
    --default-service="$PORTAL_SERVICE-backend" \
    --path-matcher-name="wisdom-matcher" \
    --backend-service-path-rules="/cortex/*=$ENGINE_SERVICE-backend,/metabolism=$ENGINE_SERVICE-backend,/validate=$ENGINE_SERVICE-backend,/whoami=$ENGINE_SERVICE-backend,/ws=$ENGINE_SERVICE-backend,/chat=$AGENT_SERVICE-backend,/ws/chat=$AGENT_SERVICE-backend" \
    --global --project "$PROJECT_ID" --quiet || \
gcloud compute url-maps update-path-matcher $URL_MAP_NAME \
    --default-service="$PORTAL_SERVICE-backend" \
    --path-matcher-name="wisdom-matcher" \
    --backend-service-path-rules="/cortex/*=$ENGINE_SERVICE-backend,/metabolism=$ENGINE_SERVICE-backend,/validate=$ENGINE_SERVICE-backend,/whoami=$ENGINE_SERVICE-backend,/ws=$ENGINE_SERVICE-backend,/chat=$AGENT_SERVICE-backend,/ws/chat=$AGENT_SERVICE-backend" \
    --global --project "$PROJECT_ID" --quiet

echo "🔒 Finalizing IAM bindings..."
for SVC in $ENGINE_SERVICE $AGENT_SERVICE $PORTAL_SERVICE; do
    gcloud iap web add-iam-policy-binding \
        --resource-type=backend-services \
        --service="$SVC-backend" \
        --member="user:$CURRENT_ACCOUNT" \
        --role="roles/iap.httpsResourceAccessor" --project "$PROJECT_ID" --quiet
done

echo "🎉 Ecosystem Deployed & Secured with Path-based Routing!"
echo "Public UI: http://$DOMAIN"
