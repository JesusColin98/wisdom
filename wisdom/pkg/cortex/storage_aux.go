package cortex

import (
	"context"
	"encoding/json"
	"fmt"
)

// CreateNamespace ensures a namespace exists.
func (c *Cortex) CreateNamespace(ctx context.Context, ns *Namespace) error {
	query := `INSERT OR IGNORE INTO namespaces (id, name, description) VALUES (?, ?, ?)`
	_, err := c.db.ExecContext(ctx, query, ns.ID, ns.Name, ns.Description)
	return err
}

// GetHistory retrieves the version history for a given node.
func (c *Cortex) GetHistory(ctx context.Context, nodeID string) ([]NodeHistory, error) {
	query := `
		SELECT node_id, content, metadata, external_links, version_timestamp
		FROM node_history
		WHERE node_id = ?
		ORDER BY history_id DESC
	`
	rows, err := c.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []NodeHistory{}
	for rows.Next() {
		var h NodeHistory
		var metadataRaw, linksRaw []byte
		if err := rows.Scan(&h.NodeID, &h.Content, &metadataRaw, &linksRaw, &h.VersionTimestamp); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadataRaw, &h.Metadata); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(linksRaw, &h.ExternalLinks); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

// ListNodesByLink retrieves nodes connected to a source via a specific relation type.
func (c *Cortex) ListNodesByLink(ctx context.Context, sourceID string, relationType string) ([]Node, error) {
	query := `
		SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.impact_score, n.stratum, n.source_mime_type, n.external_links, n.created_at, n.updated_at
		FROM nodes n
		JOIN links l ON n.id = l.target_id
		WHERE l.source_id = ? AND l.relation_type = ?
	`
	rows, err := c.db.QueryContext(ctx, query, sourceID, relationType)
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
