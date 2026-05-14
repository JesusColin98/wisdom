#!/usr/bin/env bash
# scripts/bootstrap_gcp.sh
#
# One-time GCP project bootstrap for the Wisdom Cognitive Runtime.
# Run ONCE before terraform apply to enable required APIs and create
# the Terraform state bucket.
#
# Usage:
#   export PROJECT_ID=your-gcp-project-id
#   export REGION=us-central1
#   bash scripts/bootstrap_gcp.sh

set -euo pipefail

PROJECT_ID="${PROJECT_ID:?Set PROJECT_ID env variable}"
REGION="${REGION:-us-central1}"
TF_STATE_BUCKET="${PROJECT_ID}-tf-state"

echo "==> Bootstrapping GCP project: ${PROJECT_ID}"

# 1. Set active project.
gcloud config set project "${PROJECT_ID}"

# 2. Enable all required APIs.
echo "==> Enabling GCP APIs..."
gcloud services enable \
  run.googleapis.com \
  sqladmin.googleapis.com \
  secretmanager.googleapis.com \
  pubsub.googleapis.com \
  storage.googleapis.com \
  artifactregistry.googleapis.com \
  speech.googleapis.com \
  aiplatform.googleapis.com \
  cloudresourcemanager.googleapis.com \
  iam.googleapis.com \
  logging.googleapis.com \
  --project="${PROJECT_ID}"

echo "✓ APIs enabled"

# 3. Create Terraform state bucket (if not exists).
echo "==> Creating Terraform state bucket: ${TF_STATE_BUCKET}"
if ! gsutil ls "gs://${TF_STATE_BUCKET}" &>/dev/null; then
  gsutil mb -p "${PROJECT_ID}" -l "${REGION}" "gs://${TF_STATE_BUCKET}"
  gsutil versioning set on "gs://${TF_STATE_BUCKET}"
  echo "✓ State bucket created: gs://${TF_STATE_BUCKET}"
else
  echo "  Bucket already exists, skipping."
fi

# 4. Create Artifact Registry repository (if not exists).
echo "==> Creating Artifact Registry: wisdom-repo"
if ! gcloud artifacts repositories describe wisdom-repo \
    --location="${REGION}" --project="${PROJECT_ID}" &>/dev/null; then
  gcloud artifacts repositories create wisdom-repo \
    --repository-format=docker \
    --location="${REGION}" \
    --description="Wisdom Cognitive Runtime container images" \
    --project="${PROJECT_ID}"
  echo "✓ Artifact Registry created"
else
  echo "  Artifact Registry already exists, skipping."
fi

# 5. Configure Docker to use Artifact Registry.
echo "==> Configuring Docker authentication..."
gcloud auth configure-docker "${REGION}-docker.pkg.dev" --quiet

# 6. Update Terraform backend bucket in main.tf.
echo "==> Updating Terraform backend bucket reference..."
sed -i "s/REPLACE_WITH_YOUR_PROJECT_ID-tf-state/${TF_STATE_BUCKET}/g" terraform/main.tf

echo ""
echo "════════════════════════════════════════════════════════"
echo "✓ Bootstrap complete! Next steps:"
echo ""
echo "  1. Copy terraform/terraform.tfvars.example → terraform/terraform.tfvars"
echo "     and fill in your values."
echo ""
echo "  2. cd terraform && terraform init"
echo "  3. terraform plan -out=wisdom.tfplan"
echo "  4. terraform apply wisdom.tfplan"
echo ""
echo "  5. Connect your GitHub repo to Cloud Build:"
echo "     gcloud builds triggers create github \\"
echo "       --repo-name=wisdom \\"
echo "       --repo-owner=YOUR_GITHUB_ORG \\"
echo "       --branch-pattern='^main$' \\"
echo "       --build-config=cloudbuild.yaml \\"
echo "       --project=${PROJECT_ID}"
echo "════════════════════════════════════════════════════════"
