// Package entityv1 contains the gRPC stubs for the Wisdom-Entity-Dictionary service.
// These are hand-written stubs, replaced by protoc-generated code once
// scripts/gen_proto.sh is executed.
package entityv1

import (
	"context"

	"google.golang.org/grpc"
)

// ─── Unimplemented Server ─────────────────────────────────────────────────────

type UnimplementedEntityDictionaryServer struct{}

func (UnimplementedEntityDictionaryServer) ResolveEntity(context.Context, *EntityRequest) (*EntityProfile, error) {
	return nil, nil
}
func (UnimplementedEntityDictionaryServer) TagContent(context.Context, *ContentRequest) (*TaggedContent, error) {
	return nil, nil
}
func (UnimplementedEntityDictionaryServer) RegisterEntity(context.Context, *RegisterRequest) (*EntityProfile, error) {
	return nil, nil
}
func (UnimplementedEntityDictionaryServer) GetRelationship(context.Context, *RelationshipRequest) (*RelationshipList, error) {
	return nil, nil
}

// ─── Server Interface ─────────────────────────────────────────────────────────

type EntityDictionaryServer interface {
	ResolveEntity(context.Context, *EntityRequest) (*EntityProfile, error)
	TagContent(context.Context, *ContentRequest) (*TaggedContent, error)
	RegisterEntity(context.Context, *RegisterRequest) (*EntityProfile, error)
	GetRelationship(context.Context, *RelationshipRequest) (*RelationshipList, error)
}

func RegisterEntityDictionaryServer(s grpc.ServiceRegistrar, srv EntityDictionaryServer) {}

// ─── Enums ────────────────────────────────────────────────────────────────────

type EntityScope int32

const (
	EntityScope_PRIVATE EntityScope = 0
	EntityScope_GLOBAL  EntityScope = 1
)

// ─── Messages ─────────────────────────────────────────────────────────────────

type EntityRequest struct {
	SymbolText string      `json:"symbol_text"`
	UserId     string      `json:"user_id"`
	Scope      EntityScope `json:"scope"`
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EntityProfile struct {
	Id                  string       `json:"id"`
	Symbol              string       `json:"symbol"`
	EntityType          string       `json:"entity_type"`
	CanonicalName       string       `json:"canonical_name"`
	Attributes          []*Attribute `json:"attributes"`
	Scope               EntityScope  `json:"scope"`
	RequiresResolution  bool         `json:"requires_resolution"`
}

type ContentRequest struct {
	Content string `json:"content"`
	UserId  string `json:"user_id"`
	Context string `json:"context"`
}

type Tag struct {
	Symbol     string `json:"symbol"`
	EntityId   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	StartPos   int32  `json:"start_pos"`
	EndPos     int32  `json:"end_pos"`
}

type TaggedContent struct {
	OriginalContent string `json:"original_content"`
	Tags            []*Tag `json:"tags"`
}

type RegisterRequest struct {
	Symbol        string       `json:"symbol"`
	EntityType    string       `json:"entity_type"`
	CanonicalName string       `json:"canonical_name"`
	Attributes    []*Attribute `json:"attributes"`
	UserId        string       `json:"user_id"`
	Scope         EntityScope  `json:"scope"`
}

type RelationshipRequest struct {
	FromEntityId string `json:"from_entity_id"`
	ToEntityId   string `json:"to_entity_id"`
}

type Relationship struct {
	FromEntityId string  `json:"from_entity_id"`
	ToEntityId   string  `json:"to_entity_id"`
	RelationType string  `json:"relation_type"`
	Confidence   float64 `json:"confidence"`
}

type RelationshipList struct {
	Relationships []*Relationship `json:"relationships"`
}

// ─── Client Interface ─────────────────────────────────────────────────────────

type EntityDictionaryClient interface {
	ResolveEntity(ctx context.Context, in *EntityRequest, opts ...grpc.CallOption) (*EntityProfile, error)
	TagContent(ctx context.Context, in *ContentRequest, opts ...grpc.CallOption) (*TaggedContent, error)
	RegisterEntity(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*EntityProfile, error)
	GetRelationship(ctx context.Context, in *RelationshipRequest, opts ...grpc.CallOption) (*RelationshipList, error)
}

type entityClient struct{ cc grpc.ClientConnInterface }

func NewEntityDictionaryClient(cc grpc.ClientConnInterface) EntityDictionaryClient {
	return &entityClient{cc}
}
func (c *entityClient) ResolveEntity(ctx context.Context, in *EntityRequest, opts ...grpc.CallOption) (*EntityProfile, error) {
	return nil, nil
}
func (c *entityClient) TagContent(ctx context.Context, in *ContentRequest, opts ...grpc.CallOption) (*TaggedContent, error) {
	return nil, nil
}
func (c *entityClient) RegisterEntity(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*EntityProfile, error) {
	return nil, nil
}
func (c *entityClient) GetRelationship(ctx context.Context, in *RelationshipRequest, opts ...grpc.CallOption) (*RelationshipList, error) {
	return nil, nil
}
