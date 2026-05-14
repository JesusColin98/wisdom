// Package entity implements the Wisdom-Entity-Dictionary gRPC service.
// It is the ontology service responsible for identifying and contextualizing
// entities (@People, #Topics, [[Concepts]]) within the knowledge substrate.
package entity

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
	pb "github.com/google/wisdom/pkg/entity/v1"
)

// entityPattern matches @Person, #Topic, and [[Wikilinks]] in raw text.
var entityPattern = regexp.MustCompile(`(@\w+|#\w+|\[\[([^\]]+)\]\])`)

// Server implements the EntityDictionaryServer gRPC interface.
type Server struct {
	pb.UnimplementedEntityDictionaryServer
	cortex cortexv1.CortexClient
}

// NewServer creates a new Entity Dictionary server.
func NewServer(cortex cortexv1.CortexClient) *Server {
	return &Server{cortex: cortex}
}

// ResolveEntity looks up a symbolic reference in the Cortex entity registry.
func (s *Server) ResolveEntity(ctx context.Context, req *pb.EntityRequest) (*pb.EntityProfile, error) {
	if req.SymbolText == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol_text is required")
	}

	symbol := strings.TrimSpace(req.SymbolText)

	// TODO: Query Cortex for a node of type "Entity" with matching symbol.
	// If multiple results found for the same symbol across different users,
	// return requires_resolution = true (ambiguity signal to Thalamus).
	_ = symbol

	// Placeholder: return a stub profile.
	return &pb.EntityProfile{
		Symbol:              req.SymbolText,
		EntityType:          inferEntityType(req.SymbolText),
		CanonicalName:       strings.TrimPrefix(strings.TrimPrefix(req.SymbolText, "@"), "#"),
		RequiresResolution:  false,
	}, nil
}

// TagContent scans raw Markdown content and tags all recognized entities.
func (s *Server) TagContent(ctx context.Context, req *pb.ContentRequest) (*pb.TaggedContent, error) {
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	matches := entityPattern.FindAllStringIndex(req.Content, -1)
	tags := make([]*pb.Tag, 0, len(matches))

	for _, m := range matches {
		symbol := req.Content[m[0]:m[1]]
		// Resolve each symbol to get the entity type.
		profile, err := s.ResolveEntity(ctx, &pb.EntityRequest{
			SymbolText: symbol,
			UserId:     req.UserId,
		})
		if err != nil {
			continue
		}

		tags = append(tags, &pb.Tag{
			Symbol:     symbol,
			EntityId:   fmt.Sprintf("entity:%s", profile.CanonicalName),
			EntityType: profile.EntityType,
			StartPos:   int32(m[0]),
			EndPos:     int32(m[1]),
		})
	}

	return &pb.TaggedContent{
		OriginalContent: req.Content,
		Tags:            tags,
	}, nil
}

// RegisterEntity adds a new entity to the Cortex registry.
func (s *Server) RegisterEntity(ctx context.Context, req *pb.RegisterRequest) (*pb.EntityProfile, error) {
	if req.Symbol == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol and user_id are required")
	}

	// TODO: Call cortex.Memorize with type="Entity" and the registration payload.
	return &pb.EntityProfile{
		Symbol:        req.Symbol,
		EntityType:    req.EntityType,
		CanonicalName: req.CanonicalName,
		Attributes:    req.Attributes,
		Scope:         req.Scope,
	}, nil
}

// GetRelationship finds the relationship edges between two entities in Cortex.
func (s *Server) GetRelationship(ctx context.Context, req *pb.RelationshipRequest) (*pb.RelationshipList, error) {
	if req.FromEntityId == "" || req.ToEntityId == "" {
		return nil, status.Error(codes.InvalidArgument, "from_entity_id and to_entity_id are required")
	}

	// TODO: Call cortex.Recall with depth=2 to traverse edges between the two nodes.
	return &pb.RelationshipList{Relationships: []*pb.Relationship{}}, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// inferEntityType determines entity type from its symbolic prefix.
func inferEntityType(symbol string) string {
	switch {
	case strings.HasPrefix(symbol, "@"):
		return "PERSON"
	case strings.HasPrefix(symbol, "#"):
		return "TOPIC"
	case strings.HasPrefix(symbol, "[["):
		return "CONCEPT"
	default:
		return "UNKNOWN"
	}
}
