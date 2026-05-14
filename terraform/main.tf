# ─────────────────────────────────────────────────────────────────────────────
# Wisdom Cognitive Runtime — Terraform Infrastructure
#
# Provisions the full GCP infrastructure for the distributed microservices mesh:
#   - Service Account + IAM roles
#   - Artifact Registry
#   - Cloud SQL (PostgreSQL 15 for Cortex)
#   - Secret Manager entries
#   - GCP Pub/Sub topics & push subscriptions
#   - Cloud Run services (all 8 microservices)
#   - GCS ingestion buffer bucket
#
# Rules:
#   - All secrets are stored in Secret Manager (never as plain env vars)
#   - Services communicate via gRPC over Cloud Run internal URLs
#   - Only Thalamus and Portal are publicly accessible
#   - ADK Router receives Pub/Sub push (private)
# ─────────────────────────────────────────────────────────────────────────────

terraform {
  required_version = ">= 1.7.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.30"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  backend "gcs" {
    bucket = "REPLACE_WITH_YOUR_PROJECT_ID-tf-state"
    prefix = "wisdom/terraform/state"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# ─────────────────────────────────────────────────────────────────────────────
# Locals
# ─────────────────────────────────────────────────────────────────────────────
locals {
  image_base = "${var.region}-docker.pkg.dev/${var.project_id}/wisdom-repo"
  services   = ["cortex", "thalamus", "mastery", "researcher", "curriculum", "integrations", "entity"]
}

# ─────────────────────────────────────────────────────────────────────────────
# 1. Artifact Registry
# ─────────────────────────────────────────────────────────────────────────────
resource "google_artifact_registry_repository" "wisdom_repo" {
  location      = var.region
  repository_id = "wisdom-repo"
  description   = "Wisdom Cognitive Runtime container images"
  format        = "DOCKER"
}

# ─────────────────────────────────────────────────────────────────────────────
# 2. Service Account + IAM
# ─────────────────────────────────────────────────────────────────────────────
resource "google_service_account" "wisdom_sa" {
  account_id   = "wisdom-runtime-sa"
  display_name = "Wisdom Cognitive Runtime Service Account"
}

# Bulk IAM roles for the service account.
locals {
  sa_roles = [
    "roles/secretmanager.secretAccessor",
    "roles/pubsub.publisher",
    "roles/pubsub.subscriber",
    "roles/cloudsql.client",
    "roles/storage.objectAdmin",
    "roles/aiplatform.user",          # Vertex AI Memory Bank + Gemini
    "roles/speech.client",            # Cloud STT V2
    "roles/run.invoker",              # Internal Cloud Run invocation
    "roles/logging.logWriter",
  ]
}

resource "google_project_iam_member" "wisdom_sa_roles" {
  for_each = toset(local.sa_roles)
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.wisdom_sa.email}"
}

# ─────────────────────────────────────────────────────────────────────────────
# 3. Cloud SQL — PostgreSQL 15 for Cortex Substrate
# ─────────────────────────────────────────────────────────────────────────────
resource "google_sql_database_instance" "cortex_pg" {
  name             = "wisdom-cortex-pg"
  database_version = "POSTGRES_15"
  region           = var.region

  settings {
    tier              = "db-g1-small"
    availability_type = "ZONAL"

    backup_configuration {
      enabled            = true
      start_time         = "03:00"
      binary_log_enabled = false
    }

    ip_configuration {
      ipv4_enabled = false
      # Private IP only — accessed via Cloud SQL Auth Proxy on Cloud Run.
      private_network = "projects/${var.project_id}/global/networks/default"
      ssl_mode        = "ENCRYPTED_ONLY"
    }

    database_flags {
      name  = "max_connections"
      value = "100"
    }
  }

  deletion_protection = true
}

resource "google_sql_database" "cortex_db" {
  name     = "cortexdb"
  instance = google_sql_database_instance.cortex_pg.name
}

resource "random_password" "cortex_db_password" {
  length           = 24
  special          = true
  override_special = "!#$%&*()-_=+[]{}:?"
}

resource "google_sql_user" "cortex_db_user" {
  name     = "cortexuser"
  instance = google_sql_database_instance.cortex_pg.name
  password = random_password.cortex_db_password.result
}

# ─────────────────────────────────────────────────────────────────────────────
# 4. Secret Manager — All secrets
# ─────────────────────────────────────────────────────────────────────────────
locals {
  secrets = {
    "CORTEX_DB_CONN"      = "postgres://cortexuser:${random_password.cortex_db_password.result}@127.0.0.1:5432/cortexdb?sslmode=require"
    "GEMINI_API_KEY"      = var.gemini_api_key
    "OBSIDIAN_API_KEY"    = var.obsidian_api_key
    "DEFAULT_USER_ID"     = var.default_user_id
    "GCP_PROJECT_ID"      = var.project_id
    "MEMORY_BANK_CORPUS"  = var.memory_bank_corpus
  }
}

resource "google_secret_manager_secret" "wisdom_secrets" {
  for_each  = local.secrets
  secret_id = each.key
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "wisdom_secret_versions" {
  for_each    = local.secrets
  secret      = google_secret_manager_secret.wisdom_secrets[each.key].id
  secret_data = each.value
}

# ─────────────────────────────────────────────────────────────────────────────
# 5. GCS Ingestion Buffer (24h TTL)
# ─────────────────────────────────────────────────────────────────────────────
resource "google_storage_bucket" "ingestion_buffer" {
  name                        = "${var.project_id}-wisdom-ingestion"
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = false

  lifecycle_rule {
    action { type = "Delete" }
    condition { age = 1 } # 24 hours TTL.
  }
}

# ─────────────────────────────────────────────────────────────────────────────
# 6. GCP Pub/Sub Topics & Subscriptions
# ─────────────────────────────────────────────────────────────────────────────
locals {
  pubsub_topics = [
    "wisdom.voice.transcribed",         # Thalamus → ADK Router
    "wisdom.knowledge.ingested",        # Researcher → Cerebellum
    "wisdom.researcher.scrape_progress", # Researcher → Portal
    "wisdom.integrations.sync_ready",   # Integrations → Portal
    "wisdom.integrations.item_synced",  # Integrations → Portal
    "wisdom.router.decision_logged",    # ADK Router → Portal
    "wisdom.mastery.reviewed",          # Mastery → Cerebellum
  ]
}

resource "google_pubsub_topic" "wisdom_topics" {
  for_each = toset(local.pubsub_topics)
  name     = each.value

  message_retention_duration = "86400s" # 24 hours.
}

# Push subscription: voice.transcribed → ADK Router
resource "google_pubsub_subscription" "adk_router_voice_input" {
  name  = "adk-router-voice-input"
  topic = google_pubsub_topic.wisdom_topics["wisdom.voice.transcribed"].name

  push_config {
    push_endpoint = "${google_cloud_run_v2_service.wisdom_adk_router.uri}/pubsub/voice-input"
    oidc_token {
      service_account_email = google_service_account.wisdom_sa.email
    }
  }

  ack_deadline_seconds       = 60
  message_retention_duration = "86400s"
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "300s"
  }
}

# Pull subscription: knowledge.ingested → Cerebellum worker
resource "google_pubsub_subscription" "cerebellum_knowledge" {
  name  = "cerebellum-knowledge-ingested"
  topic = google_pubsub_topic.wisdom_topics["wisdom.knowledge.ingested"].name

  ack_deadline_seconds       = 120
  message_retention_duration = "604800s" # 7 days.
  retry_policy {
    minimum_backoff = "5s"
    maximum_backoff = "60s"
  }
}

# ─────────────────────────────────────────────────────────────────────────────
# 7. Cloud Run Services — Go Microservices
# ─────────────────────────────────────────────────────────────────────────────

# Helper: secret ref shorthand.
locals {
  db_secret_ref = {
    secret  = "CORTEX_DB_CONN"
    version = "latest"
  }
}

# ── 7.1 Cortex ────────────────────────────────────────────────────────────────
resource "google_cloud_run_v2_service" "wisdom_cortex" {
  name     = "wisdom-cortex"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_ONLY" # gRPC internal only.

  template {
    service_account = google_service_account.wisdom_sa.email
    max_instance_request_concurrency = 200

    scaling {
      min_instance_count = 1
      max_instance_count = 5
    }

    containers {
      image = "${local.image_base}/wisdom-cortex:latest"
      ports { name = "h2c"; container_port = 50051 }

      env { name = "PORT"; value = "50051" }
      env {
        name = "DB_CONN_STRING"
        value_source { secret_key_ref { secret = "CORTEX_DB_CONN"; version = "latest" } }
      }

      resources {
        limits = { cpu = "1", memory = "512Mi" }
      }

      startup_probe {
        grpc { port = 50051 }
        initial_delay_seconds = 5
        period_seconds        = 5
        failure_threshold     = 10
      }
    }

    volumes {
      name = "cloudsql"
      cloud_sql_instance { instances = [google_sql_database_instance.cortex_pg.connection_name] }
    }
  }
}

# ── 7.2 Thalamus (public — API gateway) ──────────────────────────────────────
resource "google_cloud_run_v2_service" "wisdom_thalamus" {
  name     = "wisdom-thalamus"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL" # Public: Portal + clients connect here.

  template {
    service_account = google_service_account.wisdom_sa.email

    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }

    containers {
      image = "${local.image_base}/wisdom-thalamus:latest"
      ports { name = "h2c"; container_port = 50052 }

      env { name = "PORT"; value = "50052" }
      env { name = "CORTEX_GRPC_URL"; value = replace(google_cloud_run_v2_service.wisdom_cortex.uri, "https://", "") }
      env { name = "GCP_PROJECT_ID"; value = var.project_id }

      resources { limits = { cpu = "1", memory = "256Mi" } }

      startup_probe {
        grpc { port = 50052 }
        initial_delay_seconds = 3
        period_seconds        = 5
        failure_threshold     = 6
      }
    }
  }
}

