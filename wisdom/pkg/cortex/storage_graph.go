package cortex

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/wisdom/pkg/observability"
	"golang.org/x/sync/errgroup"
)

// LinkNodes creates a directed relationship between two nodes.
func (c *Cortex) LinkNodes(ctx context.Context, link *Link) error {
	return c.engine.LinkNodes(ctx, link)
}

// GetNeighbors retrieves all nodes connected to a given node.
func (c *Cortex) GetNeighbors(ctx context.Context, nodeID string) ([]*Node, error) {
	return c.engine.GetNeighbors(ctx, nodeID)
}

// Edge represents an edge in the graph for propagation.
type Edge struct {
	SourceID string
	TargetID string
	Weight   float64
}

// GetEdges retrieves edges starting from a set of nodes.
func (c *Cortex) GetEdges(ctx context.Context, sourceIDs []string) ([]Edge, error) {
	return c.engine.GetEdges(ctx, sourceIDs)
}

// ListEdges retrieves all relationships (links) in the Cortex.
func (c *Cortex) ListEdges(ctx context.Context) ([]Link, error) {
	return c.engine.ListEdges(ctx)
}

// ResolvePointer resolves a semantic alias or ID to a specific node ID.
func (c *Cortex) ResolvePointer(ctx context.Context, pointer string) (string, error) {
	return c.engine.ResolvePointer(ctx, pointer)
}

// Propagate implements Personalized PageRank (PPR) for wisdom nodes.
func (c *Cortex) Propagate(ctx context.Context, seedIDs []string, alpha float64, iterations int) (map[string]float64, error) {
	_, span := observability.Tracer.Start(ctx, "Cortex.Propagate")
	defer span.End()

	if len(seedIDs) == 0 {
		return nil, nil
	}

	scores := make(map[string]float64)
	initialScore := 1.0 / float64(len(seedIDs))
	for _, id := range seedIDs {
		scores[id] = initialScore
	}

	teleport := (1 - alpha) / float64(len(seedIDs))

	for i := 0; i < iterations; i++ {
		newScores := make(map[string]float64)
		var mu sync.Mutex

		var currentNodes []string
		for id, score := range scores {
			if score > 0 {
				currentNodes = append(currentNodes, id)
			}
		}

		const batchSize = 100
		g, gCtx := errgroup.WithContext(ctx)

		for j := 0; j < len(currentNodes); j += batchSize {
			end := j + batchSize
			if end > len(currentNodes) {
				end = len(currentNodes)
			}
			batch := currentNodes[j:end]

			g.Go(func() error {
				edges, err := c.GetEdges(gCtx, batch)
				if err != nil {
					return err
				}

				mu.Lock()
				defer mu.Unlock()
				for _, edge := range edges {
					contribution := scores[edge.SourceID] * alpha * edge.Weight
					newScores[edge.TargetID] += contribution
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, fmt.Errorf("failed to fetch edges in iteration %d: %w", i, err)
		}

		for _, seed := range seedIDs {
			newScores[seed] += teleport
		}
		scores = newScores
	}

	return scores, nil
}

// PruneNodes applies entropy-based decay to certainty weights and removes low-signal nodes.
func (c *Cortex) PruneNodes(ctx context.Context, lambda float64, threshold float64) (int, error) {
	return c.engine.PruneNodes(ctx, lambda, threshold)
}

