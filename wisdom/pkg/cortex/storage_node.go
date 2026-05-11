package cortex

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/wisdom/pkg/observability"
)

// PutNode inserts or updates a wisdom node.
func (c *Cortex) PutNode(ctx context.Context, node *Node) error {
	return c.engine.PutNode(ctx, node)
}

// GetNode retrieves a wisdom node by ID.
func (c *Cortex) GetNode(ctx context.Context, id string) (*Node, error) {
	return c.engine.GetNode(ctx, id)
}

// ListNodes retrieves all nodes in a specific namespace.
func (c *Cortex) ListNodes(ctx context.Context, namespaceID string) ([]Node, error) {
	return c.engine.ListNodes(ctx, map[string]any{"namespace_id": namespaceID})
}

// SearchNodes performs a keyword search on node content, ID, and entity class.
func (c *Cortex) SearchNodes(ctx context.Context, query string) ([]Node, error) {
	return c.engine.SearchNodes(ctx, query)
}

// PruneNodes applies entropy-based decay to certainty weights and removes low-signal nodes.
func (c *Cortex) PruneNodes(ctx context.Context, lambda float64, threshold float64) (int, error) {
	return c.engine.PruneNodes(ctx, lambda, threshold)
}

// DeleteNode removes a node from storage.
func (c *Cortex) DeleteNode(ctx context.Context, id string) error {
	return c.engine.DeleteNode(ctx, id)
}


// ListDueNodes retrieves nodes that are due for spaced repetition review.
func (c *Cortex) ListDueNodes(ctx context.Context, namespaceID string, limit int) ([]Node, error) {
	query := `
		SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, superseded_by_id, valid_from, valid_until, repetition_count, easiness_factor, next_review_at, created_at, updated_at
		FROM nodes
		WHERE namespace_id = ? AND next_review_at <= CURRENT_TIMESTAMP
		ORDER BY next_review_at ASC
		LIMIT ?
	`
	rows, err := c.db.QueryContext(ctx, query, namespaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		var node Node
		var metadataRaw, linksRaw []byte
		var supersededByID, validFrom, validUntil sql.NullString
		var nextReviewAt sql.NullTime

		err := rows.Scan(
			&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
			&node.ImpactScore, &node.Stratum, &node.SourceMimeType, &linksRaw, &supersededByID, &validFrom, &validUntil,
			&node.RepetitionCount, &node.EasinessFactor, &nextReviewAt,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if supersededByID.Valid { node.SupersededByID = supersededByID.String }
		if validFrom.Valid { node.ValidFrom, _ = time.Parse(time.RFC3339, validFrom.String) }
		if validUntil.Valid { node.ValidUntil, _ = time.Parse(time.RFC3339, validUntil.String) }
		if nextReviewAt.Valid { node.NextReviewAt = nextReviewAt.Time }

		if err := json.Unmarshal(metadataRaw, &node.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for node %s: %w", node.ID, err)
		}
		if err := json.Unmarshal(linksRaw, &node.ExternalLinks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal external links for node %s: %w", node.ID, err)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// StrengthenSynapse increases the confidence score of an existing node.
func (c *Cortex) StrengthenSynapse(ctx context.Context, nodeID string) error {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.StrengthenSynapse")
	defer span.End()

	query := `
		UPDATE nodes 
		SET confidence_score = MIN(1.0, confidence_score + 0.05),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := c.db.ExecContext(ctx, query, nodeID)
	return err
}

// UpdateConfidence adjusts the truth-score of a node based on feedback.
func (c *Cortex) UpdateConfidence(ctx context.Context, nodeID string, delta float64) error {
	query := `UPDATE nodes SET confidence_score = MIN(1.0, MAX(0.0, confidence_score + ?)), updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := c.db.ExecContext(ctx, query, delta, nodeID)
	return err
}

// ResolvePointer resolves a semantic alias or ID to a specific node ID.
// It prioritizes SYNONYM_OF links to find canonical entities.
func (c *Cortex) ResolvePointer(ctx context.Context, pointer string) (string, error) {
	// 1. Synonym check (Highest priority for aliases)
	query := `
		SELECT target_id 
		FROM links 
		WHERE source_id = ? AND relation_type = 'SYNONYM_OF'
		ORDER BY weight DESC LIMIT 1
	`
	var id string
	err := c.db.QueryRowContext(ctx, query, pointer).Scan(&id)
	if err == nil {
		return id, nil
	}

	// 2. Direct ID check
	err = c.db.QueryRowContext(ctx, `SELECT id FROM nodes WHERE id = ?`, pointer).Scan(&id)
	if err == nil {
		return id, nil
	}

	return "", fmt.Errorf("could not resolve pointer: %s", pointer)
}

// UpdateImpact adjusts the impact score of a node.
func (c *Cortex) UpdateImpact(ctx context.Context, nodeID string, delta float64) error {
	query := `UPDATE nodes SET impact_score = MIN(1.0, MAX(0.0, impact_score + ?)), updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := c.db.ExecContext(ctx, query, delta, nodeID)
	return err
}

// MoveToCold moves a node to the COLD stratum.
func (c *Cortex) MoveToCold(ctx context.Context, nodeID string) error {
	query := `UPDATE nodes SET stratum = 'COLD', updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := c.db.ExecContext(ctx, query, nodeID)
	return err
}

// RecallToHot moves a node to the HOT stratum.
func (c *Cortex) RecallToHot(ctx context.Context, nodeID string) error {
	query := `UPDATE nodes SET stratum = 'HOT', updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := c.db.ExecContext(ctx, query, nodeID)
	return err
}