# ── 7.3 Mastery ───────────────────────────────────────────────────────────────
resource "google_cloud_run_v2_service" "wisdom_mastery" {
  name     = "wisdom-mastery"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    service_account = google_service_account.wisdom_sa.email

    scaling { min_instance_count = 0; max_instance_count = 3 }

    containers {
      image = "${local.image_base}/wisdom-mastery:latest"
      ports { name = "h2c"; container_port = 50053 }

      env { name = "PORT"; value = "50053" }
      env { name = "CORTEX_GRPC_URL"; value = replace(google_cloud_run_v2_service.wisdom_cortex.uri, "https://", "") }

      resources { limits = { cpu = "500m", memory = "256Mi" } }

      startup_probe {
        grpc { port = 50053 }
        initial_delay_seconds = 3
        period_seconds        = 5
        failure_threshold     = 6
      }
    }
  }
}

# ── 7.4 Researcher ────────────────────────────────────────────────────────────
resource "google_cloud_run_v2_service" "wisdom_researcher" {
  name     = "wisdom-researcher"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    service_account = google_service_account.wisdom_sa.email

    scaling { min_instance_count = 0; max_instance_count = 5 }

    containers {
      image = "${local.image_base}/wisdom-researcher:latest"
      ports { name = "h2c"; container_port = 50054 }

      env { name = "PORT"; value = "50054" }
      env { name = "CORTEX_GRPC_URL"; value = replace(google_cloud_run_v2_service.wisdom_cortex.uri, "https://", "") }
      env { name = "GCP_PROJECT_ID"; value = var.project_id }
      env { name = "PUBSUB_TOPIC_KNOWLEDGE_INGESTED"; value = "wisdom.knowledge.ingested" }
      env { name = "INGESTION_BUCKET"; value = google_storage_bucket.ingestion_buffer.name }

      resources { limits = { cpu = "1", memory = "512Mi" } }

      startup_probe {
        grpc { port = 50054 }
        initial_delay_seconds = 3
        period_seconds        = 5
        failure_threshold     = 6
      }
    }
  }
}

