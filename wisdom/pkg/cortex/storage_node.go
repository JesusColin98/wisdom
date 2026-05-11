package cortex

import (
	"context"
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

// DeleteNode removes a node from storage.
func (c *Cortex) DeleteNode(ctx context.Context, id string) error {
	return c.engine.DeleteNode(ctx, id)
}

// ListDueNodes retrieves nodes that are due for spaced repetition review.
func (c *Cortex) ListDueNodes(ctx context.Context, namespaceID string, limit int) ([]Node, error) {
	return c.engine.ListDueNodes(ctx, namespaceID, limit)
}

// UpdateConfidence adjusts the truth-score of a node based on feedback.
func (c *Cortex) UpdateConfidence(ctx context.Context, nodeID string, delta float64) error {
	return c.engine.UpdateConfidence(ctx, nodeID, delta)
}

// UpdateImpact adjusts the impact score of a node.
func (c *Cortex) UpdateImpact(ctx context.Context, nodeID string, delta float64) error {
	return c.engine.UpdateImpact(ctx, nodeID, delta)
}

// MoveToCold moves a node to the COLD stratum.
func (c *Cortex) MoveToCold(ctx context.Context, nodeID string) error {
	return c.engine.MoveToCold(ctx, nodeID)
}

// RecallToHot moves a node to the HOT stratum.
func (c *Cortex) RecallToHot(ctx context.Context, nodeID string) error {
	return c.engine.RecallToHot(ctx, nodeID)
}

// StrengthenSynapse increases the confidence score of an existing node.
func (c *Cortex) StrengthenSynapse(ctx context.Context, nodeID string) error {
	return c.UpdateConfidence(ctx, nodeID, 0.05)
}
