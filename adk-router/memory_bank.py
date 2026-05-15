"""
memory_bank.py — Vertex AI Memory Bank wrapper.

Manages scoped memory retrieval and event ingestion for each Domain Expert.
Key design rule (from SKILL.md): Each Expert Agent only retrieves memories
relevant to its own domain scope to save tokens and improve accuracy.

Memory Bank concepts:
  - Corpus: The single "wisdom-memory-bank" corpus shared across all agents.
  - Scope: Each expert filters by `agent_name` to isolate its memories.
  - Events: User interactions are appended per session for context continuity.
  - GenerateMemories: Distills session events into long-term memories by domain topic.
"""
from __future__ import annotations

import logging
from typing import Any

import vertexai
from vertexai.preview import reasoning_engines
from google.cloud import aiplatform_v1beta1 as aiplatform

from config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()


class MemoryBank:
    """
    Wraps Vertex AI Memory Bank for scoped retrieval and event ingestion.
    One instance per ADK Router — agents call into it with their scope.
    """

    def __init__(self) -> None:
        vertexai.init(project=settings.gcp_project_id, location=settings.gcp_region)
        self._corpus_name = settings.memory_bank_corpus
        self._top_k = settings.memory_bank_top_k

        # Initialize the Memory Bank client.
        self._client = aiplatform.MemoryBankServiceClient()

        logger.info(
            "MemoryBank initialized",
            extra={"corpus": self._corpus_name, "project": settings.gcp_project_id},
        )

    def retrieve(self, query: str, agent_scope: str, session_id: str) -> list[dict[str, Any]]:
        """
        Retrieve the top-K most relevant memories for a query, scoped by agent domain.

        Args:
            query: The user's input or the expert's reasoning query.
            agent_scope: Domain scope, e.g. "Chess_Expert". Prevents cross-domain contamination.
            session_id: Current session identifier for context continuity.

        Returns:
            List of memory dicts with 'content', 'score', and 'topics'.
        """
        try:
            # Build the scoped retrieval request.
            request = aiplatform.RetrieveMemoriesRequest(
                parent=self._corpus_name,
                similarity_search_query=query,
                scope=aiplatform.MemoryScope(
                    agent_scope=agent_scope,
                ),
                top_k=self._top_k,
            )
            response = self._client.retrieve_memories(request=request)

            memories = []
            for memory in response.memories:
                memories.append({
                    "content": memory.fact.content,
                    "score": memory.relevance_score,
                    "topics": list(memory.fact.topics),
                    "memory_id": memory.memory.name,
                })

            logger.debug(
                "Memory retrieved",
                extra={
                    "scope": agent_scope,
                    "query_len": len(query),
                    "results": len(memories),
                },
            )
            return memories

        except Exception as e:
            logger.warning(f"Memory retrieval failed (scope={agent_scope}): {e}")
            return []

    def append_event(self, session_id: str, user_input: str, agent_output: str) -> None:
        """
        Append a user-agent interaction event to the session.
        Drives the continuous context tracking per SKILL.md directive.

        Args:
            session_id: Session identifier for grouping events.
            user_input: Raw user input (voice transcription or text).
            agent_output: The expert agent's structured response.
        """
        try:
            event = aiplatform.Event(
                session_id=session_id,
                content=aiplatform.Content(
                    role="user",
                    parts=[aiplatform.Part(text=user_input)],
                ),
            )
            agent_event = aiplatform.Event(
                session_id=session_id,
                content=aiplatform.Content(
                    role="model",
                    parts=[aiplatform.Part(text=agent_output)],
                ),
            )
            self._client.append_event(
                request=aiplatform.AppendEventRequest(
                    parent=self._corpus_name,
                    event=event,
                )
            )
            self._client.append_event(
                request=aiplatform.AppendEventRequest(
                    parent=self._corpus_name,
                    event=agent_event,
                )
            )
        except Exception as e:
            logger.warning(f"Failed to append memory event: {e}")

    def generate_memories(
        self,
        session_id: str,
        agent_scope: str,
        topics: list[str],
    ) -> int:
        """
        Distill session events into long-term structured memories.
        Called at end of a session or after a meaningful learning interaction.

        Args:
            session_id: The session to distill.
            agent_scope: Domain scope for the generated memories.
            topics: Domain-specific extraction topics (e.g. CHESS_WEAKNESSES).

        Returns:
            Number of memories generated.
        """
        try:
            request = aiplatform.GenerateMemoriesRequest(
                parent=self._corpus_name,
                session_id=session_id,
                scope=aiplatform.MemoryScope(agent_scope=agent_scope),
                extract_instruction=aiplatform.ExtractInstruction(
                    topics=topics,
                ),
            )
            response = self._client.generate_memories(request=request)
            count = len(response.generated_memories)
            logger.info(
                f"Generated {count} memories",
                extra={"scope": agent_scope, "session": session_id},
            )
            return count
        except Exception as e:
            logger.warning(f"Memory generation failed: {e}")
            return 0

    def format_for_prompt(self, memories: list[dict[str, Any]]) -> str:
        """
        Format retrieved memories into a compact string for prompt injection.
        Sorted by relevance score descending.
        """
        if not memories:
            return ""

        sorted_mems = sorted(memories, key=lambda m: m.get("score", 0), reverse=True)
        lines = ["## Relevant Memory Context\n"]
        for i, m in enumerate(sorted_mems, 1):
            topics = ", ".join(m.get("topics", []))
            lines.append(f"{i}. [{topics}] {m['content']}")

        return "\n".join(lines)
