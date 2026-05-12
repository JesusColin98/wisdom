package thalamus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// Coach provides generic coaching logic leveraging the neural substrate.
type Coach struct {
	Cortex    *cortex.Cortex
	Scheduler *Scheduler
	LLM       cerebellum.LLMProvider
}

// NewCoach initializes a new generic coach.
func NewCoach(c *cortex.Cortex, s *Scheduler, llm cerebellum.LLMProvider) *Coach {
	return &Coach{Cortex: c, Scheduler: s, LLM: llm}
}

// Weakness represents a knowledge gap or a pattern the user struggles with.
type Weakness struct {
	NodeID     string  `json:"node_id"`
	Content    string  `json:"content"`
	Reason     string  `json:"reason"` // e.g., "Linked via STRUGGLES_WITH", "Missing prerequisite"
	Importance float64 `json:"importance"`
}

// DiscoverWeaknesses identifies gaps in a user's knowledge graph.
func (c *Coach) DiscoverWeaknesses(ctx context.Context, userID string) ([]Weakness, error) {
	ctx, span := observability.Tracer.Start(ctx, "Coach.DiscoverWeaknesses")
	defer span.End()

	var weaknesses []Weakness

	// 1. Direct Struggles: Nodes linked via STRUGGLES_WITH from the user
	struggleNodes, err := c.Cortex.ListNodesByLink(ctx, userID, "STRUGGLES_WITH")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch struggles: %w", err)
	}

	for _, node := range struggleNodes {
		weaknesses = append(weaknesses, Weakness{
			NodeID:     node.ID,
			Content:    node.Content,
			Reason:     "Directly identified struggle",
			Importance: 1.0,
		})
	}

	// 2. Prerequisites of Mastered Nodes that are missing
	masteredNodes, err := c.Cortex.ListNodesByLink(ctx, userID, "MASTERED_BY")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mastered nodes: %w", err)
	}

	masteredMap := make(map[string]bool)
	for _, m := range masteredNodes {
		masteredMap[m.ID] = true
	}

	for _, m := range masteredNodes {
		// Find prerequisites: (P) -[PREREQUISITE_OF]-> (M)
		// Wait, PREREQUISITE_OF direction: (Prereq) -> (Dependent)
		// So target is 'm.ID', source is prereq.
		query := `
			SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.created_at, n.updated_at
			FROM nodes n
			JOIN links l ON n.id = l.source_id
			WHERE l.target_id = ? AND l.relation_type = 'PREREQUISITE_OF'
		`
		rows, err := c.Cortex.DB().QueryContext(ctx, query, m.ID)
		if err != nil {
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var p cortex.Node
			var metadataRaw []byte
			err := rows.Scan(
				&p.ID, &p.Content, &p.EntityClass, &p.Author, &p.SourceType,
				&p.SourceRef, &p.NamespaceID, &metadataRaw, &p.ConfidenceScore,
				&p.CreatedAt, &p.UpdatedAt,
			)
			if err != nil {
				continue
			}

			if !masteredMap[p.ID] {
				weaknesses = append(weaknesses, Weakness{
					NodeID:     p.ID,
					Content:    p.Content,
					Reason:     fmt.Sprintf("Missing prerequisite for mastered concept: %s", m.Content),
					Importance: 0.8,
				})
			}
		}
	}

	return weaknesses, nil
}

// ListMasteredNodes returns nodes the user has mastered.
func (c *Coach) ListMasteredNodes(ctx context.Context, userID string) ([]cortex.Node, error) {
	ctx, span := observability.Tracer.Start(ctx, "Coach.ListMasteredNodes")
	defer span.End()

	return c.Cortex.ListNodesByLink(ctx, userID, "MASTERED_BY")
}

// CurriculumItem represents a prioritized learning task.
type CurriculumItem struct {
	NodeID   string  `json:"node_id"`
	Content  string  `json:"content"`
	Priority float64 `json:"priority"` // 0.0 - 1.0
	Type     string  `json:"type"`     // "REVIEW", "NEW_CONCEPT", "PREREQUISITE"
}

// GenerateCurriculum creates a personalized learning path.
func (c *Coach) GenerateCurriculum(ctx context.Context, userID string, namespaceID string) ([]CurriculumItem, error) {
	ctx, span := observability.Tracer.Start(ctx, "Coach.GenerateCurriculum")
	defer span.End()

	var curriculum []CurriculumItem

	// 1. Fetch Due Nodes from Scheduler
	dueNodes, err := c.Scheduler.GetDueNodes(ctx, namespaceID, 20)
	if err != nil {
		return nil, err
	}

	for _, node := range dueNodes {
		// Only include if relevant to the user (e.g., they have a link to it)
		// For the Personal Knowledge Graph MVP, all nodes in the user's namespace are relevant.
		curriculum = append(curriculum, CurriculumItem{
			NodeID:   node.ID,
			Content:  node.Content,
			Priority: 1.0,
			Type:     "REVIEW",
		})
	}

	return curriculum, nil
}

// ExtractedPattern represents a structured pattern found by the LLM.
type ExtractedPattern struct {
	Content     string         `json:"content"`
	EntityClass string         `json:"entity_class"`
	Relation    string         `json:"relation"` // STRUGGLES_WITH, MASTERED_BY, etc.
	Metadata    map[string]any `json:"metadata"`
}

// ExtractPatterns analyzes content to identify knowledge units and patterns.
func (c *Coach) ExtractPatterns(ctx context.Context, userID string, namespaceID string, sourceContent string) ([]string, error) {
	ctx, span := observability.Tracer.Start(ctx, "Coach.ExtractPatterns")
	defer span.End()

	prompt := fmt.Sprintf(`
		Analyze the following content and extract atomic knowledge patterns or units (e.g., Chess tactics, Grammar rules, Code patterns).
		Format each pattern as a JSON object with: "content", "entity_class" (CONCEPT or PATTERN), "relation" (either 'STRUGGLES_WITH' if the user failed it, or 'MASTERED_BY' if they succeeded), and "metadata".
		Return a JSON array of these objects.

		CONTENT:
		%s
	`, sourceContent)

	response, err := c.LLM.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Basic JSON extraction from LLM response
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON array found in LLM response")
	}

	var patterns []ExtractedPattern
	if err := json.Unmarshal([]byte(response[start:end+1]), &patterns); err != nil {
		return nil, fmt.Errorf("failed to parse patterns: %w", err)
	}

	var nodeIDs []string
	for _, p := range patterns {
		node := &cortex.Node{
			ID:          strings.ReplaceAll(strings.ToLower(p.Content), " ", "_"), // Simple ID generation
			Content:     p.Content,
			EntityClass: p.EntityClass,
			Author:      userID,
			SourceType:  "COACH_EXTRACTION",
			NamespaceID: namespaceID,
			Metadata:    p.Metadata,
		}

		if err := c.Cortex.PutNode(ctx, node); err != nil {
			continue
		}

		// Create Link
		link := &cortex.Link{
			SourceID:     userID,
			TargetID:     node.ID,
			RelationType: p.Relation,
			Weight:       1.0,
		}
		_ = c.Cortex.LinkNodes(ctx, link)
		nodeIDs = append(nodeIDs, node.ID)
	}

	return nodeIDs, nil
}
