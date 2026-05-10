package thalamus

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// InquirerService implements the Neural-Socratic Loop.
// It identifies knowledge gaps in retrieved context and suggests expansion seeds.
type InquirerService struct {
	LLM    cerebellum.LLMProvider
	Cortex *cortex.Cortex
}

// NewInquirerService creates a new InquirerService.
func NewInquirerService(llm cerebellum.LLMProvider, cx *cortex.Cortex) *InquirerService {
	return &InquirerService{
		LLM:    llm,
		Cortex: cx,
	}
}

// AnalyzeGaps checks if the retrieved nodes are sufficient to answer the query.
// If not, it returns a list of suggested search terms or entity IDs to expand the search.
func (s *InquirerService) AnalyzeGaps(ctx context.Context, query string, retrieved []cortex.ScoredNode) ([]string, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Inquirer.AnalyzeGaps")
	defer span.End()

	if len(retrieved) == 0 {
		return []string{query}, nil // No context at all, start with query
	}

	// 1. Prepare prompt for "The Inquirer"
	var contextBuilder strings.Builder
	for i, sn := range retrieved {
		contextBuilder.WriteString(fmt.Sprintf("[%d] ID: %s | Content: %s\n", i, sn.ID, sn.Content))
	}

	prompt := fmt.Sprintf(`You are the "Inquirer" module of a Knowledge Runtime.
Your task is to analyze the provided context and determine if it's sufficient to answer the user query accurately.
If there are missing links, concepts, or data points, suggest 1-3 specific search terms or entity IDs.

User Query: %s

Retrieved Context:
%s

Instructions:
- If context is sufficient, respond with "SUFFICIENT".
- If not, list the missing keywords or node IDs, separated by commas.
- Do NOT provide a full answer, only suggest retrieval expansions.

Suggestions:`, query, contextBuilder.String())

	// 2. Query LLM
	span.AddEvent("Querying LLM for Gap Analysis")
	response, err := s.LLM.Complete(ctx, prompt)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("inquirer llm failure: %w", err)
	}

	trimmed := strings.TrimSpace(response)
	span.SetAttributes(
		observability.AttrString("inquirer.query", query),
		observability.AttrString("inquirer.response", trimmed),
	)

	if trimmed == "SUFFICIENT" {
		span.AddEvent("Context deemed sufficient")
		return nil, nil
	}

	// 3. Parse expansion seeds
	span.AddEvent("Parsing expansion seeds")
	parts := strings.Split(trimmed, ",")
	var seeds []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			seeds = append(seeds, s)
		}
	}

	span.SetAttributes(observability.AttrInt("inquirer.seed_count", len(seeds)))
	return seeds, nil
}

// ExpandSeeds resolves suggested terms into actual node IDs from Cortex.
func (s *InquirerService) ExpandSeeds(ctx context.Context, suggestions []string) ([]string, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Inquirer.ExpandSeeds")
	defer span.End()

	var expanded []string
	for _, term := range suggestions {
		// Try to resolve as pointer first
		id, err := s.Cortex.ResolvePointer(ctx, term)
		if err == nil {
			expanded = append(expanded, id)
			continue
		}

		// Otherwise, do a quick vector search to find the closest starting node
		nodes, err := s.Cortex.SearchNodes(ctx, term)
		if err == nil && len(nodes) > 0 {
			expanded = append(expanded, nodes[0].ID)
		}
	}
	span.SetAttributes(observability.AttrInt("inquirer.expanded_count", len(expanded)))
	return expanded, nil
}
