"""
experts/chess_expert.py — Chess Domain Expert ADK Agent.

Specializes in: openings, tactics, endgames, game analysis, Lichess data,
and personal repertoire management.

Memory topics: CHESS_WEAKNESSES, CHESS_OPENINGS, CHESS_TACTICS
Obsidian folder: Chess/
Anki deck: Wisdom::Chess
"""
from __future__ import annotations

from google import adk
from memory_bank import MemoryBank
from experts.base_expert import BaseExpert
from config import get_settings

settings = get_settings()

CHESS_SYSTEM_INSTRUCTION = """
You are the Wisdom Chess Expert — a deep, analytical chess coach and knowledge curator.

## Your Purpose
Transform chess learning inputs (positions, openings, games, tactics) into structured,
permanent knowledge entries in Obsidian and Anki. You help the user build a personal
chess repertoire backed by spaced repetition.

## Core Behaviors
1. **Always memorize first**: Before creating any note or card, call `memorize_concept`
   to store the concept in Cortex. Use the returned `node_id` for cross-references.
2. **Notes for concepts**: Create Obsidian notes for openings, strategic ideas, and endgame
   techniques. Use [[Wikilinks]] to connect related concepts.
3. **Cards for patterns**: Create Anki cards for specific tactics, move sequences,
   and positions that require recall under pressure.
4. **Check weaknesses**: Call `get_weaknesses` to prioritize what to study if asked.
5. **Search first**: Call `search_knowledge` before creating content to avoid duplicates.

## Output Format for Notes
Use this structure for opening notes:
```
## Overview
Brief description of the opening and its strategic ideas.

## Main Line
1. e4 e5 2. Nf3 Nc6 ...

## Key Ideas
- [ ] Idea 1
- [ ] Idea 2

## Related
- [[Parent Opening]]
- [[Key Variation]]
```

## Card Format
- Front: The position question or "What is the best move after...?"
- Back: The answer + reason + ECO code (if applicable)

## Deck Structure
- `Wisdom::Chess::Openings` — Opening lines and transpositions
- `Wisdom::Chess::Tactics` — Tactical patterns (fork, pin, skewer, etc.)
- `Wisdom::Chess::Endgames` — Endgame techniques
- `Wisdom::Chess::Strategy` — Strategic concepts (pawn structures, piece activity)
"""


class ChessExpert(BaseExpert):
    domain_id = "CHESS"
    agent_name = "Chess_Expert"
    memory_scope = "Chess_Expert"
    memory_topics = ["CHESS_WEAKNESSES", "CHESS_OPENINGS", "CHESS_TACTICS"]
    anki_deck_prefix = "Wisdom::Chess"
    obsidian_folder = "Chess/"
    system_instruction = CHESS_SYSTEM_INSTRUCTION

    def __init__(self, memory_bank: MemoryBank) -> None:
        super().__init__(memory_bank)
        # Add chess-specific tools on top of the base tools.
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
                self._tool_analyze_position,
                self._tool_add_to_repertoire,
            ],
        )

    def _tool_analyze_position(
        self,
        fen: str,
        context: str = "",
    ) -> dict:
        """
        Analyze a chess position given in FEN notation.
        Returns strategic assessment and suggested study plan.

        Args:
            fen: FEN string of the position to analyze.
            context: Optional context (e.g. 'after 1.e4 e5 2.Nf3').
        """
        # The ADK agent (Gemini) will analyze the position via reasoning.
        # In production, this could call Stockfish or Lichess API via Researcher.
        return {
            "fen": fen,
            "context": context,
            "instruction": (
                "Analyze this FEN position strategically. "
                "Identify the key ideas, weaknesses, and best plans for both sides. "
                "Then create an Obsidian note with your analysis using create_note."
            ),
        }

    def _tool_add_to_repertoire(
        self,
        opening_name: str,
        eco_code: str,
        color: str,
        main_line: str,
        key_ideas: list[str],
        user_id: str = "",
    ) -> dict:
        """
        Add an opening to the user's personal repertoire.
        Creates both an Obsidian note and Anki cards for the key moves.

        Args:
            opening_name: e.g. 'Caro-Kann Defense — Main Line'.
            eco_code: e.g. 'B15'.
            color: 'white' or 'black'.
            main_line: PGN or move list, e.g. '1.e4 c6 2.d4 d5'.
            key_ideas: List of strategic ideas to remember.
            user_id: User ID.
        """
        # Store in Cortex.
        node_id = self._cortex.memorize(
            node_type="Opening",
            payload={
                "name": opening_name,
                "eco": eco_code,
                "color": color,
                "main_line": main_line,
                "key_ideas": key_ideas,
                "domain": "CHESS",
            },
        )

        # Build note content.
        ideas_md = "\n".join(f"- {idea}" for idea in key_ideas)
        content = (
            f"## Overview\n{opening_name} ({eco_code}) — {color} repertoire.\n\n"
            f"## Main Line\n{main_line}\n\n"
            f"## Key Ideas\n{ideas_md}\n\n"
            f"## Related\n- [[Chess/Openings/Index]]\n"
            f"<!-- wisdom_node_id: {node_id} -->"
        )

        note_result = self._tool_create_note(
            title=opening_name,
            content=content,
            path=f"Chess/Openings/{opening_name.replace(' ', '-')}.md",
            tags=["#chess/openings", f"#chess/eco/{eco_code.lower()}"],
            relationships=["[[Chess/Openings/Index]]"],
            mastery_score=0.3,  # New openings start at FRAGILE.
            wisdom_node_id=node_id,
            user_id=user_id,
        )

        # Create a summary Anki card.
        card_result = self._tool_create_flashcard(
            front=f"What are the key ideas in the {opening_name}?",
            back=ideas_md + f"\n\nMain line: {main_line}",
            deck_name="Wisdom::Chess::Openings",
            tags=["Wisdom::Chess", f"eco::{eco_code}"],
            wisdom_node_id=node_id,
            user_id=user_id,
        )

        return {
            "node_id": node_id,
            "note": note_result,
            "card": card_result,
            "opening": opening_name,
        }
