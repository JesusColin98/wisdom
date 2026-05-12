terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# 1. Custom Service Account for Wisdom
resource "google_service_account" "wisdom_sa" {
  account_id   = "nexusstate-mcp-sa"
  display_name = "Wisdom MCP Service Account"
}

# 2. Secret Manager IAM Binding for Service Account
resource "google_secret_manager_secret_iam_member" "gemini_api_key_access" {
  secret_id = "GEMINI_API_KEY"
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.wisdom_sa.email}"
}

# 3. GCS Bucket for SQLite Persistence (GCS FUSE)
resource "google_storage_bucket" "wisdom_db_bucket" {
  name          = "${var.project_id}-wisdom-cortex-db"
  location      = var.region
  force_destroy = false
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_iam_member" "wisdom_bucket_access" {
  bucket = google_storage_bucket.wisdom_db_bucket.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.wisdom_sa.email}"
}

# 4. Cloud Run Service: Wisdom Unified (Go Engine + Portal)
resource "google_cloud_run_v2_service" "wisdom_unified" {
  name     = "wisdom-unified"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.wisdom_sa.email

    containers {
      image = "us-central1-docker.pkg.dev/${var.project_id}/wisdom-repo/wisdom-unified:latest"
      
      env {
        name  = "WISDOM_PORT"
        value = "8080"
      }
      env {
        name  = "WISDOM_DB_PATH"
        value = "/cortex-storage/wisdom.db"
      }

      volume_mounts {
        name       = "cortex-volume"
        mount_path = "/cortex-storage"
      }
      
      ports {
        container_port = 8080
      }
    }

    volumes {
      name = "cortex-volume"
      gcs {
        bucket = google_storage_bucket.wisdom_db_bucket.name
        read_only = false
      }
    }
  }
}

# 5. Cloud Run Service: Wisdom Chat (Python Voice Proxy)
resource "google_cloud_run_v2_service" "wisdom_chat" {
  name     = "wisdom-chat"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.wisdom_sa.email

    containers {
      image = "us-central1-docker.pkg.dev/${var.project_id}/wisdom-repo/wisdom-chat:latest"
      
      env {
        name  = "WISDOM_ENGINE_URL"
        value = google_cloud_run_v2_service.wisdom_unified.uri
      }

      env {
        name = "GEMINI_API_KEY"
        value_source {
          secret_key_ref {
            secret  = "GEMINI_API_KEY"
            version = "latest"
          }
        }
      }
      
      ports {
        container_port = 8080
      }
    }
  }
}

# 6. Make Unified Service Public (Or restrict based on IAP)
resource "google_cloud_run_service_iam_member" "unified_public_access" {
  location = google_cloud_run_v2_service.wisdom_unified.location
  project  = google_cloud_run_v2_service.wisdom_unified.project
  service  = google_cloud_run_v2_service.wisdom_unified.name
  role     = "roles/run.invoker"
  member   = "allUsers" # Replace with IAP Service Account later if restricted
}

# 7. Make Chat Service Public (Or restrict based on IAP)
resource "google_cloud_run_service_iam_member" "chat_public_access" {
  location = google_cloud_run_v2_service.wisdom_chat.location
  project  = google_cloud_run_v2_service.wisdom_chat.project
  service  = google_cloud_run_v2_service.wisdom_chat.name
  role     = "roles/run.invoker"
  member   = "allUsers" # Replace with IAP Service Account later if restricted
}