# ── 7.5 Curriculum ────────────────────────────────────────────────────────────
resource "google_cloud_run_v2_service" "wisdom_curriculum" {
  name     = "wisdom-curriculum"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    service_account = google_service_account.wisdom_sa.email

    scaling { min_instance_count = 0; max_instance_count = 3 }

    containers {
      image = "${local.image_base}/wisdom-curriculum:latest"
      ports { name = "h2c"; container_port = 50055 }

      env { name = "PORT"; value = "50055" }
      env { name = "CORTEX_GRPC_URL"; value = replace(google_cloud_run_v2_service.wisdom_cortex.uri, "https://", "") }
      env { name = "MASTERY_GRPC_URL"; value = replace(google_cloud_run_v2_service.wisdom_mastery.uri, "https://", "") }

      resources { limits = { cpu = "500m", memory = "256Mi" } }

      startup_probe {
        grpc { port = 50055 }
        initial_delay_seconds = 3
        period_seconds        = 5
        failure_threshold     = 6
      }
    }
  }
}

# ── 7.6 Integrations (MCP Bridge) ────────────────────────────────────────────
resource "google_cloud_run_v2_service" "wisdom_integrations" {
  name     = "wisdom-integrations"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    service_account = google_service_account.wisdom_sa.email

    scaling { min_instance_count = 1; max_instance_count = 3 }

    containers {
      image = "${local.image_base}/wisdom-integrations:latest"
      ports { name = "h2c"; container_port = 50056 }

      env { name = "PORT"; value = "50056" }
      env { name = "CORTEX_GRPC_URL"; value = replace(google_cloud_run_v2_service.wisdom_cortex.uri, "https://", "") }
      env { name = "MASTERY_GRPC_URL"; value = replace(google_cloud_run_v2_service.wisdom_mastery.uri, "https://", "") }
      # MCP servers run on host — not accessible from Cloud Run. Use PENDING_SYNC queue.
      env { name = "OBSIDIAN_MCP_URL"; value = "disabled" }
      env { name = "ANKI_MCP_URL"; value = "disabled" }
      env {
        name = "DEFAULT_USER_ID"
        value_source { secret_key_ref { secret = "DEFAULT_USER_ID"; version = "latest" } }
      }
      env {
        name = "OBSIDIAN_API_KEY"
        value_source { secret_key_ref { secret = "OBSIDIAN_API_KEY"; version = "latest" } }
      }

      resources { limits = { cpu = "500m", memory = "256Mi" } }

      startup_probe {
        grpc { port = 50056 }
        initial_delay_seconds = 3
        period_seconds        = 5
        failure_threshold     = 6
      }
    }
  }
}

