variable "project_id" {
  description = "GCP project ID where all Wisdom resources will be deployed."
  type        = string
}

variable "region" {
  description = "GCP region for all resources."
  type        = string
  default     = "us-central1"
}

variable "gemini_api_key" {
  description = "Google Gemini API key for ADK Router and expert agents."
  type        = string
  sensitive   = true
}

variable "obsidian_api_key" {
  description = "Obsidian Local REST API plugin key for the Integrations MCP bridge."
  type        = string
  sensitive   = true
  default     = ""
}

variable "default_user_id" {
  description = "Default user ID for single-user deployments."
  type        = string
  default     = "default"
}

variable "memory_bank_corpus" {
  description = "Vertex AI Memory Bank corpus name for the ADK Router."
  type        = string
  default     = "wisdom-memory-bank"
}
