package thalamus

import (
	"context"
	"fmt"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// HierarchyManager implements Phase 4 Tree-RAG logic.
// It uses recursive CTEs to reconstruct deep lineages (Org charts, Role history).
type HierarchyManager struct {
	Cortex *cortex.Cortex
}

// NewHierarchyManager creates a new HierarchyManager.
func NewHierarchyManager(cx *cortex.Cortex) *HierarchyManager {
	return &HierarchyManager{Cortex: cx}
}

// GetLineage retrieves the full ancestral or descendant tree for a node.
// direction: "UP" (Ancestors/Parents) or "DOWN" (Descendants/Children)
func (m *HierarchyManager) GetLineage(ctx context.Context, nodeID string, direction string) ([]cortex.Node, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Hierarchy.GetLineage")
	defer span.End()

	linkField := "source_id"
	targetField := "target_id"
	if direction == "UP" {
		linkField = "target_id"
		targetField = "source_id"
	}

	// Recursive CTE to find all related nodes in the hierarchy
	query := fmt.Sprintf(`
		WITH RECURSIVE lineage AS (
			-- Anchor member
			SELECT %s as id, 0 as depth
			FROM links
			WHERE %s = ? AND relation_type IN ('PARENT_OF', 'MEMBER_OF', 'REPORTS_TO')
			
			UNION ALL
			
			-- Recursive member
			SELECT l.%s, lineage.depth + 1
			FROM links l
			JOIN lineage ON l.%s = lineage.id
			WHERE l.relation_type IN ('PARENT_OF', 'MEMBER_OF', 'REPORTS_TO')
			  AND lineage.depth < 10 -- Safety cap
		)
		SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.created_at, n.updated_at
		FROM nodes n
		JOIN lineage ON n.id = lineage.id
		GROUP BY n.id
		ORDER BY lineage.depth ASC
	`, targetField, linkField, targetField, linkField)

	rows, err := m.Cortex.DB().QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("lineage query failed: %w", err)
	}
	defer rows.Close()

	var nodes []cortex.Node
	for rows.Next() {
		var n cortex.Node
		var metadataRaw []byte
		err := rows.Scan(
			&n.ID, &n.Content, &n.EntityClass, &n.Author, &n.SourceType,
			&n.SourceRef, &n.NamespaceID, &metadataRaw, &n.ConfidenceScore,
			&n.CreatedAt, &n.UpdatedAt,
		)
		if err == nil {
			nodes = append(nodes, n)
		}
	}

	return nodes, nil
}
