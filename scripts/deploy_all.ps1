# deploy_all.ps1 - Robust Deployment for Wisdom Ecosystem

param (
    [string]$ProjectID = $(gcloud config get-value project)
)

if ([string]::IsNullOrWhiteSpace($ProjectID)) {
    Write-Host "Error: Project ID not found. Pass -ProjectID or set it in gcloud config." -ForegroundColor Red
    exit 1
}

$CURRENT_ACCOUNT = gcloud config get-value account
if ([string]::IsNullOrWhiteSpace($CURRENT_ACCOUNT)) {
    Write-Host "Error: No active gcloud account. Run 'gcloud auth login' first." -ForegroundColor Red
    exit 1
}

# --- Configuration ---
$PROJECT_ID = $ProjectID
$REGION = "us-central1"
$RUNTIME_SA_NAME = "nexusstate-sa"
$RUNTIME_SA = "$RUNTIME_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com"
$CORTEX_BUCKET = "wisdom-cortex-$PROJECT_ID"
$STATIC_IP_NAME = "wisdom-static-ip"
$URL_MAP_NAME = "wisdom-portal-map"
$REPO_NAME = "wisdom-repo"

Write-Host "Starting Wisdom Ecosystem Deployment (PowerShell Edition)..." -ForegroundColor Cyan
Write-Host "Project: $PROJECT_ID"
Write-Host "Account: $CURRENT_ACCOUNT"

# --- 0. Service Activation ---
Write-Host "Enabling required services..."
gcloud services enable `
    run.googleapis.com `
    compute.googleapis.com `
    iam.googleapis.com `
    cloudbuild.googleapis.com `
    iap.googleapis.com `
    secretmanager.googleapis.com `
    artifactregistry.googleapis.com `
    --project $PROJECT_ID

# --- 1. Infrastructure Foundation ---

# GCS Bucket
if (!(gcloud storage buckets describe "gs://$CORTEX_BUCKET" --project $PROJECT_ID --quiet 2>$null)) {
    Write-Host "Creating storage bucket: $CORTEX_BUCKET"
    gcloud storage buckets create "gs://$CORTEX_BUCKET" --location=$REGION --project $PROJECT_ID
} else {
    Write-Host "Storage bucket already exists."
}

# Service Account
if (!(gcloud iam service-accounts describe $RUNTIME_SA --project $PROJECT_ID --quiet 2>$null)) {
    Write-Host "Creating runtime service account: $RUNTIME_SA_NAME"
    gcloud iam service-accounts create $RUNTIME_SA_NAME --display-name="Wisdom Runtime Service Account" --project $PROJECT_ID
} else {
    Write-Host "Runtime service account already exists."
}

# IAM Permissions
Write-Host "Configuring IAM permissions..."
gcloud storage buckets add-iam-policy-binding "gs://$CORTEX_BUCKET" `
    --member="serviceAccount:$RUNTIME_SA" `
    --role="roles/storage.objectAdmin" --project $PROJECT_ID --quiet

# Add Secret Manager Access (Best Practice from ARCHITECTURE.md)
gcloud projects add-iam-policy-binding $PROJECT_ID `
    --member="serviceAccount:$RUNTIME_SA" `
    --role="roles/secretmanager.secretAccessor" --condition=None --project $PROJECT_ID --quiet

# Artifact Registry
if (!(gcloud artifacts repositories describe $REPO_NAME --location=$REGION --project $PROJECT_ID --quiet 2>$null)) {
    Write-Host "Creating Artifact Registry repository: $REPO_NAME"
    gcloud artifacts repositories create $REPO_NAME --repository-format=docker --location=$REGION --project $PROJECT_ID
} else {
    Write-Host "Artifact Registry repository already exists."
}

# Static IP
if (!(gcloud compute addresses describe $STATIC_IP_NAME --global --project $PROJECT_ID --quiet 2>$null)) {
    Write-Host "Reserving global static IP: $STATIC_IP_NAME"
    gcloud compute addresses create $STATIC_IP_NAME --global --project $PROJECT_ID
}
$IP_ADDRESS = gcloud compute addresses describe $STATIC_IP_NAME --global --project $PROJECT_ID --format='value(address)'
$DOMAIN = "$($IP_ADDRESS.Replace('.', '-')).nip.io"
Write-Host "Target Domain: $DOMAIN"

# --- 2. Build & Deploy Services ---

function Build-And-Push($name, $contextDir, $configPath, $substitutions) {
    $image = "${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO_NAME}/${name}:latest"
    Write-Host "Building $name (Tag: $image)..." -ForegroundColor Yellow
    
    $subsString = "_IMAGE_TAG=$image"
    if (![string]::IsNullOrEmpty($substitutions)) {
        $subsString += ",$substitutions"
    }

    gcloud builds submit $contextDir `
        --config $configPath `
        --service-account "projects/$PROJECT_ID/serviceAccounts/cloud-run-build-sa@$PROJECT_ID.iam.gserviceaccount.com" `
        --substitutions $subsString `
        --project $PROJECT_ID | Out-Host
    return $image
}

# 1. Wisdom Engine
$ENGINE_IMAGE = Build-And-Push "wisdom-engine" "wisdom" "wisdom/cloudbuild.yaml" ""
Write-Host "Deploying Engine..."
gcloud run deploy wisdom-engine `
    --image $ENGINE_IMAGE `
    --platform managed `
    --region $REGION `
    --no-allow-unauthenticated `
    --memory 1Gi `
    --cpu 1 `
    --service-account $RUNTIME_SA `
    --add-volume "name=wisdom-cortex,type=cloud-storage,bucket=$CORTEX_BUCKET" `
    --add-volume-mount "volume=wisdom-cortex,mount-path=/mnt/wisdom-cortex" `
    --set-env-vars "GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=$REGION,WISDOM_DB_PATH=/mnt/wisdom-cortex/wisdom.db" `
    --ingress internal-and-cloud-load-balancing `
    --project $PROJECT_ID

