package thalamus

import (
	"context"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// ReinforcementService handles synaptic strengthening and pruning signals.
type ReinforcementService struct {
	Cortex *cortex.Cortex
}

// NewReinforcementService creates a new ReinforcementService.
func NewReinforcementService(cx *cortex.Cortex) *ReinforcementService {
	return &ReinforcementService{
		Cortex: cx,
	}
}

// ReinforcePath strengthens the confidence and impact of nodes that were used to solve a task.
// It also strengthens the links between these nodes.
func (s *ReinforcementService) ReinforcePath(ctx context.Context, nodeIDs []string) error {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Reinforce.ReinforcePath")
	defer span.End()

	if len(nodeIDs) == 0 {
		return nil
	}

	// 1. Strengthen individual nodes
	for _, id := range nodeIDs {
		// Increase confidence (Truth score)
		_ = s.Cortex.StrengthenSynapse(ctx, id)
		
		// Increase Impact Score (Pruning survival)
		// We use a small increment to reward usage
		_ = s.Cortex.UpdateImpact(ctx, id, 0.05)
	}

	// 2. Strengthen links between co-retrieved nodes (Association)
	if len(nodeIDs) > 1 {
		for i := 0; i < len(nodeIDs); i++ {
			for j := i + 1; j < len(nodeIDs); j++ {
				// Bi-directional strengthening of association
				_ = s.Cortex.LinkNodes(ctx, &cortex.Link{
					SourceID:     nodeIDs[i],
					TargetID:     nodeIDs[j],
					RelationType: "ASSOCIATED_WITH",
					Weight:       1.0, // Base weight, LinkNodes handles UPSERT/UPDATE
				})
			}
		}
	}

	return nil
}

// PenalizePath decreases the confidence of nodes that led to a failure or were rejected by the user.
func (s *ReinforcementService) PenalizePath(ctx context.Context, nodeIDs []string) error {
	for _, id := range nodeIDs {
		_ = s.Cortex.UpdateConfidence(ctx, id, -0.1)
		_ = s.Cortex.UpdateImpact(ctx, id, -0.05)
	}
	return nil
}

// UpvoteNode provides a strong positive signal for a specific knowledge unit.
func (s *ReinforcementService) UpvoteNode(ctx context.Context, nodeID string) error {
	_ = s.Cortex.UpdateConfidence(ctx, nodeID, 0.2)
	return s.Cortex.UpdateImpact(ctx, nodeID, 0.1)
}

// DownvoteNode provides a strong negative signal for a specific knowledge unit.
func (s *ReinforcementService) DownvoteNode(ctx context.Context, nodeID string) error {
	_ = s.Cortex.UpdateConfidence(ctx, nodeID, -0.2)
	return s.Cortex.UpdateImpact(ctx, nodeID, -0.1)
}

