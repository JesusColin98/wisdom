# Wisdom Configuration Guide

This document explains the tunable parameters of the Knowledge Runtime. These can be adjusted via the `wisdom_config.json` file, the Portal UI, or by the Chatbot to modulate the system's behavior.

## 1. Retrieval Parameters

### `default_retrieval_depth` (Iterations)
- **What it does:** Controls how many "hops" the system takes through the knowledge graph during semantic propagation.
- **Increase when:** You are asking complex "Why" or "How" questions that require connecting multiple distant facts (e.g., "Sam Morgan's role history across 3 years").
- **Decrease when:** You want fast, direct answers to simple facts (e.g., "What is the GAIA ID?").
- **Cost Impact:** Higher depth increases processing time and retrieval cost.

### `uncertainty_threshold` (0.0 - 1.0)
- **What it does:** Determines when to switch from "Low Cost" (Fast/Surface) to "High Cost" (Deep/Socratic) mode.
- **Lower (0.3):** The system becomes "inquisitive" and triggers deep reasoning even for mild doubts.
- **Higher (0.8):** The system is "confident" and prefers fast cached answers unless it's extremely confused.
- **Case Use:** Set to `0.4` for critical debugging sessions where accuracy is paramount.

### `token_budget` (Tokens)
- **What it does:** Caps the total context size injected into the LLM.
- **Increase when:** The model needs detailed background or multiple conflicting viewpoints to resolve a query.
- **Decrease when:** You want to minimize token metabolism and force the system to provide only the most influential (`MCMI`) nodes.

## 2. Validation & Strictness

### `strictness_pressure` (Hallucination Guard)
- **What it does:** Modulates the `SCG-Mem Trie` aggressive filter.
- **1.0 (Strict):** Zero tolerance. If a term isn't in the Cortex, the model's mention of it is flagged.
- **0.1 (Dynamic):** High tolerance. Allows the model to use its internal training data freely.
- **Case Use:** Set to `1.0` when asking for exact API flags or Production policies.

## 3. Knowledge Lifecycle

### `entropy_factor` (Lambda)
- **What it does:** The rate at which certainty weights decay over time ($W_{t} \cdot e^{-\lambda \Delta t}$).
- **Higher (0.05):** "Forgetful" brain. Facts need frequent reinforcement. Good for highly dynamic data.
- **Lower (0.001):** "Persistent" brain. Facts stay trusted for a long time. Good for core principles.

### `pruning_threshold` (Certainty survival)
- **What it does:** The minimum weight a node needs to stay in the `HOT` stratum.
- **Increase when:** The database is getting cluttered with low-signal noise.
- **Decrease when:** You want to preserve even the smallest historical traces.
