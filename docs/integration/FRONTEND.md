# Integration Guide: Project Wisdom Frontend

This guide explains how to leverage Wisdom's Knowledge Runtime to create high-utility, visually aware applications.

## 1. Visualizing the Knowledge Mesh

Wisdom stores data as a stratified graph. To visualize it effectively:
1. **Stratum Colors:** 
   - `HOT` nodes (SQLite/Cache): Neon Green.
   - `COLD` nodes (OdinANN/Deep): Deep Blue.
2. **Relational Layout:** Use a force-directed graph to show dependencies (`DEPENDS_ON`, `PARENT_OF`).

## 2. Handling Metabolic Alerts (Dopamine Loop)

Project Wisdom broadcasts its efficiency via WebSockets.
- **TSR Low (< 0.4):** The frontend should show a "Low Signal" warning or suggest the user refine their query to avoid wasting tokens.
- **Dopamine Spike:** Trigger visual reinforcement (e.g., subtle glow) when `IMPACT_SCORE` is high.

## 3. Real-time Hallucination Guard

As your chatbot streams text, send chunks to `/ws/metabolism` for validation.
- **Strict Mode:** If validation fails, the frontend should immediately highlight the "ungrounded" text with a red underline.
- **Adaptive Strictness:** The system automatically shifts between `DYNAMIC` (Creative) and `STRICT` (Technical) modes based on user intent.

## 4. Hierarchy Exploration

Use the `/cortex/lineage` endpoint to allow users to "drill down" or "zoom out" of hierarchies. This is ideal for org charts, file trees, or historical lineages.