# ── 7.7 Entity ────────────────────────────────────────────────────────────────
resource "google_cloud_run_v2_service" "wisdom_entity" {
  name     = "wisdom-entity"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    service_account = google_service_account.wisdom_sa.email

    scaling { min_instance_count = 0; max_instance_count = 3 }

    containers {
      image = "${local.image_base}/wisdom-entity:latest"
      ports { name = "h2c"; container_port = 50057 }

      env { name = "PORT"; value = "50057" }
      env { name = "CORTEX_GRPC_URL"; value = replace(google_cloud_run_v2_service.wisdom_cortex.uri, "https://", "") }

      resources { limits = { cpu = "500m", memory = "256Mi" } }

      startup_probe {
        grpc { port = 50057 }
        initial_delay_seconds = 3
        period_seconds        = 5
        failure_threshold     = 6
      }
    }
  }
}

# ── 7.8 ADK Router (Python) ───────────────────────────────────────────────────
resource "google_cloud_run_v2_service" "wisdom_adk_router" {
  name     = "wisdom-adk-router"
  location = var.region
  # Private: only receives Pub/Sub push and internal Thalamus calls.
  ingress  = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  template {
    service_account = google_service_account.wisdom_sa.email

    scaling { min_instance_count = 1; max_instance_count = 5 }

    containers {
      image = "${local.image_base}/wisdom-adk-router:latest"
      ports { container_port = 8081 }

      env { name = "PORT";           value = "8081" }
      env { name = "GCP_PROJECT_ID"; value = var.project_id }
      env { name = "GCP_REGION";     value = var.region }

      env { name = "CORTEX_GRPC_URL";       value = replace(google_cloud_run_v2_service.wisdom_cortex.uri, "https://", "") }
      env { name = "MASTERY_GRPC_URL";      value = replace(google_cloud_run_v2_service.wisdom_mastery.uri, "https://", "") }
      env { name = "RESEARCHER_GRPC_URL";   value = replace(google_cloud_run_v2_service.wisdom_researcher.uri, "https://", "") }
      env { name = "CURRICULUM_GRPC_URL";   value = replace(google_cloud_run_v2_service.wisdom_curriculum.uri, "https://", "") }
      env { name = "INTEGRATIONS_GRPC_URL"; value = replace(google_cloud_run_v2_service.wisdom_integrations.uri, "https://", "") }
      env { name = "ENTITY_GRPC_URL";       value = replace(google_cloud_run_v2_service.wisdom_entity.uri, "https://", "") }

      env { name = "PUBSUB_INPUT_SUBSCRIPTION";  value = google_pubsub_subscription.adk_router_voice_input.name }
      env { name = "PUBSUB_ROUTING_LOG_TOPIC";   value = "wisdom.router.decision_logged" }
      env { name = "ROUTER_MODEL";               value = "gemini-2.0-flash" }
      env { name = "EXPERT_MODEL";               value = "gemini-2.5-pro" }

      env {
        name = "MEMORY_BANK_CORPUS"
        value_source { secret_key_ref { secret = "MEMORY_BANK_CORPUS"; version = "latest" } }
      }
      env {
        name = "DEFAULT_USER_ID"
        value_source { secret_key_ref { secret = "DEFAULT_USER_ID"; version = "latest" } }
      }

      resources { limits = { cpu = "2", memory = "2Gi" } }

      startup_probe {
        http_get { path = "/health"; port = 8081 }
        initial_delay_seconds = 10
        period_seconds        = 5
        failure_threshold     = 12
      }

      liveness_probe {
        http_get { path = "/health"; port = 8081 }
        period_seconds    = 30
        failure_threshold = 3
      }
    }
  }

  depends_on = [
    google_pubsub_subscription.adk_router_voice_input,
  ]
}

