"""
experts/tech_expert.py — Software Engineering Domain Expert ADK Agent.

Specializes in: algorithms, system design, Go, Python, TypeScript, databases,
cloud architecture, design patterns, and code review.

Memory topics: TECH_STACK, CODE_PATTERNS, SYSTEM_DESIGN_DECISIONS
Obsidian folder: Tech/
Anki deck: Wisdom::Tech
"""
from __future__ import annotations

from google import adk
from memory_bank import MemoryBank
from experts.base_expert import BaseExpert
from config import get_settings

settings = get_settings()

TECH_SYSTEM_INSTRUCTION = """
You are the Wisdom Tech Expert — a senior software engineer and knowledge architect.

## Your Purpose
Transform software engineering knowledge (algorithms, design patterns, architecture
decisions, code snippets, system design) into permanent Obsidian notes and Anki cards
optimized for engineering interviews and daily practice.

## Core Behaviors
1. **Always memorize first**: Store concepts to Cortex before creating notes or cards.
2. **Code in notes**: Obsidian notes should include working code examples with language tags.
3. **Pattern for cards**: Use Basic cards for "What is X?", Cloze for code completion patterns.
4. **Big-O notation**: Always include time/space complexity for algorithm cards.
5. **ADR pattern**: For architecture decisions, use the ADR (Architecture Decision Record) format.
6. **Search first**: Check for existing notes before duplicating content.

## Obsidian Note Structure for Algorithms
```
## Complexity
- Time: O(n log n)
- Space: O(n)

## When to Use
...

## Implementation
\`\`\`go
func example() {}
\`\`\`

## Common Pitfalls
- ...

## Related
- [[Binary Search]]
```

## Card Format
- Algorithm card: Front: "What is the time complexity of Merge Sort?", Back: "O(n log n) time, O(n) space"
- Code cloze: "The time complexity of binary search is {{c1::O(log n)}}"
- Pattern definition: Front: "What problem does the Observer pattern solve?", Back: "..."

## Deck Structure
- `Wisdom::Tech::Algorithms` — Algorithm complexity and implementation
- `Wisdom::Tech::SystemDesign` — Architecture patterns and trade-offs
- `Wisdom::Tech::Languages` — Language-specific syntax and patterns
- `Wisdom::Tech::Databases` — SQL, NoSQL, query optimization
"""


class TechExpert(BaseExpert):
    domain_id = "TECH"
    agent_name = "Tech_Expert"
    memory_scope = "Tech_Expert"
    memory_topics = ["TECH_STACK", "CODE_PATTERNS", "SYSTEM_DESIGN_DECISIONS"]
    anki_deck_prefix = "Wisdom::Tech"
    obsidian_folder = "Tech/"
    system_instruction = TECH_SYSTEM_INSTRUCTION

    def __init__(self, memory_bank: MemoryBank) -> None:
        super().__init__(memory_bank)
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
                self._tool_create_algorithm_note,
                self._tool_create_adr,
            ],
        )

    def _tool_create_algorithm_note(
        self,
        algorithm_name: str,
        time_complexity: str,
        space_complexity: str,
        description: str,
        when_to_use: str,
        code_example: str,
        language: str = "go",
        pitfalls: list[str] | None = None,
        related: list[str] | None = None,
        user_id: str = "",
    ) -> dict:
        """
        Create a structured algorithm reference note in Obsidian and corresponding Anki cards.

        Args:
            algorithm_name: e.g. 'Binary Search'.
            time_complexity: e.g. 'O(log n)'.
            space_complexity: e.g. 'O(1)'.
            description: What the algorithm does and its key insight.
            when_to_use: Conditions under which this is the right choice.
            code_example: Implementation in the specified language.
            language: Programming language for the code block.
            pitfalls: Common implementation mistakes.
            related: Related algorithm names for Wikilinks.
            user_id: User ID.
        """
        pitfalls_md = "\n".join(f"- {p}" for p in (pitfalls or []))
        related_md = "\n".join(f"- [[Tech/Algorithms/{r}]]" for r in (related or []))

        content = (
            f"## Complexity\n"
            f"- Time: `{time_complexity}`\n"
            f"- Space: `{space_complexity}`\n\n"
            f"## What It Does\n{description}\n\n"
            f"## When to Use\n{when_to_use}\n\n"
            f"## Implementation\n```{language}\n{code_example}\n```\n\n"
            f"## Common Pitfalls\n{pitfalls_md or 'None noted.'}\n\n"
            f"## Related\n{related_md or '—'}\n"
        )

        node_id = self._cortex.memorize(
            node_type="Algorithm",
            payload={
                "name": algorithm_name,
                "time_complexity": time_complexity,
                "space_complexity": space_complexity,
                "language": language,
                "domain": "TECH",
            },
        )

        note_result = self._tool_create_note(
            title=algorithm_name,
            content=content,
            path=f"Tech/Algorithms/{algorithm_name.replace(' ', '-')}.md",
            tags=["#tech/algorithm", f"#tech/complexity/{time_complexity.replace('(','').replace(')','').replace(' ','').lower()}"],
            mastery_score=0.4,
            wisdom_node_id=node_id,
            user_id=user_id,
        )

        # Complexity card.
        card = self._tool_create_flashcard(
            front=f"What is the time and space complexity of {algorithm_name}?",
            back=f"⏱ Time: `{time_complexity}`\n💾 Space: `{space_complexity}`\n\n{description}",
            deck_name="Wisdom::Tech::Algorithms",
            wisdom_node_id=node_id,
            user_id=user_id,
        )

        return {"node_id": node_id, "algorithm": algorithm_name, "note": note_result, "card": card}

    def _tool_create_adr(
        self,
        title: str,
        status: str,
        context: str,
        decision: str,
        consequences: str,
        alternatives: list[str] | None = None,
        user_id: str = "",
    ) -> dict:
        """
        Create an Architecture Decision Record (ADR) note in Obsidian.

        Args:
            title: Decision title, e.g. 'Use GCP Pub/Sub over NATS'.
            status: 'PROPOSED', 'ACCEPTED', 'DEPRECATED', 'SUPERSEDED'.
            context: Why this decision was needed.
            decision: The decision made and its rationale.
            consequences: Positive and negative outcomes.
            alternatives: Other options that were considered.
            user_id: User ID.
        """
        alts_md = "\n".join(f"- {a}" for a in (alternatives or []))
        content = (
            f"## Status\n**{status}**\n\n"
            f"## Context\n{context}\n\n"
            f"## Decision\n{decision}\n\n"
            f"## Consequences\n{consequences}\n\n"
            f"## Alternatives Considered\n{alts_md or 'None documented.'}\n"
        )

        node_id = self._cortex.memorize(
            node_type="ADR",
            payload={
                "title": title,
                "status": status,
                "decision": decision,
                "domain": "TECH",
            },
        )

        note_result = self._tool_create_note(
            title=f"ADR — {title}",
            content=content,
            path=f"Tech/ADRs/{title.replace(' ', '-')}.md",
            tags=["#tech/adr", f"#tech/adr/status/{status.lower()}"],
            mastery_score=0.8,  # ADRs are reference material, not SRS content.
            wisdom_node_id=node_id,
            user_id=user_id,
        )

        return {"node_id": node_id, "title": title, "note": note_result}
