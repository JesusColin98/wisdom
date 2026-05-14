output "cortex_url" {
  description = "Internal gRPC URL for Cortex substrate."
  value       = google_cloud_run_v2_service.wisdom_cortex.uri
}

output "thalamus_url" {
  description = "Public URL for the Thalamus API gateway (Portal + clients)."
  value       = google_cloud_run_v2_service.wisdom_thalamus.uri
}

output "mastery_url" {
  description = "Internal URL for the Mastery SRS service."
  value       = google_cloud_run_v2_service.wisdom_mastery.uri
}

output "researcher_url" {
  description = "Internal URL for the Researcher ingestion service."
  value       = google_cloud_run_v2_service.wisdom_researcher.uri
}

output "curriculum_url" {
  description = "Internal URL for the Curriculum planning service."
  value       = google_cloud_run_v2_service.wisdom_curriculum.uri
}

output "integrations_url" {
  description = "Internal URL for the Integrations MCP bridge."
  value       = google_cloud_run_v2_service.wisdom_integrations.uri
}

output "entity_url" {
  description = "Internal URL for the Entity extraction service."
  value       = google_cloud_run_v2_service.wisdom_entity.uri
}

output "adk_router_url" {
  description = "Internal URL for the ADK Router (Python cognitive layer)."
  value       = google_cloud_run_v2_service.wisdom_adk_router.uri
}

output "wisdom_sa_email" {
  description = "Wisdom Runtime service account email."
  value       = google_service_account.wisdom_sa.email
}

output "ingestion_bucket" {
  description = "GCS ingestion buffer bucket name (24h TTL)."
  value       = google_storage_bucket.ingestion_buffer.name
}

output "artifact_registry_url" {
  description = "Artifact Registry base URL for Docker images."
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/wisdom-repo"
}
