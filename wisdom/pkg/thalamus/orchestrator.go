package thalamus

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// Orchestrator aggregates context from various substrates.
type Orchestrator struct {
	cortex *cortex.Cortex
	cache  *Cache
}

// NewOrchestrator creates a new Thalamic Orchestrator.
func NewOrchestrator(cx *cortex.Cortex, c *Cache) *Orchestrator {
	return &Orchestrator{
		cortex: cx,
		cache:  c,
	}
}

// ResolvePointer resolves a human-friendly alias to a node ID.
func (o *Orchestrator) ResolvePointer(ctx context.Context, pointer string) (string, error) {
	return o.cortex.ResolvePointer(ctx, pointer)
}

// GetImpactGraph performs a breadth-first traversal of dependencies to identify
// all downstream components affected by a node.
func (o *Orchestrator) GetImpactGraph(ctx context.Context, nodeID string, maxDepth int) ([]cortex.Node, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.GetImpactGraph")
	defer span.End()

	rootID, err := o.cortex.ResolvePointer(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	impacted := make(map[string]cortex.Node)
	queue := []string{rootID}
	depths := map[string]int{rootID: 0}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if depths[curr] >= maxDepth {
			continue
		}

		// 1. Dependency check (Upstream/Downstream)
		query := `
			SELECT target_id 
			FROM links 
			WHERE source_id = ? AND relation_type IN ('DEPENDS_ON', 'PARENT_OF')
		`
		rows, err := o.cortex.DB().QueryContext(ctx, query, curr)
		if err != nil {
			continue
		}
		
		var neighborIDs []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err == nil {
				neighborIDs = append(neighborIDs, id)
			}
		}
		rows.Close()

		for _, id := range neighborIDs {
			if _, seen := impacted[id]; !seen {
				node, err := o.cortex.GetNode(ctx, id)
				if err == nil && node != nil {
					impacted[id] = *node
					depths[id] = depths[curr] + 1
					queue = append(queue, id)
				}
			}
		}
	}

	var results []cortex.Node
	for _, n := range impacted {
		results = append(results, n)
	}
	return results, nil
}

// GetNearbyWisdom explores the graph using Personalized PageRank to find conceptually related nodes,
// even if they don't share keywords with the seed set.
func (o *Orchestrator) GetNearbyWisdom(ctx context.Context, seedIDs []string, iterations int) ([]cortex.ScoredNode, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.GetNearbyWisdom")
	defer span.End()

	// 1. Propagate relevance signals (Synaptic Propagation)
	scores, err := o.cortex.Propagate(ctx, seedIDs, 0.85, iterations)
	if err != nil {
		return nil, err
	}

	// 2. Hydrate nodes
	var results []cortex.ScoredNode
	for id, score := range scores {
		node, err := o.cortex.GetNode(ctx, id)
		if err != nil || node == nil {
			continue
		}
		// Filter out superseded truths to keep the memory clean
		if node.SupersededByID != "" {
			continue
		}
		results = append(results, cortex.ScoredNode{
			Node:  *node,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// Recall fetches context for the LLM, balancing working memory (Hippocampus)
// and long-term wisdom (Cortex).
func (o *Orchestrator) Recall(ctx context.Context, sessionID string, seeds []string, budget int) (*Context, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Recall")
	defer span.End()

	session, ok := o.cache.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	// 1. Semantic Discovery via Synaptic Propagation
	nearby, err := o.GetNearbyWisdom(ctx, seeds, 2)
	if err != nil {
		return nil, fmt.Errorf("wisdom discovery failed: %w", err)
	}

	// 2. Filter and Budgeting
	var aggregated []string
	currentSize := 0
	for _, sn := range nearby {
		// Heuristic: (content length + namespace + metadata overhead) / 4
		nodeCost := (len(sn.Content) + 400) / 4
		if currentSize+nodeCost > budget {
			break
		}

		aggregated = append(aggregated, sn.Content)
		currentSize += nodeCost
	}

	return &Context{
		Session: session,
		Wisdom:  aggregated,
		Budget:  budget,
	}, nil
}
