package thalamus

import (
	"bytes"
	"context"
	"strings"
	"text/template"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
	pb "github.com/google/wisdom/pkg/thalamus/v1"
)

// Server implements the gRPC Thalamus service.
type Server struct {
	pb.UnimplementedThalamusServer
	cortexClient cortexv1.CortexClient
}

// NewServer creates a new Thalamus Server connected to Cortex.
func NewServer(cortexClient cortexv1.CortexClient) *Server {
	return &Server{
		cortexClient: cortexClient,
	}
}

// HydrateContext fetches relevant data from Cortex and formats it deterministically.
func (s *Server) HydrateContext(ctx context.Context, req *pb.QueryRequest) (*pb.ContextPayload, error) {
	if req.Query == "" && len(req.Tags) == 0 {
		return nil, status.Error(codes.InvalidArgument, "either query or tags must be provided")
	}

	// 1. Fetch relevant nodes from Cortex
	// Note: Currently just using the tags as metadata filters for simplicity
	filters := make(map[string]string)
	for _, tag := range req.Tags {
		filters["tag"] = tag // Assuming 'tag' is a metadata field in Cortex
	}

	cortexReq := &cortexv1.FactRequest{
		Query:           req.Query,
		MetadataFilters: filters,
	}

	factList, err := s.cortexClient.QueryFacts(ctx, cortexReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query cortex: %v", err)
	}

	// 2. Format the response deterministically (Maximize Token-to-Signal)
	var sb strings.Builder
	var sources []string
	
	// Basic Template for deterministic formatting
	const tpl = `
--- Fact: {{.ID}} ---
{{.Content}}
`
	parsedTpl, err := template.New("fact").Parse(tpl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse template: %v", err)
	}

	for _, fact := range factList.Facts {
		sources = append(sources, fact.Id)

		// We extract content from the generic payload
		var contentStr string
		if fact.Payload != nil {
			if contentVal, ok := fact.Payload.Fields["content"]; ok {
				contentStr = contentVal.GetStringValue()
			}
		}

		data := struct {
			ID      string
			Content string
		}{
			ID:      fact.Id,
			Content: contentStr,
		}

		var buf bytes.Buffer
		if err := parsedTpl.Execute(&buf, data); err == nil {
			sb.WriteString(buf.String())
		}
	}

	formattedMarkdown := sb.String()
	
	// Rough token estimation (very naive, usually ~4 chars per token)
	estimatedTokens := int32(len(formattedMarkdown) / 4)

	return &pb.ContextPayload{
		FormattedMarkdown: formattedMarkdown,
		EstimatedTokens:   estimatedTokens,
		Sources:           sources,
	}, nil
}

// AuditThought receives LLM traces and saves them to Cortex as Signals.
func (s *Server) AuditThought(ctx context.Context, req *pb.ThoughtTrace) (*pb.AuditResponse, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	// Map the trace to a Signal Payload for Cortex
	// Construct the protobuf Struct manually or copy the existing metadata
	payload := req.Metadata
	if payload == nil {
		// In a real implementation, you'd construct a structpb.Struct here
		// For the stub/build pass, we skip the deep mapping
	}
	
	cortexReq := &cortexv1.IngestRequest{
		Type:          "Signal",
		Payload:       payload,
		Confidence:    1.0,
		RequiresHuman: false,
		Ttl:           nil, // Let Cortex use its default Signal TTL or keep it null
	}

	nodeIDResp, err := s.cortexClient.Memorize(ctx, cortexReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to memorize thought trace: %v", err)
	}

	return &pb.AuditResponse{
		Success: true,
		TraceId: nodeIDResp.Id,
	}, nil
}
