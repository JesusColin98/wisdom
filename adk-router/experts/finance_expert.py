"""
experts/finance_expert.py — Finance Domain Expert ADK Agent.

Specializes in: financial markets, stocks, ETFs, crypto, Fibras (MX REITs),
macroeconomics, portfolio management, and financial thesis tracking.

Memory topics: FINANCIAL_GOALS, PORTFOLIO_POSITIONS, MARKET_THESIS
Obsidian folder: Finance/
Anki deck: Wisdom::Finance
"""
from __future__ import annotations

import json
from google import adk
from memory_bank import MemoryBank
from experts.base_expert import BaseExpert
from config import get_settings

settings = get_settings()

FINANCE_SYSTEM_INSTRUCTION = """
You are the Wisdom Finance Expert — a rigorous financial analyst and learning curator.

## Your Purpose
Transform financial knowledge inputs (market analysis, company theses, macro concepts,
Fibras analysis) into structured, actionable notes in Obsidian and Anki flashcards for
long-term retention.

## Core Behaviors
1. **Always memorize first**: Store new concepts in Cortex before creating notes or cards.
2. **Thesis tracking**: For stocks or Fibras, always create a thesis note with entry rationale,
   key metrics, and risks.
3. **Concept cards**: Financial formulas, ratios (P/E, DCF, cap rate), and definitions
   are ideal Anki cloze cards.
4. **Risk-aware**: Always note the key risks for any investment thesis.
5. **Search first**: Check existing knowledge before creating duplicates.

## Output Format for Stock/Fibra Thesis Notes
```
## Thesis
One-sentence investment thesis.

## Key Metrics
- P/E: X.X
- Dividend yield: X.X%
- Market cap: $Xb

## Bull Case
...

## Bear Case / Risks
...

## Action Triggers
- Buy more if: ...
- Sell if: ...
```

## Card Format
- Financial ratios: Use cloze: "P/E ratio = {{c1::Price / EPS}}"
- Definitions: Front: "What is DCF?", Back: "Discounted Cash Flow — method to value..."
- Formulas: Use cloze for formula components

## Deck Structure
- `Wisdom::Finance::Concepts` — Financial theory and formulas
- `Wisdom::Finance::Fibras` — MX REIT analysis (Fibra Uno, Macquarie, etc.)
- `Wisdom::Finance::Stocks` — Stock theses
- `Wisdom::Finance::Macro` — Macroeconomic concepts
"""


class FinanceExpert(BaseExpert):
    domain_id = "FINANCE"
    agent_name = "Finance_Expert"
    memory_scope = "Finance_Expert"
    memory_topics = ["FINANCIAL_GOALS", "PORTFOLIO_POSITIONS", "MARKET_THESIS"]
    anki_deck_prefix = "Wisdom::Finance"
    obsidian_folder = "Finance/"
    system_instruction = FINANCE_SYSTEM_INSTRUCTION

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
                self._tool_create_investment_thesis,
                self._tool_track_portfolio_position,
            ],
        )

    def _tool_create_investment_thesis(
        self,
        ticker: str,
        asset_type: str,
        thesis: str,
        key_metrics: dict,
        bull_case: str,
        bear_case: str,
        action_triggers: dict,
        user_id: str = "",
    ) -> dict:
        """
        Create a structured investment thesis note for a stock, ETF, or Fibra.

        Args:
            ticker: Ticker symbol, e.g. 'FUNO11' or 'VTI'.
            asset_type: 'STOCK', 'ETF', 'FIBRA', 'CRYPTO', 'BOND'.
            thesis: One-sentence investment thesis.
            key_metrics: Dict of metric name → value, e.g. {'P/E': 12.3, 'yield': '8.5%'}.
            bull_case: Detailed bull case reasoning.
            bear_case: Key risks and bear case.
            action_triggers: Dict with 'buy' and 'sell' conditions.
            user_id: User ID.
        """
        metrics_md = "\n".join(f"- {k}: {v}" for k, v in key_metrics.items())
        triggers_md = (
            f"- Buy more if: {action_triggers.get('buy', 'TBD')}\n"
            f"- Sell if: {action_triggers.get('sell', 'TBD')}"
        )

        content = (
            f"## Thesis\n{thesis}\n\n"
            f"## Asset Info\n- Ticker: {ticker}\n- Type: {asset_type}\n\n"
            f"## Key Metrics\n{metrics_md}\n\n"
            f"## Bull Case\n{bull_case}\n\n"
            f"## Bear Case / Risks\n{bear_case}\n\n"
            f"## Action Triggers\n{triggers_md}\n"
        )

        node_id = self._cortex.memorize(
            node_type="InvestmentThesis",
            payload={
                "ticker": ticker,
                "asset_type": asset_type,
                "thesis": thesis,
                "metrics": key_metrics,
                "domain": "FINANCE",
            },
        )

        note_result = self._tool_create_note(
            title=f"{ticker} — Investment Thesis",
            content=content,
            path=f"Finance/{asset_type.title()}/{ticker}.md",
            tags=[f"#finance/{asset_type.lower()}", f"#finance/ticker/{ticker.lower()}"],
            mastery_score=0.5,
            wisdom_node_id=node_id,
            user_id=user_id,
        )

        return {"node_id": node_id, "ticker": ticker, "note": note_result}

    def _tool_track_portfolio_position(
        self,
        ticker: str,
        shares: float,
        avg_cost: float,
        currency: str = "USD",
        user_id: str = "",
    ) -> dict:
        """
        Store a portfolio position in Cortex for tracking.
        Used by the Memory Bank to maintain PORTFOLIO_POSITIONS context.

        Args:
            ticker: Ticker symbol.
            shares: Number of shares/units held.
            avg_cost: Average cost basis per share.
            currency: Position currency ('USD', 'MXN').
            user_id: User ID.
        """
        node_id = self._cortex.memorize(
            node_type="PortfolioPosition",
            payload={
                "ticker": ticker,
                "shares": shares,
                "avg_cost": avg_cost,
                "currency": currency,
                "user_id": user_id or settings.default_user_id,
                "domain": "FINANCE",
            },
        )
        return {
            "node_id": node_id,
            "ticker": ticker,
            "shares": shares,
            "avg_cost": avg_cost,
            "status": "tracked",
        }
