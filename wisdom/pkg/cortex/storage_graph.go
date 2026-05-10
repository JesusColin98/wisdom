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
	query := `
		INSERT INTO links (source_id, target_id, relation_type, weight)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(source_id, target_id, relation_type) DO UPDATE SET
			weight = excluded.weight
	`
	_, err := c.db.ExecContext(ctx, query, link.SourceID, link.TargetID, link.RelationType, link.Weight)
	return err
}

// GetNeighbors retrieves all nodes connected to a given node.
func (c *Cortex) GetNeighbors(ctx context.Context, nodeID string) ([]*Node, error) {
	query := `
		SELECT n.id, n.content, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.created_at, n.updated_at
		FROM nodes n
		JOIN links l ON n.id = l.target_id
		WHERE l.source_id = ?
	`
	rows, err := c.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []*Node{}
	for rows.Next() {
		var node Node
		var metadataRaw []byte
		err := rows.Scan(
			&node.ID, &node.Content, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, &node)
	}
	return nodes, nil
}

// Edge represents an edge in the graph for propagation.
type Edge struct {
	SourceID string
	TargetID string
	Weight   float64
}

// GetEdges retrieves edges starting from a set of nodes.
func (c *Cortex) GetEdges(ctx context.Context, sourceIDs []string) ([]Edge, error) {
	if len(sourceIDs) == 0 {
		return nil, nil
	}

	placeholders := ""
	args := make([]any, len(sourceIDs))
	for i, id := range sourceIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}

	query := fmt.Sprintf(`SELECT source_id, target_id, weight FROM links WHERE source_id IN (%s)`, placeholders)
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.SourceID, &e.TargetID, &e.Weight); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, nil
}

// ListEdges retrieves all relationships (links) in the Cortex.
func (c *Cortex) ListEdges(ctx context.Context) ([]Link, error) {
	query := `SELECT source_id, target_id, relation_type, weight, created_at FROM links`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []Link{}
	for rows.Next() {
		var link Link
		err := rows.Scan(&link.SourceID, &link.TargetID, &link.RelationType, &link.Weight, &link.CreatedAt)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
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
// W_{t+1} = W_t * e^{-\lambda * \Delta t}
func (c *Cortex) PruneNodes(ctx context.Context, lambda float64, threshold float64) (int, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.PruneNodes")
	defer span.End()

	// 1. Decay confidence scores based on time since last update
	decayQuery := `
		UPDATE nodes 
		SET confidence_score = confidence_score * exp(-? * (strftime('%s', 'now') - strftime('%s', updated_at))),
		    updated_at = CURRENT_TIMESTAMP
		WHERE confidence_score > 0
	`
	_, err := c.db.ExecContext(ctx, decayQuery, lambda)
	if err != nil {
		return 0, err
	}

	// 2. Delete nodes below threshold and with low impact score
	deleteQuery := `DELETE FROM nodes WHERE confidence_score < ? AND impact_score < ?`
	res, err := c.db.ExecContext(ctx, deleteQuery, threshold, threshold)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := res.RowsAffected()
	return int(rowsAffected), nil
}
