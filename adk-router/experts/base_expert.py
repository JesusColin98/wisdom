"""
experts/base_expert.py — Base class for all Domain Expert Agents.

Each expert is an ADK Agent with:
  1. Scoped memory retrieval (no cross-domain contamination)
  2. A standard set of tools (memorize, create_note, create_card)
  3. Domain-specific system instructions and memory topics
"""
from __future__ import annotations

import logging
from typing import Any

from google import adk

from memory_bank import MemoryBank
from grpc_clients import (
    get_cortex_client,
    get_integrations_client,
    get_mastery_client,
)
from config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()


class BaseExpert:
    """
    Base class for all Domain Expert ADK Agents.
    Subclasses define: domain_id, agent_name, memory_topics, system_instruction,
    anki_deck_prefix, and obsidian_folder.
    """

    domain_id: str = "GENERAL"
    agent_name: str = "General_Expert"
    memory_scope: str = "General_Expert"
    memory_topics: list[str] = ["GENERAL_INSIGHTS"]
    anki_deck_prefix: str = "Wisdom::General"
    obsidian_folder: str = "General/"
    system_instruction: str = "You are a general knowledge expert."

    def __init__(self, memory_bank: MemoryBank) -> None:
        self.memory_bank = memory_bank
        self._cortex = get_cortex_client()
        self._integrations = get_integrations_client()
        self._mastery = get_mastery_client()

        # Build the ADK agent with domain-specific tools.
        self._agent = adk.Agent(
            name=self.agent_name,
            model=settings.expert_model,
            instruction=self.system_instruction,
            tools=[
                self._tool_memorize_concept,
                self._tool_create_note,
                self._tool_create_flashcard,
                self._tool_create_cloze_card,
                self._tool_get_weaknesses,
                self._tool_search_knowledge,
            ],
        )

    # ─── ADK Tools (Standard, available to all experts) ──────────────────────

    def _tool_memorize_concept(
        self,
        title: str,
        content: str,
        concept_type: str = "Fact",
        confidence: float = 0.8,
    ) -> dict:
        """
        Store a new concept in the Wisdom Cortex knowledge graph.
        Use this FIRST before creating Obsidian notes or Anki cards.

        Args:
            title: The concept name or title.
            content: Full Markdown content of the concept.
            concept_type: 'Fact', 'Theory', 'Procedure', or 'Example'.
            confidence: How certain the expert is (0.0 to 1.0).

        Returns:
            Dict with 'node_id' — use this for backlinks in notes and cards.
        """
        node_id = self._cortex.memorize(
            node_type=concept_type,
            payload={
                "title": title,
                "content": content,
                "domain": self.domain_id,
                "agent": self.agent_name,
            },
            confidence=confidence,
        )
        logger.info(f"{self.agent_name}: memorized concept '{title}' (id={node_id})")
        return {"node_id": node_id, "title": title, "status": "memorized"}

    def _tool_create_note(
        self,
        title: str,
        content: str,
        path: str | None = None,
        tags: list[str] | None = None,
        relationships: list[str] | None = None,
        mastery_score: float = 0.5,
        wisdom_node_id: str = "",
        user_id: str = "",
    ) -> dict:
        """
        Create a structured Markdown note in Obsidian via the Integrations bridge.
        Always call memorize_concept FIRST to get the wisdom_node_id.

        Args:
            title: Note title (used in YAML frontmatter).
            content: Markdown body of the note.
            path: Vault-relative path. Auto-generated if not provided.
            tags: YAML frontmatter tags (domain prefix is auto-added).
            relationships: Wikilinks to related concepts.
            mastery_score: Initial mastery score 0.0–1.0.
            wisdom_node_id: Cortex node ID for back-reference.
            user_id: User ID for the Integrations service.

        Returns:
            Dict with 'success' and 'status'.
        """
        target_path = path or f"{self.obsidian_folder}{title.replace(' ', '-')}.md"
        domain_tag = f"#{self.domain_id.lower()}"
        all_tags = list(set([domain_tag] + (tags or [])))

        result = self._integrations.create_note(
            agent_name=self.agent_name,
            user_id=user_id or settings.default_user_id,
            path=target_path,
            title=title,
            content=content,
            tags=all_tags,
            relationships=relationships or [],
            mastery_score=mastery_score,
        )
        logger.info(f"{self.agent_name}: create_note '{title}' → {result.get('status')}")
        return result

    def _tool_create_flashcard(
        self,
        front: str,
        back: str,
        deck_name: str | None = None,
        tags: list[str] | None = None,
        wisdom_node_id: str = "",
        user_id: str = "",
    ) -> dict:
        """
        Create a Basic flashcard (Front/Back) in Anki via the Integrations bridge.

        Args:
            front: Question side of the card. Supports Markdown.
            back: Answer side. Include the source Obsidian link if available.
            deck_name: Target deck. Defaults to domain prefix (e.g. 'Wisdom::Chess').
            tags: Additional tags beyond the domain tag.
            wisdom_node_id: Cortex node ID for mastery sync.
            user_id: User ID.

        Returns:
            Dict with 'success' and 'status'.
        """
        target_deck = deck_name or self.anki_deck_prefix
        all_tags = [f"Wisdom::{self.domain_id}"] + (tags or [])

        return self._integrations.create_card(
            agent_name=self.agent_name,
            user_id=user_id or settings.default_user_id,
            deck_name=target_deck,
            front=front,
            back=back,
            card_type="BASIC",
            tags=all_tags,
            wisdom_node_id=wisdom_node_id,
        )

    def _tool_create_cloze_card(
        self,
        cloze_text: str,
        extra: str = "",
        deck_name: str | None = None,
        tags: list[str] | None = None,
        wisdom_node_id: str = "",
        user_id: str = "",
    ) -> dict:
        """
        Create a Cloze deletion card in Anki.
        Format: 'The {{c1::Sicilian Defense}} is the most popular chess opening.'

        Args:
            cloze_text: Text with {{c1::...}} cloze markers.
            extra: Additional context shown on the back.
            deck_name: Target deck.
            tags: Additional tags.
            wisdom_node_id: Cortex node ID for mastery sync.
            user_id: User ID.
        """
        target_deck = deck_name or self.anki_deck_prefix
        all_tags = [f"Wisdom::{self.domain_id}"] + (tags or [])

        return self._integrations.create_card(
            agent_name=self.agent_name,
            user_id=user_id or settings.default_user_id,
            deck_name=target_deck,
            front="",
            back="",
            card_type="CLOZE",
            cloze_text=cloze_text,
            tags=all_tags,
            wisdom_node_id=wisdom_node_id,
        )

    def _tool_get_weaknesses(self, user_id: str = "", limit: int = 5) -> dict:
        """
        Get the user's weakest concepts in this domain from the Mastery service.
        Use to suggest what to study next or to prioritize card creation.

        Returns:
            Dict with 'concepts' list and 'domain'.
        """
        all_weaknesses = self._mastery.get_weaknesses(
            user_id=user_id or settings.default_user_id,
            limit=limit * 3,  # Over-fetch and filter by domain.
        )
        domain_weaknesses = [
            w for w in all_weaknesses
            if self.domain_id.lower() in str(w.get("tags", [])).lower()
        ][:limit]

        return {"domain": self.domain_id, "concepts": domain_weaknesses}

    def _tool_search_knowledge(self, query: str) -> dict:
        """
        Search existing knowledge in the Cortex graph for this domain.
        Use before creating new content to avoid duplicates.

        Args:
            query: Search phrase, e.g. 'Caro-Kann defense moves'.

        Returns:
            Dict with 'results' list.
        """
        results = self._cortex.query_facts(
            query=query,
            filters={"domain": self.domain_id},
        )
        return {"results": results[:10], "domain": self.domain_id}

    # ─── Main Process Method ──────────────────────────────────────────────────

    async def process(
        self,
        user_input: str,
        session_id: str,
        user_id: str,
        context: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        """
        Main entry point for processing a user request via this expert.

        1. Retrieve scoped memories for the query.
        2. Build prompt with memory context.
        3. Run ADK agent.
        4. Append session event to Memory Bank.
        5. Return structured response.
        """
        # Step 1: Retrieve scoped memories.
        memories = self.memory_bank.retrieve(
            query=user_input,
            agent_scope=self.memory_scope,
            session_id=session_id,
        )
        memory_context = self.memory_bank.format_for_prompt(memories)

        # Step 2: Build enriched prompt.
        enriched_prompt = _build_expert_prompt(
            user_input=user_input,
            memory_context=memory_context,
            user_id=user_id,
            context=context or {},
        )

        # Step 3: Run ADK agent (tools are called automatically by the ADK).
        runner = adk.Runner(
            agent=self._agent,
            app_name=f"wisdom-{self.domain_id.lower()}-expert",
            session_service=adk.sessions.InMemorySessionService(),
        )
        response_text = ""
        async for event in runner.run_async(
            user_id=user_id,
            session_id=session_id,
            new_message=adk.types.Content(
                role="user",
                parts=[adk.types.Part(text=enriched_prompt)],
            ),
        ):
            if event.is_final_response() and event.content:
                for part in event.content.parts:
                    if part.text:
                        response_text += part.text

        # Step 4: Append session event to Memory Bank.
        self.memory_bank.append_event(
            session_id=session_id,
            user_input=user_input,
            agent_output=response_text,
        )

        return {
            "domain": self.domain_id,
            "agent": self.agent_name,
            "response": response_text,
            "memories_used": len(memories),
            "session_id": session_id,
        }


def _build_expert_prompt(
    user_input: str,
    memory_context: str,
    user_id: str,
    context: dict[str, Any],
) -> str:
    """Build a structured prompt for the expert agent."""
    parts = [f"User: {user_input}"]

    if memory_context:
        parts.append(f"\n{memory_context}")

    if context.get("weaknesses"):
        parts.append(
            "\n## Current Weaknesses\n"
            + "\n".join(f"- {w}" for w in context["weaknesses"])
        )

    parts.append(f"\n[user_id: {user_id}]")
    return "\n".join(parts)
