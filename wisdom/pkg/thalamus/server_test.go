package thalamus

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
	pb "github.com/google/wisdom/pkg/thalamus/v1"
)

// mockCortexClient simulates the Cortex gRPC service for testing.
type mockCortexClient struct {
	cortexv1.CortexClient // Embed to satisfy interface for unused methods
}

func (m *mockCortexClient) QueryFacts(ctx context.Context, in *cortexv1.FactRequest, opts ...grpc.CallOption) (*cortexv1.FactList, error) {
	// Create mock payloads
	payload1, _ := structpb.NewStruct(map[string]any{"content": "Chess is a board game played between two players."})
	payload2, _ := structpb.NewStruct(map[string]any{"content": "The Sicilian Defense starts with 1.e4 c5."})

	return &cortexv1.FactList{
		Facts: []*cortexv1.Node{
			{Id: "fact-1", Payload: payload1},
			{Id: "fact-2", Payload: payload2},
		},
	}, nil
}

func (m *mockCortexClient) Memorize(ctx context.Context, in *cortexv1.IngestRequest, opts ...grpc.CallOption) (*cortexv1.NodeID, error) {
	return &cortexv1.NodeID{Id: "new-trace-id-123"}, nil
}

func TestServer_HydrateContext(t *testing.T) {
	mockClient := &mockCortexClient{}
	server := NewServer(mockClient)

	req := &pb.QueryRequest{
		Query: "chess openings",
		Tags:  []string{"chess"},
	}

	resp, err := server.HydrateContext(context.Background(), req)
	if err != nil {
		t.Fatalf("HydrateContext failed: %v", err)
	}

	if !strings.Contains(resp.FormattedMarkdown, "Fact: fact-1") {
		t.Errorf("Expected markdown to contain 'Fact: fact-1'")
	}
	if !strings.Contains(resp.FormattedMarkdown, "Sicilian Defense starts with 1.e4 c5") {
		t.Errorf("Expected markdown to contain payload content")
	}

	if len(resp.Sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(resp.Sources))
	}

	if resp.EstimatedTokens <= 0 {
		t.Errorf("Expected EstimatedTokens to be calculated, got %d", resp.EstimatedTokens)
	}
}

func TestServer_AuditThought(t *testing.T) {
	mockClient := &mockCortexClient{}
	server := NewServer(mockClient)

	metadata, _ := structpb.NewStruct(map[string]any{"step": 1})

	req := &pb.ThoughtTrace{
		SessionId:      "session-1",
		Prompt:         "What is the best opening?",
		ChainOfThought: "Thinking about 1.e4...",
		FinalResponse:  "e4 is solid.",
		Metadata:       metadata,
	}

	resp, err := server.AuditThought(context.Background(), req)
	if err != nil {
		t.Fatalf("AuditThought failed: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success to be true")
	}
	if resp.TraceId != "new-trace-id-123" {
		t.Errorf("Expected TraceId 'new-trace-id-123', got '%s'", resp.TraceId)
	}
}
