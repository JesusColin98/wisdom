"""
experts/dynamic_expert.py — Dynamic Expert Agent generated from configuration.
"""
from __future__ import annotations

import logging
from typing import Any

from experts.base_expert import BaseExpert
from memory_bank import MemoryBank

logger = logging.getLogger(__name__)


class DynamicExpert(BaseExpert):
    """
    A Domain Expert instantiated dynamically from a domain configuration dictionary,
    rather than a hardcoded python class.
    """

    def __init__(self, memory_bank: MemoryBank, config: dict[str, Any]) -> None:
        """
        Initialize a DynamicExpert with configuration properties.
        
        Expected config fields:
        - id: Domain ID (e.g. 'PHILOSOPHY')
        - agent: Agent Name (e.g. 'PhilosophyExpert')
        - description: Description of the domain
        - system_instruction: The custom system prompt for the Gemini model
        - keywords: list of keywords
        - memory_scope: The memory collection scope
        - anki_deck_prefix: e.g. 'Wisdom::Philosophy'
        - obsidian_folder: e.g. 'Philosophy/'
        - memory_topics: list of default memory topics
        """
        self.domain_id = config.get("id", "UNKNOWN")
        self.agent_name = config.get("agent", f"{self.domain_id.capitalize()}Expert")
        
        # Determine instruction from config, or use a default one based on description
        desc = config.get("description", f"You are an expert in {self.domain_id}.")
        self.system_instruction = config.get("system_instruction", f"You are a Wisdom Domain Expert. Your domain is {self.domain_id}. {desc}")
        
        self.memory_scope = config.get("memory_scope", f"{self.domain_id.capitalize()}_Expert")
        self.memory_topics = config.get("memory_topics", [f"{self.domain_id}_INSIGHTS"])
        self.anki_deck_prefix = config.get("anki_deck_prefix", f"Wisdom::{self.domain_id.capitalize()}")
        self.obsidian_folder = config.get("obsidian_folder", f"{self.domain_id.capitalize()}/")
        
        # Super init handles ADK agent initialization with these dynamic attributes.
        super().__init__(memory_bank)
        logger.info(f"Instantiated DynamicExpert for domain '{self.domain_id}'")
