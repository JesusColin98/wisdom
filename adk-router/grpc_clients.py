"""
grpc_clients.py — Lazy gRPC client singletons for all Go microservices.

Design rule: ADK Expert Agents must remain lightweight — all heavy data
operations (store, retrieve, analyze) are delegated to Go services via gRPC.

Usage:
    from grpc_clients import get_integrations_client, get_mastery_client
    result = await get_integrations_client().create_card(...)
"""
from __future__ import annotations

import json
import logging
from typing import Any
from functools import lru_cache

import grpc

from config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()


def _insecure_channel(url: str) -> grpc.Channel:
    """Creates an insecure gRPC channel. In production, replace with TLS."""
    return grpc.insecure_channel(url)


# ─── Lightweight HTTP-over-gRPC-REST clients ─────────────────────────────────
# Since we don't have generated Python protobuf stubs yet (protoc runs in Go),
# we use gRPC transcoding via a REST gateway pattern. The Go Thalamus service
# exposes an HTTP/2 gateway on port 8080 that mirrors gRPC calls as REST.
# This lets us call Go services from Python without maintaining dual proto stubs.
#
# When Python proto stubs are generated (scripts/gen_proto_py.sh), replace
# these with proper gRPC stub calls.

import urllib.request
import urllib.error


def _http_post(base_url: str, path: str, payload: dict[str, Any]) -> dict[str, Any]:
    """
    Simple synchronous HTTP POST to a gRPC-transcoded gateway endpoint.
    Replaces gRPC stub calls until Python proto stubs are generated.
    """
    url = f"http://{base_url}{path}"
    body = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=body, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        raise RuntimeError(f"gRPC-gateway error {e.code} at {url}: {e.read().decode()}") from e
    except Exception as e:
        raise RuntimeError(f"gRPC-gateway unreachable at {url}: {e}") from e


# ─── Cortex Client ───────────────────────────────────────────────────────────

class CortexClient:
    """Client for the Wisdom-Cortex storage substrate."""

    BASE = settings.cortex_grpc_url

    def memorize(self, node_type: str, payload: dict, confidence: float = 1.0) -> str:
        """Store a new node in Cortex. Returns the node ID."""
        result = _http_post(self.BASE, "/api/v1/memorize", {
            "type": node_type,
            "payload": payload,
            "confidence": confidence,
            "requires_human": False,
        })
        return result.get("id", "")

    def recall(self, node_id: str, depth: int = 1) -> dict:
        """Retrieve a node and its neighbors from Cortex."""
        return _http_post(self.BASE, "/api/v1/recall", {
            "id": node_id,
            "depth": depth,
        })

    def query_facts(self, query: str, filters: dict | None = None) -> list[dict]:
        """JSONB metadata search over Cortex nodes (legacy, non-semantic)."""
        result = _http_post(self.BASE, "/api/v1/query", {
            "query": query,
            "metadata_filters": filters or {},
        })
        return result.get("facts", [])

    def semantic_search(
        self,
        query: str,
        limit: int = 10,
        domain_filter: str = "",
        type_filter: str = "",
        min_score: float = 0.4,
    ) -> list[dict]:
        """
        Hybrid semantic search (pgvector HNSW + full-text) over Cortex nodes.

        Fallback chain:
          1. pgvector HNSW ANN + full-text RRF fusion (if embeddings exist)
          2. Full-text ts_content search (schema V3 tsvector)
          3. JSONB query_facts (if /cortex/search endpoint not yet deployed)

        Returns list of dicts: [{node, score, mode}, ...]
        """
        try:
            result = _http_post(self.BASE, "/api/v1/cortex/search", {
                "query": query,
                "limit": limit,
                "domain_filter": domain_filter,
                "type_filter": type_filter,
                "min_score": min_score,
            })
            results = result.get("results", [])
            logger.debug(
                "SemanticSearch: %d results for '%s' via mode=%s",
                len(results), query[:60], result.get("mode", "unknown"),
            )
            return results
        except RuntimeError as e:
            # Graceful fallback: semantic endpoint not available yet — use JSONB.
            logger.warning("semantic_search unavailable (%s) — falling back to query_facts", e)
            filters = {"domain": domain_filter} if domain_filter else None
            facts = self.query_facts(query, filters)
            return [{"node": f, "score": 0.5, "mode": "jsonb_fallback"} for f in facts]

    def search(self, query: str, domain: str = "", limit: int = 10) -> list[dict]:
        """Convenience wrapper: semantic_search scoped to a domain."""
        return self.semantic_search(query=query, domain_filter=domain, limit=limit)


