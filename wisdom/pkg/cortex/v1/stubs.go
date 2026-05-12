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

type CortexServer interface {
	Memorize(context.Context, *IngestRequest) (*NodeID, error)
	Recall(context.Context, *RecallRequest) (*CognitionResponse, error)
	QueryFacts(context.Context, *FactRequest) (*FactList, error)
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

// Client stubs
type CortexClient interface {
	Memorize(ctx context.Context, in *IngestRequest, opts ...grpc.CallOption) (*NodeID, error)
	Recall(ctx context.Context, in *RecallRequest, opts ...grpc.CallOption) (*CognitionResponse, error)
	QueryFacts(ctx context.Context, in *FactRequest, opts ...grpc.CallOption) (*FactList, error)
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
