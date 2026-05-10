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
	metadataJSON, err := json.Marshal(node.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	linksJSON, err := json.Marshal(node.ExternalLinks)
	if err != nil {
		return fmt.Errorf("failed to marshal external links: %w", err)
	}

	query := `
		INSERT INTO nodes (id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, superseded_by_id, valid_from, valid_until, repetition_count, easiness_factor, next_review_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			entity_class = excluded.entity_class,
			author = excluded.author,
			source_type = excluded.source_type,
			source_ref = excluded.source_ref,
			metadata = excluded.metadata,
			confidence_score = excluded.confidence_score,
			impact_score = excluded.impact_score,
			stratum = excluded.stratum,
			source_mime_type = excluded.source_mime_type,
			external_links = excluded.external_links,
			superseded_by_id = excluded.superseded_by_id,
			valid_from = excluded.valid_from,
			valid_until = excluded.valid_until,
			repetition_count = excluded.repetition_count,
			easiness_factor = excluded.easiness_factor,
			next_review_at = excluded.next_review_at,
			updated_at = CURRENT_TIMESTAMP
	`
	var supersededBy sql.NullString
	if node.SupersededByID != "" {
		supersededBy = sql.NullString{String: node.SupersededByID, Valid: true}
	}

	var nextReviewAt sql.NullTime
	if !node.NextReviewAt.IsZero() {
		nextReviewAt = sql.NullTime{Time: node.NextReviewAt, Valid: true}
	}

	if node.Stratum == "" {
		node.Stratum = "HOT"
	}
	if node.SourceMimeType == "" {
		node.SourceMimeType = "text/plain"
	}

	_, err = c.db.ExecContext(ctx, query,
		node.ID, node.Content, node.EntityClass, node.Author, node.SourceType,
		node.SourceRef, node.NamespaceID, metadataJSON, node.ConfidenceScore,
		node.ImpactScore, node.Stratum, node.SourceMimeType, linksJSON, supersededBy, node.ValidFrom, node.ValidUntil,
		node.RepetitionCount, node.EasinessFactor, nextReviewAt,
	)
	if err != nil {
		observability.Logger.Error("PutNode failed", "error", err, "node_id", node.ID, "namespace_id", node.NamespaceID, "superseded_by_id", node.SupersededByID)
		return err
	}

	// Update SCG-Mem Trie
	c.trie.Insert(node.Content, node.ID)
	c.trie.Insert(node.ID, node.ID)

	return nil
}

// GetNode retrieves a wisdom node by ID.
func (c *Cortex) GetNode(ctx context.Context, id string) (*Node, error) {
	query := `SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, superseded_by_id, valid_from, valid_until, repetition_count, easiness_factor, next_review_at, created_at, updated_at FROM nodes WHERE id = ?`
	row := c.db.QueryRowContext(ctx, query, id)

	var node Node
	var metadataRaw, linksRaw []byte
	var supersededByID, validFrom, validUntil sql.NullString
	var nextReviewAt sql.NullTime

	err := row.Scan(
		&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
		&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
		&node.ImpactScore, &node.Stratum, &node.SourceMimeType, &linksRaw, &supersededByID, &validFrom, &validUntil,
		&node.RepetitionCount, &node.EasinessFactor, &nextReviewAt,
		&node.CreatedAt, &node.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if supersededByID.Valid { node.SupersededByID = supersededByID.String }
	if validFrom.Valid { node.ValidFrom, _ = time.Parse(time.RFC3339, validFrom.String) }
	if validUntil.Valid { node.ValidUntil, _ = time.Parse(time.RFC3339, validUntil.String) }
	if nextReviewAt.Valid { node.NextReviewAt = nextReviewAt.Time }

	if err := json.Unmarshal(metadataRaw, &node.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	if err := json.Unmarshal(linksRaw, &node.ExternalLinks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal external links: %w", err)
	}

	return &node, nil
}

// ListNodes retrieves all nodes in a specific namespace.
func (c *Cortex) ListNodes(ctx context.Context, namespaceID string) ([]Node, error) {
	query := `SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, created_at, updated_at FROM nodes WHERE namespace_id = ?`
	rows, err := c.db.QueryContext(ctx, query, namespaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		var node Node
		var metadataRaw []byte
		err := rows.Scan(
			&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
			&node.ImpactScore, &node.Stratum, &node.SourceMimeType,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadataRaw, &node.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for node %s: %w", node.ID, err)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// SearchNodes performs a keyword search on node content, ID, and entity class.
func (c *Cortex) SearchNodes(ctx context.Context, query string) ([]Node, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.SearchNodes")
	defer span.End()

	sqlQuery := `
		SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, created_at, updated_at
		FROM nodes
		WHERE content LIKE ? OR id LIKE ? OR entity_class LIKE ?
	`
	likeQuery := "%" + query + "%"
	rows, err := c.db.QueryContext(ctx, sqlQuery, likeQuery, likeQuery, likeQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		var node Node
		var metadataRaw, linksRaw []byte
		err := rows.Scan(
			&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
			&node.ImpactScore, &node.Stratum, &node.SourceMimeType, &linksRaw,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
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
