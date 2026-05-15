"""
router.py — Intent classifier and expert dispatcher for the Wisdom ADK Router.

The Router is a lightweight Gemini Flash agent that:
  1. Classifies the domain of a user's input (CHESS, FINANCE, LANGUAGE, TECH, GENERAL)
  2. Dispatches to the appropriate Domain Expert agent
  3. Logs the routing decision to Pub/Sub for Portal observability

Design rule: The Router itself does NO knowledge work. It only classifies and delegates.
Heavy reasoning happens in the Domain Experts (Gemini Pro).
"""
from __future__ import annotations

import json
import logging
import time
import uuid
from pathlib import Path
from typing import Any

from google import adk
from google.cloud import pubsub_v1

from config import get_settings
from memory_bank import MemoryBank
from experts import ChessExpert, FinanceExpert, LanguageExpert, TechExpert, BaseExpert, DynamicExpert
from grpc_clients import get_cortex_client

logger = logging.getLogger(__name__)
settings = get_settings()

# Load baseline domain configuration from domains.json.
_DOMAINS_PATH = Path(__file__).parent / "domains.json"
_BASELINE_DOMAINS: list[dict] = json.loads(_DOMAINS_PATH.read_text())["domains"]

def _build_router_instruction(domains_list: list[dict]) -> str:
    domain_ids = [d["id"] for d in domains_list]
    domain_rules = "\n".join([f"{i+4}. Use {d['id']} for: {d.get('description', '')}" for i, d in enumerate(domains_list)])
    return f"""
You are the Wisdom Cognitive Router — a fast, precise intent classifier.

## Your ONLY Job
Classify user input into exactly ONE domain from this list:
{json.dumps(domain_ids)}

## Classification Rules
1. Return ONLY a JSON object: {{"domain": "DOMAIN_ID", "confidence": 0.95, "reason": "one sentence"}}
2. Do NOT provide advice, explanations, or learning content.
3. If you are unsure, return GENERAL.
{domain_rules}

## Examples
- "How does the Caro-Kann defend against e4?" → CHESS
- "What is the P/E ratio of Apple?" → FINANCE  
- "How do I conjugate tener in the subjunctive?" → LANGUAGE
- "Explain the time complexity of QuickSort" → TECH
- "What are the benefits of sleep?" → GENERAL
"""

