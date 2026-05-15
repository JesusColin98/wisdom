"""
main.py — ADK Router FastAPI entrypoint.

Exposes two interfaces:
  1. HTTP POST /route — for direct REST calls (Portal, development)
  2. GCP Pub/Sub push subscription — for voice input from Thalamus (production)

The Pub/Sub subscription receives wisdom.voice.transcribed events from the
Go Thalamus service after Google Cloud STT processing.
"""
from __future__ import annotations

import asyncio
import base64
import json
import logging
import uuid
from contextlib import asynccontextmanager
from typing import Any

import structlog
from fastapi import FastAPI, HTTPException, Request, status
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from config import get_settings
from router import CognitiveRouter

# ─── Logging Setup ────────────────────────────────────────────────────────────
structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer(),
    ],
)
logger = structlog.get_logger()
settings = get_settings()

# ─── Application Lifecycle ────────────────────────────────────────────────────
router: CognitiveRouter | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize the CognitiveRouter on startup."""
    global router
    logger.info("ADK Router starting up...", project=settings.gcp_project_id)
    router = CognitiveRouter()
    logger.info("CognitiveRouter ready.")
    yield
    logger.info("ADK Router shutting down.")


# ─── FastAPI App ──────────────────────────────────────────────────────────────
app = FastAPI(
    title="Wisdom ADK Router",
    description="Cognitive routing layer for the Wisdom Knowledge Runtime.",
    version="1.0.0",
    lifespan=lifespan,
)


# ─── Request / Response Models ────────────────────────────────────────────────

class RouteRequest(BaseModel):
    """Direct routing request (REST API)."""
    input: str
    user_id: str = settings.default_user_id
    session_id: str | None = None
    metadata: dict[str, Any] | None = None


class RouteResponse(BaseModel):
    """Routing response returned to the caller."""
    domain: str
    agent: str
    confidence: float
    response: str
    session_id: str
    elapsed_ms: int
    memories_used: int


class PubSubMessage(BaseModel):
    """GCP Pub/Sub push message envelope."""
    message: dict[str, Any]
    subscription: str


class ExpertConfigRequest(BaseModel):
    """Configuration for a new dynamic domain expert."""
    id: str
    agent: str | None = None
    description: str
    system_instruction: str | None = None
    keywords: list[str] = []
    memory_topics: list[str] = []
    anki_deck_prefix: str | None = None
    obsidian_folder: str | None = None


# ─── Routes ───────────────────────────────────────────────────────────────────

@app.get("/health")
async def health():
    """Health check endpoint — used by Cloud Run startup probe."""
    return {"status": "ok", "service": "wisdom-adk-router"}


@app.post("/route", response_model=RouteResponse)
async def route_input(req: RouteRequest) -> RouteResponse:
    """
    Route a user input to the appropriate domain expert and return its response.

    This is the primary REST endpoint, used by:
    - The Thalamus gateway (for synchronous HTTP routing)
    - The Portal frontend (for real-time test routing)
    - Development / curl testing
    """
    if not router:
        raise HTTPException(status_code=503, detail="Router not initialized")

    if not req.input.strip():
        raise HTTPException(status_code=400, detail="'input' must not be empty")

    try:
        result = await router.route(
            user_input=req.input,
            user_id=req.user_id,
            session_id=req.session_id,
            metadata=req.metadata,
        )
        return RouteResponse(**result)
    except Exception as e:
        logger.exception("Routing error", error=str(e))
        raise HTTPException(status_code=500, detail=f"Routing failed: {e}") from e


@app.post("/pubsub/voice-input", status_code=status.HTTP_204_NO_CONTENT)
async def pubsub_voice_input(req: Request):
    """
    GCP Pub/Sub push subscription endpoint.
    Receives wisdom.voice.transcribed events from the Thalamus service.

    Event payload (base64-encoded message.data):
    {
        "type": "wisdom.voice.transcribed",
        "text": "how does the caro-kann defense work?",
        "user_id": "user-123",
        "session_id": "session-abc",
        "confidence": 0.97,
        "language_code": "en-US"
    }

    Returns 204 on success (Pub/Sub acks the message).
    Returns 4xx/5xx to trigger Pub/Sub retry.
    """
    if not router:
        raise HTTPException(status_code=503, detail="Router not initialized")

    try:
        body = await req.json()
        message = body.get("message", {})

        # Decode base64 Pub/Sub message data.
        encoded_data = message.get("data", "")
        if not encoded_data:
            logger.warning("Pub/Sub message has no data, acking.")
            return JSONResponse(status_code=204, content=None)

        payload = json.loads(base64.b64decode(encoded_data).decode("utf-8"))

        text = payload.get("text", "").strip()
        if not text:
            logger.warning("Pub/Sub message has empty text, acking.")
            return JSONResponse(status_code=204, content=None)

        user_id = payload.get("user_id", settings.default_user_id)
        session_id = payload.get("session_id", f"pubsub-{uuid.uuid4().hex[:8]}")

        logger.info(
            "Voice input received via Pub/Sub",
            user=user_id,
            session=session_id,
            text_len=len(text),
        )

        # Route asynchronously — Pub/Sub doesn't wait for the result.
        asyncio.create_task(
            router.route(
                user_input=text,
                user_id=user_id,
                session_id=session_id,
                metadata={"source": "pubsub", "confidence": payload.get("confidence")},
            )
        )

        return JSONResponse(status_code=204, content=None)

    except json.JSONDecodeError as e:
        logger.warning("Invalid Pub/Sub payload", error=str(e))
        return JSONResponse(status_code=204, content=None)  # Ack to avoid infinite retry.
    except Exception as e:
        logger.exception("Pub/Sub handler error", error=str(e))
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/domains")
async def list_domains():
    """Return the active domain configuration (including dynamic ones)."""
    if not router:
        # Fallback if router not initialized
        from pathlib import Path
        import json
        return json.loads((Path(__file__).parent / "domains.json").read_text())
    
    return {"domains": router._domains}


@app.post("/api/v1/experts")
async def register_expert(config: ExpertConfigRequest):
    """
    Register a new dynamic domain expert.
    Persists the config in Cortex and reloads the router.
    """
    if not router:
        raise HTTPException(status_code=503, detail="Router not initialized")

    try:
        # Ensure ID is uppercase
        config.id = config.id.upper()
        
        # Default values if missing
        if not config.agent:
            config.agent = f"{config.id.capitalize()}Expert"
        if not config.anki_deck_prefix:
            config.anki_deck_prefix = f"Wisdom::{config.id.capitalize()}"
        if not config.obsidian_folder:
            config.obsidian_folder = f"{config.id.capitalize()}/"
        if not config.memory_topics:
            config.memory_topics = [f"{config.id}_INSIGHTS"]

        # 1. Store in Cortex as a DomainConfig node
        # We use node_type='DomainConfig' which our router looks for
        from grpc_clients import get_cortex_client
        cortex = get_cortex_client()
        
        cortex.memorize(
            node_type="DomainConfig",
            payload=config.dict(),
            confidence=1.0
        )

        # 2. Trigger router reload
        router.load_domains()

        return {
            "status": "success",
            "message": f"Expert '{config.id}' registered and router reloaded",
            "domain": config.id
        }
    except Exception as e:
        logger.exception("Failed to register expert", error=str(e))
        raise HTTPException(status_code=500, detail=f"Registration failed: {e}")


@app.post("/session/{session_id}/generate-memories")
async def generate_session_memories(session_id: str, user_id: str = settings.default_user_id):
    """
    Trigger memory generation for a completed session.
    Call this at the end of a study session to distill interactions into
    long-term Memory Bank entries for each domain touched.
    """
    if not router:
        raise HTTPException(status_code=503, detail="Router not initialized")

    results = {}
    for domain_id, expert in router._experts.items():
        count = router.memory_bank.generate_memories(
            session_id=session_id,
            agent_scope=expert.memory_scope,
            topics=expert.memory_topics,
        )
        if count > 0:
            results[domain_id] = count

    return {"session_id": session_id, "generated_memories": results}


# ─── Entry Point ──────────────────────────────────────────────────────────────
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=settings.port,
        log_level=settings.log_level.lower(),
        reload=False,
    )
