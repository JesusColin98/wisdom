# Integration Guide: Project Wisdom MCP Agent

This guide explains how to expose Project Wisdom as a **Model Context Protocol (MCP)** tool provider.

## 1. Wisdom as a Tool Provider

Other agents (e.g., SRE Agent, Developer Agent) can use Wisdom as their primary knowledge runtime.

### Available Tools:
- **`recall_wisdom`:** The primary retrieval tool. It detects intent and returns pattern-aware context.
- **`trace_causality`:** (KG Pattern) Specialized for root-cause analysis.
- **`calculate_risk`:** (Star Pattern) Returns entity risk scoring.

## 2. Shared Cortex Mandate

In an MCP environment, Wisdom acts as the "Shared Brain". Multiple agents contributing to the same Cortex reinforce the graph over time, creating a virtuous cycle of expertise.

## 3. Implementation (Python Example)

```python
# Using Wisdom as an MCP Tool
@mcp.tool()
async def get_system_context(query: str, user_id: str):
    """Retrieves grounded context from Project Wisdom runtime."""
    response = await wisdom_client.recall(
        user_id=user_id,
        query=query,
        seeds=[],
        uncertainty=0.7
    )
    return response.wisdom
```

## 4. Grounding with SCG-Mem

Agents should use the `validate_output` tool during their final output phase to ensure they aren't hallucinating structural technical terms that contradict the Cortex.
