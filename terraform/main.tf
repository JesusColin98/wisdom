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

# 3. Cloud SQL PostgreSQL Instance for Cortex Substrate
resource "google_sql_database_instance" "cortex_db_instance" {
  name             = "${var.project_id}-cortex-pg"
  database_version = "POSTGRES_15"
  region           = var.region

  settings {
    tier = "db-f1-micro" # Use appropriate tier for production
    ip_configuration {
      ipv4_enabled = true # Keep true if Cloud Run needs public IP access without VPC connector
    }
  }

  deletion_protection = false # Set to true in real production
}

resource "google_sql_database" "cortex_db" {
  name     = "cortexdb"
  instance = google_sql_database_instance.cortex_db_instance.name
}

resource "random_password" "cortex_db_password" {
  length  = 16
  special = true
}

resource "google_sql_user" "cortex_db_user" {
  name     = "cortexuser"
  instance = google_sql_database_instance.cortex_db_instance.name
  password = random_password.cortex_db_password.result
}

# Store DB Connection String in Secret Manager
resource "google_secret_manager_secret" "cortex_db_conn" {
  secret_id = "CORTEX_DB_CONN"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "cortex_db_conn_version" {
  secret      = google_secret_manager_secret.cortex_db_conn.id
  secret_data = "postgres://${google_sql_user.cortex_db_user.name}:${random_password.cortex_db_password.result}@${google_sql_database_instance.cortex_db_instance.public_ip_address}:5432/${google_sql_database.cortex_db.name}?sslmode=disable"
}

resource "google_secret_manager_secret_iam_member" "cortex_db_conn_access" {
  secret_id = google_secret_manager_secret.cortex_db_conn.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.wisdom_sa.email}"
}

# 4. Cloud Run Service: Cortex gRPC Substrate
resource "google_cloud_run_v2_service" "wisdom_cortex" {
  name     = "wisdom-cortex"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.wisdom_sa.email

    containers {
      image = "us-central1-docker.pkg.dev/${var.project_id}/wisdom-repo/wisdom-cortex:latest"
      
      env {
        name  = "PORT"
        value = "50051"
      }
      env {
        name = "DB_CONN_STRING"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.cortex_db_conn.secret_id
            version = "latest"
          }
        }
      }
      
      ports {
        name           = "h2c"
        container_port = 50051
      }
    }
  }
}

# 5. Cloud Run Service: Thalamus Gateway
resource "google_cloud_run_v2_service" "wisdom_thalamus" {
  name     = "wisdom-thalamus"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.wisdom_sa.email

    containers {
      image = "us-central1-docker.pkg.dev/${var.project_id}/wisdom-repo/wisdom-thalamus:latest"
      
      env {
        name  = "PORT"
        value = "50052"
      }
      env {
        name  = "CORTEX_GRPC_URL"
        value = replace(google_cloud_run_v2_service.wisdom_cortex.uri, "https://", "")
      }
      
      ports {
        name           = "h2c"
        container_port = 50052
      }
    }
  }
}

# 6. Cloud Run Service: Wisdom Chat (Python Voice Proxy)
resource "google_cloud_run_v2_service" "wisdom_chat" {
  name     = "wisdom-chat"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.wisdom_sa.email

    containers {
      image = "us-central1-docker.pkg.dev/${var.project_id}/wisdom-repo/wisdom-chat:latest"
      
      env {
        name  = "WISDOM_THALAMUS_URL"
        value = google_cloud_run_v2_service.wisdom_thalamus.uri
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

# 7. Make Cortex Service Private to VPC/Other services (Optional/Best Practice)
# For now keeping it public to test, but ideally only Thalamus talks to Cortex
resource "google_cloud_run_service_iam_member" "cortex_public_access" {
  location = google_cloud_run_v2_service.wisdom_cortex.location
  project  = google_cloud_run_v2_service.wisdom_cortex.project
  service  = google_cloud_run_v2_service.wisdom_cortex.name
  role     = "roles/run.invoker"
  member   = "allUsers" # Replace with specific service account later
}

# 8. Make Thalamus Service Public
resource "google_cloud_run_service_iam_member" "thalamus_public_access" {
  location = google_cloud_run_v2_service.wisdom_thalamus.location
  project  = google_cloud_run_v2_service.wisdom_thalamus.project
  service  = google_cloud_run_v2_service.wisdom_thalamus.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# 11. Cloud Run Job: Researcher Scraper
resource "google_cloud_run_v2_job" "wisdom_researcher" {
  name     = "wisdom-researcher"
  location = var.region

  template {
    template {
      service_account = google_service_account.wisdom_sa.email
      
      containers {
        image = "us-central1-docker.pkg.dev/${var.project_id}/wisdom-repo/wisdom-researcher:latest"
        
        env {
          name  = "TARGET_URLS"
          value = "https://example.com" # Can be overridden per execution
        }
        env {
          name  = "NATS_URL"
          value = "nats://demo.nats.io:4222" # Placeholder
        }
      }
    }
  }
}
