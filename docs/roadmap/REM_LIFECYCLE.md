# The REM Lifecycle: Memory Distillation & Cache Purge

The REM (Rapid Eye Movement) cycle is Wisdom's mechanism for consolidating ephemeral chat data into the permanent Knowledge Substrate (Cortex).

## 1. Lifecycle States
A conversation session passes through these states:
- **ACTIVE**: Real-time interaction. Data stored in SQLite (Edge) and Firestore (Signals).
- **IDLE**: No activity for > 20 minutes.
- **ABANDONED**: No activity for > 2 hours.
- **DISTILLED**: REM has extracted high-signal facts. Cache cleared.

## 2. Trigger Mechanisms

### A. Event-Driven (Post-Session)
When the user explicitly ends a chat or the `abandoned` timeout is reached:
1.  **Extraction**: `REMService` scans the Firestore/SQLite logs.
2.  **Entity Mapping**: Identified `@People` and `#Topics` are registered in the `EntityDictionary`.
3.  **Fact Promotion**: Significant conclusions are moved to the "Hechos" Layer (Cloud SQL).
4.  **Purge**: The local SQLite cache for that session is deleted.

### B. Periodic (The "Nightly" REM)
Every 24 hours (e.g., 03:00 AM local time):
1.  **Global Audit**: Scan all sessions from the last 24h.
2.  **Cross-Session Linking**: Find relationships between different conversations (e.g., you mentioned a book in Chat A and a chess move in Chat B).
3.  **Signal Decay**: Prune low-impact "Signals" (TSR < 0.1).

## 3. Resilience Scenarios
- **System Crash during REM**: Logs are persisted in Firestore *before* purging SQLite, ensuring we can retry the distillation.
- **Manual "Remember This"**: A user command `!save` triggers an immediate mini-REM for the current block.