# ─── Integrations Client ─────────────────────────────────────────────────────

class IntegrationsClient:
    """Client for the Wisdom-Integrations MCP bridge service."""

    BASE = settings.integrations_grpc_url

    def create_note(
        self,
        agent_name: str,
        user_id: str,
        path: str,
        title: str,
        content: str,
        tags: list[str] | None = None,
        relationships: list[str] | None = None,
        mastery_score: float = 0.5,
    ) -> dict:
        """Push a knowledge note to Obsidian via the MCP bridge."""
        return _http_post(self.BASE, "/api/v1/integrations/note", {
            "agent_name": agent_name,
            "user_id": user_id,
            "target_path": path,
            "content": content,
            "metadata": {
                "title": title,
                "tags": tags or [],
                "mastery_score": mastery_score,
            },
            "relationships": relationships or [],
        })

    def create_card(
        self,
        agent_name: str,
        user_id: str,
        deck_name: str,
        front: str,
        back: str,
        card_type: str = "BASIC",
        tags: list[str] | None = None,
        cloze_text: str = "",
        wisdom_node_id: str = "",
    ) -> dict:
        """Push an Anki flashcard via the MCP bridge."""
        return _http_post(self.BASE, "/api/v1/integrations/card", {
            "agent_name": agent_name,
            "user_id": user_id,
            "deck_name": deck_name,
            "card_type": card_type,
            "front": front,
            "back": back,
            "cloze_text": cloze_text,
            "tags": tags or ["Wisdom"],
            "wisdom_node_id": wisdom_node_id,
        })


# ─── Mastery Client ───────────────────────────────────────────────────────────

class MasteryClient:
    """Client for the Wisdom-Mastery SRS service."""

    BASE = settings.mastery_grpc_url

    def get_weaknesses(self, user_id: str, limit: int = 10) -> list[dict]:
        """Get the user's weakest concepts for curriculum prioritization."""
        result = _http_post(self.BASE, "/api/v1/mastery/weaknesses", {
            "user_id": user_id,
            "limit": limit,
        })
        return result.get("concepts", [])

    def get_due_cards(self, user_id: str) -> list[dict]:
        """Get concepts due for review (Metabolism SRS output)."""
        result = _http_post(self.BASE, "/api/v1/mastery/due", {
            "user_id": user_id,
        })
        return result.get("cards", [])


# ─── Researcher Client ────────────────────────────────────────────────────────

class ResearcherClient:
    """Client for the Wisdom-Researcher autonomous content gathering service."""

    BASE = settings.researcher_grpc_url

    def investigate(self, topic: str, domain: str, user_id: str, depth: int = 2) -> dict:
        """Trigger an asynchronous research job on a topic."""
        return _http_post(self.BASE, "/api/v1/research/investigate", {
            "topic": topic,
            "domain": domain,
            "user_id": user_id,
            "depth": depth,
        })


# ─── Singletons ──────────────────────────────────────────────────────────────

@lru_cache(maxsize=1)
def get_cortex_client() -> CortexClient:
    return CortexClient()


@lru_cache(maxsize=1)
def get_integrations_client() -> IntegrationsClient:
    return IntegrationsClient()


@lru_cache(maxsize=1)
def get_mastery_client() -> MasteryClient:
    return MasteryClient()


@lru_cache(maxsize=1)
def get_researcher_client() -> ResearcherClient:
    return ResearcherClient()
