package cortexv1

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Stubs for protobuf generated code to allow local compilation

type UnimplementedCortexServer struct{}

func (UnimplementedCortexServer) Memorize(context.Context, *IngestRequest) (*NodeID, error) {
	return nil, nil
}
func (UnimplementedCortexServer) Recall(context.Context, *RecallRequest) (*CognitionResponse, error) {
	return nil, nil
}
func (UnimplementedCortexServer) QueryFacts(context.Context, *FactRequest) (*FactList, error) {
	return nil, nil
}
func (UnimplementedCortexServer) SemanticSearch(context.Context, *SemanticSearchRequest) (*SemanticSearchResponse, error) {
	return nil, nil
}

type CortexServer interface {
	Memorize(context.Context, *IngestRequest) (*NodeID, error)
	Recall(context.Context, *RecallRequest) (*CognitionResponse, error)
	QueryFacts(context.Context, *FactRequest) (*FactList, error)
	SemanticSearch(context.Context, *SemanticSearchRequest) (*SemanticSearchResponse, error)
}

func RegisterCortexServer(s grpc.ServiceRegistrar, srv CortexServer) {}

type IngestRequest struct {
	Id            string
	Type          string
	Payload       *structpb.Struct
	Confidence    float64
	RequiresHuman bool
	Ttl           *timestamppb.Timestamp
}

type NodeID struct {
	Id string
}

type RecallRequest struct {
	Id    string
	Depth int32
}

type Node struct {
	Id            string
	Type          string
	Payload       *structpb.Struct
	Confidence    float64
	RequiresHuman bool
	Ttl           *timestamppb.Timestamp
	CreatedAt     *timestamppb.Timestamp
	UpdatedAt     *timestamppb.Timestamp
}

type Edge struct {
	SourceId  string
	TargetId  string
	Relation  string
	CreatedAt *timestamppb.Timestamp
}

type CognitionResponse struct {
	Center    *Node
	OutEdges  []*Edge
	InEdges   []*Edge
	Neighbors []*Node
}

type FactRequest struct {
	Query           string
	MetadataFilters map[string]string
}

type FactList struct {
	Facts []*Node
}

// SemanticSearchRequest is the input for hybrid vector+full-text search.
type SemanticSearchRequest struct {
	Query        string
	Limit        int32
	DomainFilter string
	TypeFilter   string
	MinScore     float64
}

// SearchResult wraps a Node with its similarity score and search mode.
type SearchResult struct {
	Node  *Node
	Score float32
	Mode  string // "vector", "fulltext", "hybrid", "jsonb_fallback"
}

// SemanticSearchResponse is the output of a semantic search.
type SemanticSearchResponse struct {
	Results []*SearchResult
	Mode    string
}

// Client stubs
type CortexClient interface {
	Memorize(ctx context.Context, in *IngestRequest, opts ...grpc.CallOption) (*NodeID, error)
	Recall(ctx context.Context, in *RecallRequest, opts ...grpc.CallOption) (*CognitionResponse, error)
	QueryFacts(ctx context.Context, in *FactRequest, opts ...grpc.CallOption) (*FactList, error)
	SemanticSearch(ctx context.Context, in *SemanticSearchRequest, opts ...grpc.CallOption) (*SemanticSearchResponse, error)
}

type cortexClient struct {
	cc grpc.ClientConnInterface
}

func NewCortexClient(cc grpc.ClientConnInterface) CortexClient {
	return &cortexClient{cc}
}

func (c *cortexClient) Memorize(ctx context.Context, in *IngestRequest, opts ...grpc.CallOption) (*NodeID, error) {
	return nil, nil
}

func (c *cortexClient) Recall(ctx context.Context, in *RecallRequest, opts ...grpc.CallOption) (*CognitionResponse, error) {
	return nil, nil
}

func (c *cortexClient) QueryFacts(ctx context.Context, in *FactRequest, opts ...grpc.CallOption) (*FactList, error) {
	return nil, nil
}

func (c *cortexClient) SemanticSearch(ctx context.Context, in *SemanticSearchRequest, opts ...grpc.CallOption) (*SemanticSearchResponse, error) {
	return nil, nil
}
