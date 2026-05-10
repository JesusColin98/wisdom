package thalamus

import (
	"context"
	"fmt"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// IdentityService manages user personas and expertise within the shared Cortex.
type IdentityService struct {
	Cortex *cortex.Cortex
}

// NewIdentityService creates a new IdentityService.
func NewIdentityService(cx *cortex.Cortex) *IdentityService {
	return &IdentityService{Cortex: cx}
}

// RegisterUser ensures a PERSON node exists for the given UserID.
func (s *IdentityService) RegisterUser(ctx context.Context, userID, name string) error {
	node := &cortex.Node{
		ID:              userID,
		Content:         fmt.Sprintf("User Persona: %s", name),
		EntityClass:     "PERSON",
		Author:          "SYSTEM",
		SourceType:      "IDENTITY",
		NamespaceID:     "ns-users",
		ConfidenceScore: 1.0,
		ImpactScore:     1.0,
		Metadata: map[string]any{
			"display_name": name,
		},
	}
	return s.Cortex.PutNode(ctx, node)
}

// LinkExpertise connects a user to a namespace they are active in.
func (s *IdentityService) LinkExpertise(ctx context.Context, userID, namespaceID string) error {
	link := &cortex.Link{
		SourceID:     userID,
		TargetID:     namespaceID,
		RelationType: "EXPERT_IN",
		Weight:       0.1, // Initial weight, grows with reinforcement
	}
	return s.Cortex.LinkNodes(ctx, link)
}

// GetUserExpertise retrieves the namespaces where the user has high activity.
func (s *IdentityService) GetUserExpertise(ctx context.Context, userID string) ([]string, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Identity.GetUserExpertise")
	defer span.End()

	query := `
		SELECT target_id 
		FROM links 
		WHERE source_id = ? AND relation_type = 'EXPERT_IN'
		ORDER BY weight DESC
	`
	rows, err := s.Cortex.DB().QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expertises []string
	for rows.Next() {
		var nsID string
		if err := rows.Scan(&nsID); err == nil {
			expertises = append(expertises, nsID)
		}
	}
	return expertises, nil
}
