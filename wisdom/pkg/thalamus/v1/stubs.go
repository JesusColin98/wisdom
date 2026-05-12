package thalamusv1

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Stubs for protobuf generated code to allow local compilation

type UnimplementedThalamusServer struct{}

func (UnimplementedThalamusServer) HydrateContext(context.Context, *QueryRequest) (*ContextPayload, error) {
	return nil, nil
}
func (UnimplementedThalamusServer) AuditThought(context.Context, *ThoughtTrace) (*AuditResponse, error) {
	return nil, nil
}

type ThalamusServer interface {
	HydrateContext(context.Context, *QueryRequest) (*ContextPayload, error)
	AuditThought(context.Context, *ThoughtTrace) (*AuditResponse, error)
}

func RegisterThalamusServer(s grpc.ServiceRegistrar, srv ThalamusServer) {}

type QueryRequest struct {
	Query       string
	TokenBudget int32
	Tags        []string
}

type ContextPayload struct {
	FormattedMarkdown string
	EstimatedTokens   int32
	Sources           []string
}

type ThoughtTrace struct {
	SessionId      string
	Prompt         string
	ChainOfThought string
	FinalResponse  string
	Timestamp      *timestamppb.Timestamp
	Metadata       *structpb.Struct
}

type AuditResponse struct {
	Success bool
	TraceId string
}

// Client stub
type ThalamusClient interface {
	HydrateContext(ctx context.Context, in *QueryRequest, opts ...grpc.CallOption) (*ContextPayload, error)
	AuditThought(ctx context.Context, in *ThoughtTrace, opts ...grpc.CallOption) (*AuditResponse, error)
}

type thalamusClient struct {
	cc grpc.ClientConnInterface
}

func NewThalamusClient(cc grpc.ClientConnInterface) ThalamusClient {
	return &thalamusClient{cc}
}

func (c *thalamusClient) HydrateContext(ctx context.Context, in *QueryRequest, opts ...grpc.CallOption) (*ContextPayload, error) {
	return nil, nil
}

func (c *thalamusClient) AuditThought(ctx context.Context, in *ThoughtTrace, opts ...grpc.CallOption) (*AuditResponse, error) {
	return nil, nil
}