# ─────────────────────────────────────────────────────────────────────────────
# 8. IAM — Service-to-Service Invocation
# ─────────────────────────────────────────────────────────────────────────────
locals {
  internal_services = [
    google_cloud_run_v2_service.wisdom_cortex.name,
    google_cloud_run_v2_service.wisdom_mastery.name,
    google_cloud_run_v2_service.wisdom_researcher.name,
    google_cloud_run_v2_service.wisdom_curriculum.name,
    google_cloud_run_v2_service.wisdom_integrations.name,
    google_cloud_run_v2_service.wisdom_entity.name,
    google_cloud_run_v2_service.wisdom_adk_router.name,
  ]
}

# The Wisdom SA can invoke all internal services.
resource "google_cloud_run_v2_service_iam_member" "internal_invoker" {
  for_each = toset(local.internal_services)
  location = var.region
  name     = each.value
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.wisdom_sa.email}"
}

# Pub/Sub SA can push to ADK Router (needed for push subscriptions).
resource "google_cloud_run_v2_service_iam_member" "pubsub_push_invoker" {
  location = var.region
  name     = google_cloud_run_v2_service.wisdom_adk_router.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

# Thalamus is public (Portal + mobile clients).
resource "google_cloud_run_v2_service_iam_member" "thalamus_public" {
  location = var.region
  name     = google_cloud_run_v2_service.wisdom_thalamus.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

data "google_project" "current" {}
