"""
experts/language_expert.py — Language Learning Domain Expert ADK Agent.

Specializes in: grammar rules, vocabulary acquisition, pronunciation,
speaking practice, cultural context, and conjugation tables.

Memory topics: LANGUAGE_GOALS, VOCABULARY_GAPS, GRAMMAR_WEAKNESSES
Obsidian folder: Language/
Anki deck: Wisdom::Language
"""
from __future__ import annotations

from google import adk
from memory_bank import MemoryBank
from experts.base_expert import BaseExpert
from config import get_settings

settings = get_settings()

LANGUAGE_SYSTEM_INSTRUCTION = """
You are the Wisdom Language Expert — a patient, systematic language learning curator.

## Your Purpose
Transform language learning inputs (words, phrases, grammar rules, pronunciation tips)
into optimized Anki cards and structured Obsidian grammar reference notes.

## Core Behaviors
1. **Always memorize first**: Store vocabulary/grammar to Cortex before creating cards.
2. **Cloze for conjugation**: Use cloze cards for verb conjugation tables.
   Example: "Yo {{c1::tengo}} (tener, presente)"
3. **Basic for vocabulary**: Use Basic cards (Front: Target language, Back: Native language).
4. **Grammar notes**: Create Obsidian notes for grammar rules with examples and exceptions.
5. **Context sentences**: Always include a context example sentence for vocabulary cards.
6. **Etymology when useful**: Add word origins to aid memorization.

## Card Naming Conventions
- Vocabulary: Front = word in target language, Back = meaning + example sentence + pronunciation
- Grammar: Use cloze for fill-in-the-blank patterns
- Phrases: Front = situation, Back = phrase + alternatives

## Obsidian Note Structure for Grammar Rules
```
## Rule
[Clear statement of the rule]

## When to Use
...

## Examples
1. [Example 1]
2. [Example 2]

## Exceptions
- [Exception 1]

## Common Mistakes
- [Mistake and correction]
```

## Deck Structure
- `Wisdom::Language::[TargetLanguage]::Vocabulary` — Word-level cards
- `Wisdom::Language::[TargetLanguage]::Grammar` — Grammar pattern cards
- `Wisdom::Language::[TargetLanguage]::Phrases` — Conversational phrases
"""


class LanguageExpert(BaseExpert):
    domain_id = "LANGUAGE"
    agent_name = "Language_Expert"
    memory_scope = "Language_Expert"
    memory_topics = ["LANGUAGE_GOALS", "VOCABULARY_GAPS", "GRAMMAR_WEAKNESSES"]
    anki_deck_prefix = "Wisdom::Language"
    obsidian_folder = "Language/"
    system_instruction = LANGUAGE_SYSTEM_INSTRUCTION

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
                self._tool_create_vocabulary_card,
                self._tool_create_grammar_note,
            ],
        )

    def _tool_create_vocabulary_card(
        self,
        word: str,
        target_language: str,
        native_translation: str,
        example_sentence: str,
        pronunciation: str = "",
        etymology: str = "",
        deck_override: str = "",
        user_id: str = "",
    ) -> dict:
        """
        Create an optimized vocabulary Anki card with example sentence and pronunciation.

        Args:
            word: The word in the target language.
            target_language: Language code, e.g. 'es', 'fr', 'ja'.
            native_translation: Translation in the user's native language.
            example_sentence: A natural usage example in the target language.
            pronunciation: IPA or phonetic hint.
            etymology: Word origin if useful for memorization.
            deck_override: Custom deck name (default: Wisdom::Language::[lang]::Vocabulary).
            user_id: User ID.
        """
        deck = deck_override or f"Wisdom::Language::{target_language.upper()}::Vocabulary"

        pronunciation_line = f"\n**/{pronunciation}/**" if pronunciation else ""
        etymology_line = f"\n*From: {etymology}*" if etymology else ""

        back = (
            f"**{native_translation}**{pronunciation_line}\n\n"
            f"*Example:* {example_sentence}{etymology_line}"
        )

        node_id = self._cortex.memorize(
            node_type="Vocabulary",
            payload={
                "word": word,
                "language": target_language,
                "translation": native_translation,
                "example": example_sentence,
                "domain": "LANGUAGE",
            },
        )

        card_result = self._tool_create_flashcard(
            front=word,
            back=back,
            deck_name=deck,
            tags=[f"Wisdom::Language::{target_language.upper()}", "vocabulary"],
            wisdom_node_id=node_id,
            user_id=user_id,
        )

        return {"node_id": node_id, "word": word, "card": card_result}

    def _tool_create_grammar_note(
        self,
        rule_title: str,
        target_language: str,
        rule_explanation: str,
        examples: list[str],
        exceptions: list[str] | None = None,
        common_mistakes: list[str] | None = None,
        user_id: str = "",
    ) -> dict:
        """
        Create a structured grammar reference note in Obsidian and corresponding Anki cards.

        Args:
            rule_title: e.g. 'Spanish Subjunctive — Present Tense'.
            target_language: Language code.
            rule_explanation: Clear statement of the grammar rule.
            examples: List of example sentences illustrating the rule.
            exceptions: Optional list of exceptions.
            common_mistakes: Optional list of common errors to avoid.
            user_id: User ID.
        """
        examples_md = "\n".join(f"{i+1}. {ex}" for i, ex in enumerate(examples))
        exceptions_md = (
            "\n".join(f"- {e}" for e in exceptions) if exceptions else "None."
        )
        mistakes_md = (
            "\n".join(f"- {m}" for m in common_mistakes) if common_mistakes else "None noted."
        )

        content = (
            f"## Rule\n{rule_explanation}\n\n"
            f"## When to Use\nSee rule explanation above.\n\n"
            f"## Examples\n{examples_md}\n\n"
            f"## Exceptions\n{exceptions_md}\n\n"
            f"## Common Mistakes\n{mistakes_md}\n"
        )

        node_id = self._cortex.memorize(
            node_type="GrammarRule",
            payload={
                "title": rule_title,
                "language": target_language,
                "rule": rule_explanation,
                "domain": "LANGUAGE",
            },
        )

        note_result = self._tool_create_note(
            title=rule_title,
            content=content,
            path=f"Language/{target_language.upper()}/Grammar/{rule_title.replace(' ', '-')}.md",
            tags=[f"#language/{target_language}", "#grammar"],
            mastery_score=0.3,
            wisdom_node_id=node_id,
            user_id=user_id,
        )

        # Create a cloze card for the most important example.
        if examples:
            cloze_result = self._tool_create_cloze_card(
                cloze_text=examples[0],
                extra=f"Rule: {rule_explanation}",
                deck_name=f"Wisdom::Language::{target_language.upper()}::Grammar",
                wisdom_node_id=node_id,
                user_id=user_id,
            )
        else:
            cloze_result = {}

        return {
            "node_id": node_id,
            "rule": rule_title,
            "note": note_result,
            "cloze_card": cloze_result,
        }
