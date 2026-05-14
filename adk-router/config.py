"""
config.py — Centralized settings for the ADK Router.
Loaded from environment variables (or .env file in local dev).
All Cloud Run / GCP settings are validated at startup.
"""
from __future__ import annotations

from pydantic_settings import BaseSettings, SettingsConfigDict
from functools import lru_cache


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    # ── GCP ──────────────────────────────────────────────────────────────────
    gcp_project_id: str
    gcp_region: str = "us-central1"

    # ── Vertex AI ─────────────────────────────────────────────────────────────
    # Gemini Flash for fast intent routing.
    router_model: str = "gemini-2.0-flash"
    # Gemini Pro for deep expert reasoning.
    expert_model: str = "gemini-2.5-pro"

    # Vertex AI Memory Bank corpus name (shared across all experts; scoped per session).
    memory_bank_corpus: str = "wisdom-memory-bank"
    memory_bank_top_k: int = 5  # Max memories to retrieve per query.

    # ── Go Service gRPC URLs ──────────────────────────────────────────────────
    cortex_grpc_url: str = "localhost:50051"
    thalamus_grpc_url: str = "localhost:50052"
    mastery_grpc_url: str = "localhost:50053"
    researcher_grpc_url: str = "localhost:50054"
    curriculum_grpc_url: str = "localhost:50055"
    integrations_grpc_url: str = "localhost:50056"
    entity_grpc_url: str = "localhost:50057"

    # ── GCP Pub/Sub ───────────────────────────────────────────────────────────
    # Subscription that receives voice/text signals from Thalamus.
    pubsub_input_subscription: str = "adk-router-voice-input"
    # Topic to publish routing decisions for Portal observability.
    pubsub_routing_log_topic: str = "wisdom.router.decision_logged"

    # ── Server ────────────────────────────────────────────────────────────────
    port: int = 8081
    log_level: str = "INFO"

    # ── Default User ──────────────────────────────────────────────────────────
    # Used for memory bank session scoping.
    default_user_id: str = "default"


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings()
