package thalamus

import (
	"context"
	"fmt"
	"math"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// RiskEngine implements Phase 4 Star-Mesh hybrid logic for Entity Risk Scoring.
type RiskEngine struct {
	Cortex *cortex.Cortex
}

// NewRiskEngine creates a new RiskEngine.
func NewRiskEngine(cx *cortex.Cortex) *RiskEngine {
	return &RiskEngine{Cortex: cx}
}

// RiskScore represents the calculated risk for a node.
type RiskScore struct {
	NodeID     string  `json:"node_id"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

// CalculateEntityRisk identifies hotspots and propagates risk scores.
func (e *RiskEngine) CalculateEntityRisk(ctx context.Context, rootID string, depth int) ([]RiskScore, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.RiskEngine.CalculateEntityRisk")
	defer span.End()

	// 1. Identify Star Center (The Hotspot)
	// Risk is high if a node has many 'NEGATIVELY_IMPACTS' or 'FAILED_BY' links.
	query := `
		SELECT n.id, COUNT(l.id) as negative_link_count
		FROM nodes n
		JOIN links l ON n.id = l.target_id
		WHERE l.relation_type IN ('NEGATIVELY_IMPACTS', 'FAILED_BY', 'RISK_FACTOR')
		  AND n.id = ?
		GROUP BY n.id
	`
	var nodeID string
	var negCount int
	err := e.Cortex.DB().QueryRowContext(ctx, query, rootID).Scan(&nodeID, &negCount)
	
	baseRisk := 0.0
	if err == nil {
		// Logarithmic scale for base risk based on negative associations
		baseRisk = math.Min(1.0, float64(negCount)/10.0)
	} else {
		// If node has no direct negative links, we check its inherent confidence
		node, err := e.Cortex.GetNode(ctx, rootID)
		if err == nil && node != nil {
			baseRisk = 1.0 - node.ConfidenceScore
		}
	}

	// 2. Mesh Propagation (Weighted Decay)
	// Risk propagates to dependencies: if A depends on B, and B is risky, A's risk increases.
	// We use the Propagate method from Cortex but with a custom "Risk Decay" logic.
	
	// For now, let's use the semantic propagation as a proxy for risk spread, 
	// weighting by dependency types.
	propagation, err := e.Cortex.Propagate(ctx, []string{rootID}, 0.70, depth)
	if err != nil {
		return nil, fmt.Errorf("risk propagation failed: %w", err)
	}

	var results []RiskScore
	for id, influence := range propagation {
		score := influence * baseRisk
		if score < 0.1 {
			continue // Filter noise
		}

		reason := "Propagated risk via dependency mesh"
		if id == rootID {
			reason = "Direct risk hotspot center"
		}

		results = append(results, RiskScore{
			NodeID:     id,
			Score:      score,
			Reason:     reason,
			Confidence: baseRisk, // Simplification
		})
	}

	return results, nil
}
