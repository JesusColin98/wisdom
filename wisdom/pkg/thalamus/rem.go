package thalamus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// REMService handles the consolidation of transient memories into the long-term Cortex.
// REM stands for Rapid Evidence Mapping (or the biological REM cycle).
type REMService struct {
	Hippocampus *Hippocampus
	Cortex      *cortex.Cortex
	Chat        *Chat
	Clustering  *ClusteringService
}

// ConsolidateSession scans the Hippocampus logs for a session, extracts wisdom,
// and anchors it into the Cortex.
func (s *REMService) ConsolidateSession(ctx context.Context, sessionID string) (int, error) {
	ctx, span := observability.Tracer.Start(ctx, "REM.Consolidate")
	defer span.End()

	logs, err := s.Hippocampus.GetLogs(ctx, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch logs: %w", err)
	}
	if len(logs) == 0 {
		return 0, nil
	}

	// 1. Build distillation prompt
	var logBuilder strings.Builder
	for _, l := range logs {
		logBuilder.WriteString(fmt.Sprintf("%s: %s\n", l.Role, l.Content))
	}

	prompt := fmt.Sprintf(`Analyze the following session logs and extract "Universal Truths", "Entity Attributes", or "Historical Transitions" that should be remembered permanently.
Focus on:
1. ENTITIES: Identify people, concepts, or systems (e.g., "User: Jesus", "Concept: Sicilian Defense", "Service: JobServer").
2. ATTRIBUTES: Extract level, roles, strengths, or weaknesses (e.g., "ELO: 1500", "Role: Tech Lead", "Vocabulary: 'Pristine' - meaning clean").
3. TRANSITIONS: Note if a previous fact is superseded (e.g., "Role changed from SRE to Manager").
4. PATTERNS: Recurring errors or best practices.

Format each finding as a separate paragraph. Categorize them conceptually.
Logs:
%s`, logBuilder.String())

	// 2. Use LLM to distill wisdom
	wisdomText, err := s.Chat.LLM.Complete(ctx, prompt)
	if err != nil {
		return 0, fmt.Errorf("failed to distill wisdom via LLM: %w", err)
	}

	// 3. Anchor findings to Cortex
	findings := strings.Split(wisdomText, "\n\n")
	anchoredCount := 0
	for _, f := range findings {
		content := strings.TrimSpace(f)
		if content == "" {
			continue
		}

		// 3.1 Generate embedding for finding
		embedding, err := s.Chat.LLM.Embed(ctx, content)
		if err != nil {
			observability.Logger.Warn("Failed to generate embedding for finding", "error", err)
			// Continue with standard anchoring if embedding fails
		}

		// 3.2 Thalamic Gate: Novelty Check with Dynamic Thresholding
		if embedding != nil {
			// Find the best match regardless of threshold first
			results, err := s.Cortex.VectorSearch(ctx, embedding, 1)
			if err != nil {
				observability.Logger.Error("Failed similarity check", "error", err)
			} else if len(results) > 0 {
				match := results[0]
				
				// Dynamic Threshold:
				// If existing node has low confidence, we are more lenient (0.88) to consolidate evidence.
				// If existing node is a "Universal Truth" (high confidence), we are stricter (0.94).
				threshold := 0.92
				if match.ConfidenceScore < 0.3 {
					threshold = 0.88
				} else if match.ConfidenceScore > 0.8 {
					threshold = 0.94
				}

				if match.Score >= threshold {
					// Redundant knowledge: Strengthen existing synapse
					if err := s.Cortex.StrengthenSynapse(ctx, match.ID); err == nil {
						observability.Logger.Info("Thalamic Gate: Knowledge consolidated (Dynamic Threshold)", 
							"node_id", match.ID, 
							"similarity", match.Score, 
							"threshold", threshold)
						continue
					}
				}
			}
		}

		node := &cortex.Node{
			ID:              fmt.Sprintf("rem-%s-%d", sessionID, anchoredCount),
			Content:         content,
			Author:          "wisdom-rem",
			SourceType:      "REM_CYCLE",
			SourceRef:       sessionID,
			NamespaceID:     "ns-general", // Default namespace
			ConfidenceScore: 0.5,          // Initial confidence
		}

		if err := s.Cortex.PutNode(ctx, node); err != nil {
			continue
		}

		// Store vector if generated
		if embedding != nil {
			if err := s.Cortex.PutVector(ctx, node.ID, embedding, "distil-v1"); err != nil {
				observability.Logger.Error("Failed to store vector for new node", "node_id", node.ID, "error", err)
			}
		}

		anchoredCount++
	}

	// 4. Clear Hippocampus
	if err := s.Hippocampus.Clear(ctx, sessionID); err != nil {
		return anchoredCount, fmt.Errorf("failed to clear logs: %w", err)
	}

	return anchoredCount, nil
}

// ConsolidateAllSessions finds all inactive sessions and consolidates them.
// This is the core of the automated "Daily REM Cycle".
func (s *REMService) ConsolidateAllSessions(ctx context.Context, inactiveFor time.Duration) (int, error) {
	ctx, span := observability.Tracer.Start(ctx, "REM.ConsolidateAll")
	defer span.End()

	sessions, err := s.Cortex.GetInactiveSessions(ctx, inactiveFor)
	if err != nil {
		return 0, fmt.Errorf("failed to list inactive sessions: %w", err)
	}

	totalAnchored := 0
	for _, sid := range sessions {
		count, err := s.ConsolidateSession(ctx, sid)
		if err != nil {
			// Log error but continue with other sessions
			observability.Logger.Error("Failed to consolidate session", "session_id", sid, "error", err)
			continue
		}
		totalAnchored += count
	}

	// Trigger Dynamic Reorganization if new knowledge was anchored
	if totalAnchored > 0 && s.Clustering != nil {
		moved, err := s.Clustering.ReorganizeNSGeneral(ctx)
		if err != nil {
			observability.Logger.Error("Failed to reorganize ns-general", "error", err)
		} else if moved > 0 {
			observability.Logger.Info("Dynamic Reorganization complete", "clusters_moved", moved)
		}
	}

	return totalAnchored, nil
}
