#!/bin/bash
set -e

# Project Wisdom: Cortex Backup Script
# Synchronizes the SQLite database and RPForest index to Google Cloud Storage.
# Usage: ./backup_cortex.sh [BUCKET_NAME] [REGION]

BUCKET_NAME=${1:-"wisdom-backups-$(gcloud config get-value project)"}
REGION=${2:-"us-east1"} # Secondary region for DR
DB_PATH="wisdom.db"
INDEX_PATH="wisdom.db.rpforest"

echo "🧠 Starting Cortex Backup to GCS ($BUCKET_NAME in $REGION)..."

# 1. Ensure bucket exists in secondary region
gsutil mb -l $REGION gs://$BUCKET_NAME 2>/dev/null || echo "ℹ️ Bucket already exists."

# 2. SQLite Safe Backup (using .backup to avoid corruption)
echo "📦 Creating SQLite backup snapshot..."
sqlite3 $DB_PATH ".backup 'wisdom.db.snapshot'"

# 3. Upload to GCS
echo "📤 Uploading files to GCS..."
gsutil cp wisdom.db.snapshot gs://$BUCKET_NAME/backups/$(date +%Y%m%d-%H%M%S)/wisdom.db
if [ -f "$INDEX_PATH" ]; then
    gsutil cp $INDEX_PATH gs://$BUCKET_NAME/backups/$(date +%Y%m%d-%H%M%S)/wisdom.db.rpforest
fi

# 4. Cleanup local snapshot
rm wisdom.db.snapshot

echo "✅ Backup complete. Recovery available at gs://$BUCKET_NAME/backups/"