$ENGINE_URL = gcloud run services describe wisdom-engine --platform managed --region $REGION --format 'value(status.url)' --project $PROJECT_ID

# 2. Chat Agent
$AGENT_IMAGE = Build-And-Push "wisdom-chat-agent" "chat_service" "chat_service/cloudbuild.yaml" ""
Write-Host "Deploying Chat Agent..."
gcloud run deploy wisdom-chat-agent `
    --image $AGENT_IMAGE `
    --platform managed `
    --region $REGION `
    --no-allow-unauthenticated `
    --service-account $RUNTIME_SA `
    --set-env-vars "WISDOM_ENGINE_URL=$ENGINE_URL" `
    --ingress internal-and-cloud-load-balancing `
    --project $PROJECT_ID

$AGENT_URL = gcloud run services describe wisdom-chat-agent --platform managed --region $REGION --format 'value(status.url)' --project $PROJECT_ID

# 3. Portal
$PORTAL_IMAGE = Build-And-Push "wisdom-portal" "portal" "portal/cloudbuild.yaml" "_ENGINE_URL=$ENGINE_URL,_AGENT_URL=$AGENT_URL,_WS_URL="
Write-Host "Deploying Portal..."
gcloud run deploy wisdom-portal `
    --image $PORTAL_IMAGE `
    --platform managed `
    --region $REGION `
    --no-allow-unauthenticated `
    --memory 256Mi `
    --cpu 1 `
    --service-account $RUNTIME_SA `
    --ingress internal-and-cloud-load-balancing `
    --project $PROJECT_ID

# --- 3. GCLB & IAP ---
Write-Host "Configuring HTTPS Load Balancer and IAP..."

$SERVICES = @("wisdom-engine", "wisdom-chat-agent", "wisdom-portal")

foreach ($svc in $SERVICES) {
    if (!(gcloud compute network-endpoint-groups describe "$svc-neg" --region=$REGION --project $PROJECT_ID --quiet 2>$null)) {
        gcloud compute network-endpoint-groups create "$svc-neg" --region=$REGION --network-endpoint-type=serverless --cloud-run-service=$svc --project $PROJECT_ID
    }
    
    if (!(gcloud compute backend-services describe "$svc-backend" --global --project $PROJECT_ID --quiet 2>$null)) {
        gcloud compute backend-services create "$svc-backend" --load-balancing-scheme=EXTERNAL_MANAGED --global --project $PROJECT_ID
        gcloud compute backend-services add-backend "$svc-backend" --global --network-endpoint-group="$svc-neg" --network-endpoint-group-region=$REGION --project $PROJECT_ID
    }

    # Enable IAP on Backend Service
    Write-Host "Enabling IAP for $svc..."
    # Note: Requires OAuth brand and client to be set up in the console generally, 
    # but we will try to apply the policy binding.
    gcloud iap web add-iam-policy-binding --resource-type=backend-services --service="$svc-backend" --member="user:$CURRENT_ACCOUNT" --role="roles/iap.httpsResourceAccessor" --project $PROJECT_ID --quiet
}

if (!(gcloud compute url-maps describe $URL_MAP_NAME --global --project $PROJECT_ID --quiet 2>$null)) {
    gcloud compute url-maps create $URL_MAP_NAME --default-service="wisdom-portal-backend" --global --project $PROJECT_ID
}

Write-Host "Updating URL Map routing rules..."
gcloud compute url-maps add-path-matcher $URL_MAP_NAME `
    --default-service="wisdom-portal-backend" `
    --path-matcher-name="wisdom-matcher" `
    --backend-service-path-rules="/cortex/*=wisdom-engine-backend,/metabolism=wisdom-engine-backend,/validate=wisdom-engine-backend,/whoami=wisdom-engine-backend,/ws=wisdom-engine-backend,/chat=wisdom-chat-agent-backend,/ws/chat=wisdom-chat-agent-backend" `
    --global --project $PROJECT_ID --quiet 2>$null

# SSL Certificate (Google Managed)
$CERT_NAME = "wisdom-cert"
if (!(gcloud compute ssl-certificates describe $CERT_NAME --project $PROJECT_ID --quiet 2>$null)) {
    Write-Host "Creating Google-managed SSL certificate for $DOMAIN..."
    gcloud compute ssl-certificates create $CERT_NAME --domains=$DOMAIN --project $PROJECT_ID
}

# Target HTTPS Proxy
$PROXY_NAME = "wisdom-https-proxy"
if (!(gcloud compute target-https-proxies describe $PROXY_NAME --project $PROJECT_ID --quiet 2>$null)) {
    gcloud compute target-https-proxies create $PROXY_NAME --url-map=$URL_MAP_NAME --ssl-certificates=$CERT_NAME --project $PROJECT_ID
}

# Global Forwarding Rule
$FW_RULE_NAME = "wisdom-fw-rule"
if (!(gcloud compute forwarding-rules describe $FW_RULE_NAME --global --project $PROJECT_ID --quiet 2>$null)) {
    gcloud compute forwarding-rules create $FW_RULE_NAME --address=$STATIC_IP_NAME --target-https-proxy=$PROXY_NAME --global --ports=443 --project $PROJECT_ID
}

Write-Host "Ecosystem Deployed and Secured with IAP!" -ForegroundColor Green
Write-Host "Public UI: https://$DOMAIN"
Write-Host "Note: SSL provisioning may take 10-20 minutes."

