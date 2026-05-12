variable "project_id" {
  description = "The Google Cloud Project ID"
  type        = string
}

variable "region" {
  description = "The region to deploy the Cloud Run services"
  type        = string
  default     = "us-central1"
}