class CognitiveRouter:
    """
    The central routing hub — classifies intent and dispatches to domain experts.
    One instance per ADK Router process.
    """

    def __init__(self) -> None:
        self.memory_bank = MemoryBank()
        self._cortex = get_cortex_client()
        self._pubsub = pubsub_v1.PublisherClient()
        self._routing_topic = self._pubsub.topic_path(
            settings.gcp_project_id,
            settings.pubsub_routing_log_topic,
        )
        
        self._domains: list[dict] = []
        self._keyword_index: dict[str, str] = {}
        self._experts: dict[str, BaseExpert] = {}
        self._router_agent = None

        self.load_domains()

    def load_domains(self) -> None:
        """Loads baseline domains, fetches dynamic domains from Cortex, and rebuilds router agents."""
        logger.info("Loading baseline and dynamic domains...")
        
        self._domains = list(_BASELINE_DOMAINS)
        
        # Try fetching dynamic domains from Cortex
        try:
            dynamic_facts = self._cortex.query_facts(query="", filters={"node_type": "DomainConfig"})
            for fact in dynamic_facts:
                if "payload" in fact and "id" in fact["payload"]:
                    domain_config = fact["payload"]
                    # If not already present in baseline, add it
                    if not any(d["id"] == domain_config["id"] for d in self._domains):
                        self._domains.append(domain_config)
            logger.info("Fetched %d dynamic domains from Cortex.", len(dynamic_facts))
        except Exception as e:
            logger.warning("Failed to fetch dynamic domains from Cortex. Proceeding with baseline. Error: %s", e)

        # Rebuild Keyword Index
        self._keyword_index = {}
        for domain in self._domains:
            for kw in domain.get("keywords", []):
                self._keyword_index[kw.lower()] = domain["id"]

        # Rebuild the lightweight router agent (Gemini Flash).
        self._router_agent = adk.Agent(
            name="CognitiveRouter",
            model=settings.router_model,
            instruction=_build_router_instruction(self._domains),
        )

        # Initialize domain experts.
        self._experts = {}
        for d in self._domains:
            domain_id = d["id"]
            if domain_id == "CHESS":
                self._experts[domain_id] = ChessExpert(self.memory_bank)
            elif domain_id == "FINANCE":
                self._experts[domain_id] = FinanceExpert(self.memory_bank)
            elif domain_id == "LANGUAGE":
                self._experts[domain_id] = LanguageExpert(self.memory_bank)
            elif domain_id == "TECH":
                self._experts[domain_id] = TechExpert(self.memory_bank)
            elif domain_id == "GENERAL":
                self._experts[domain_id] = BaseExpert(self.memory_bank)
            else:
                self._experts[domain_id] = DynamicExpert(self.memory_bank, d)

        logger.info("CognitiveRouter loaded with %d domain experts.", len(self._experts))

    async def route(
        self,
        user_input: str,
        user_id: str,
        session_id: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        """
        Main routing method: classify intent → dispatch to expert → return response.

        Args:
            user_input: User's voice transcription or text message.
            user_id: Authenticated user identifier.
            session_id: Session ID for memory continuity (auto-generated if None).
            metadata: Optional metadata (voice confidence, source, etc.).

        Returns:
            Dict with domain, agent, response, session_id, and routing metadata.
        """
        start_ts = time.time()
        session_id = session_id or f"session-{uuid.uuid4().hex[:12]}"

        # Step 1: Fast keyword pre-classification (avoids LLM call for obvious cases).
        domain_id = self._keyword_classify(user_input)
        confidence = 0.85
        reason = "keyword match"

        # Step 2: If keyword classification is uncertain, use Gemini Flash.
        if domain_id is None:
            classification = await self._llm_classify(user_input, session_id, user_id)
            domain_id = classification.get("domain", "GENERAL")
            confidence = classification.get("confidence", 0.5)
            reason = classification.get("reason", "llm classification")

        expert = self._experts.get(domain_id, self._experts["GENERAL"])

        logger.info(
            "Routing decision",
            extra={
                "domain": domain_id,
                "confidence": confidence,
                "reason": reason,
                "user_id": user_id,
                "session_id": session_id,
            },
        )

        # Step 3: Dispatch to the domain expert.
        expert_response = await expert.process(
            user_input=user_input,
            session_id=session_id,
            user_id=user_id,
            context=metadata or {},
        )

        elapsed_ms = int((time.time() - start_ts) * 1000)

        # Step 4: Log routing decision to Pub/Sub for Portal observability.
        routing_event = {
            "type": "wisdom.router.decision_logged",
            "session_id": session_id,
            "user_id": user_id,
            "domain": domain_id,
            "confidence": confidence,
            "reason": reason,
            "elapsed_ms": elapsed_ms,
            "input_length": len(user_input),
        }
        self._publish_routing_log(routing_event)

        return {
            "domain": domain_id,
            "agent": expert.agent_name,
            "confidence": confidence,
            "reason": reason,
            "response": expert_response.get("response", ""),
            "session_id": session_id,
            "elapsed_ms": elapsed_ms,
            "memories_used": expert_response.get("memories_used", 0),
        }

    def _keyword_classify(self, text: str) -> str | None:
        """
        Fast O(n) keyword scan for high-confidence domains.
        Returns None if no keyword matches — triggers LLM fallback.
        """
        words = text.lower().split()
        domain_hits: dict[str, int] = {}

        for word in words:
            if word in self._keyword_index:
                d = self._keyword_index[word]
                domain_hits[d] = domain_hits.get(d, 0) + 1

        if not domain_hits:
            return None

        # Only use keyword match if there's a clear winner (2+ hits) to avoid false positives.
        best = max(domain_hits, key=domain_hits.get)
        if domain_hits[best] >= 2 or (len(domain_hits) == 1 and domain_hits[best] >= 1):
            return best

        return None

    async def _llm_classify(
        self, user_input: str, session_id: str, user_id: str
    ) -> dict[str, Any]:
        """Classify intent using Gemini Flash when keyword matching is insufficient."""
        runner = adk.Runner(
            agent=self._router_agent,
            app_name="wisdom-cognitive-router",
            session_service=adk.sessions.InMemorySessionService(),
        )

        response_text = ""
        async for event in runner.run_async(
            user_id=user_id,
            session_id=f"router-{session_id}",
            new_message=adk.types.Content(
                role="user",
                parts=[adk.types.Part(text=user_input)],
            ),
        ):
            if event.is_final_response() and event.content:
                for part in event.content.parts:
                    if part.text:
                        response_text += part.text

        # Parse JSON response from the router agent.
        try:
            # Extract JSON from response (it might have surrounding text).
            start = response_text.find("{")
            end = response_text.rfind("}") + 1
            if start >= 0 and end > start:
                return json.loads(response_text[start:end])
        except (json.JSONDecodeError, ValueError):
            logger.warning("Router LLM returned non-JSON: %s", response_text[:200])

        return {"domain": "GENERAL", "confidence": 0.4, "reason": "parse_failed"}

    def _publish_routing_log(self, event: dict[str, Any]) -> None:
        """Publish routing decision to Pub/Sub for Portal real-time observability."""
        try:
            data = json.dumps(event).encode("utf-8")
            self._pubsub.publish(self._routing_topic, data)
        except Exception as e:
            logger.warning("Failed to publish routing log: %s", e)
