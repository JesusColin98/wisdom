package thalamus

import (
	"context"
	"fmt"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// SREAssistant implements Phase 4 KG-RAG logic for Causal RCASage.
type SREAssistant struct {
	Cortex *cortex.Cortex
}

// NewSREAssistant creates a new SREAssistant.
func NewSREAssistant(cx *cortex.Cortex) *SREAssistant {
	return &SREAssistant{Cortex: cx}
}

// CausalChain represents a path of related production events/nodes.
type CausalChain struct {
	Nodes []cortex.Node `json:"nodes"`
	Score float64       `json:"score"`
}

// TraceCausalPath explores the KG for "Causal Chains" of production issues.
// It looks for patterns like: Error Node -> Component -> Dependent Service -> Recent Rollout.
func (a *SREAssistant) TraceCausalPath(ctx context.Context, incidentNodeID string) ([]CausalChain, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.SREAssistant.TraceCausalPath")
	defer span.End()

	// 1. Fetch the incident anchor node
	root, err := a.Cortex.GetNode(ctx, incidentNodeID)
	if err != nil || root == nil {
		return nil, fmt.Errorf("incident node not found: %w", err)
	}

	// 2. Discover potential causal paths via KG Traversal
	// We look for specifically 'CAUSED_BY', 'LINKED_TO', 'DEPLOYED_IN' relationships.
	query := `
		SELECT n2.id, l1.relation_type, n3.id, l2.relation_type, n4.id, l3.relation_type
		FROM nodes n1
		JOIN links l1 ON n1.id = l1.source_id
		JOIN nodes n2 ON l1.target_id = n2.id
		LEFT JOIN links l2 ON n2.id = l2.source_id
		LEFT JOIN nodes n3 ON l2.target_id = n3.id
		LEFT JOIN links l3 ON n3.id = l3.source_id
		LEFT JOIN nodes n4 ON l3.target_id = n4.id
		WHERE n1.id = ? 
		  AND l1.relation_type IN ('CAUSED_BY', 'AFFECTS', 'REPORTED_IN')
		LIMIT 10
	`
	// This is a simplified causal trace. In a full implementation, we'd use recursive depth 
	// with semantic filters for SRE-specific entity classes (ERROR, SERVICE, ROLLOUT).

	rows, err := a.Cortex.DB().QueryContext(ctx, query, incidentNodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []CausalChain
	for rows.Next() {
		var id2, rel1, id3, rel2, id4, rel3 *string
		if err := rows.Scan(&id2, &rel1, &id3, &rel2, &id4, &rel3); err != nil {
			continue
		}

		chain := CausalChain{Nodes: []cortex.Node{*root}}
		
		// Helper to hydrate and add node
		addNode := func(id *string) {
			if id != nil {
				n, _ := a.Cortex.GetNode(ctx, *id)
				if n != nil {
					chain.Nodes = append(chain.Nodes, *n)
				}
			}
		}

		addNode(id2)
		addNode(id3)
		addNode(id4)

		// Simple scoring based on length and relevance
		chain.Score = float64(len(chain.Nodes)) / 4.0
		chains = append(chains, chain)
	}

	return chains, nil
}
