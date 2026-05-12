output "unified_engine_url" {
  description = "The public URL of the Unified Go Engine & Portal"
  value       = google_cloud_run_v2_service.wisdom_unified.uri
}

output "chat_service_url" {
  description = "The public URL of the Python Chat WebSockets proxy"
  value       = google_cloud_run_v2_service.wisdom_chat.uri
}

output "db_bucket_name" {
  description = "The name of the GCS bucket storing wisdom.db"
  value       = google_storage_bucket.wisdom_db_bucket.name
}
