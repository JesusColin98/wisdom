package thalamus

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Chat handles the conversational logic of Wisdom, acting as the primary
// interface for grounded user interactions.
type Chat struct {
	// Storage provides access to the semantic memory (Cortex).
	Storage *cortex.Cortex
	// LLM is the provider for Large Language Model completions.
	LLM cerebellum.LLMProvider
	// Hippocampus manages transient session memory.
	Hippocampus *Hippocampus
	// Orchestrator handles deep context retrieval and refinement.
	Orchestrator *Orchestrator
}

// Reason processes a query and returns a cognitive map of how nodes relate to it.
func (c *Chat) Reason(ctx context.Context, query string) (string, []cortex.ScoredNode, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Reason")
	defer span.End()

	// 1. Generate embedding for query
	embedding, err := c.LLM.Embed(ctx, query)
	if err != nil {
		observability.Logger.Warn("Reason: failed to generate embedding, falling back to keyword search", "error", err)
	}

	// 2. Use Hybrid Search to find the most relevant nodes
	results, err := c.Storage.HybridSearch(ctx, query, embedding, 5)
	if err != nil {
		return "", nil, err
	}

	var mapBuilder strings.Builder
	mapBuilder.WriteString("Cognitive Map for query: " + query + "\n")
	for _, res := range results {
		mapBuilder.WriteString(fmt.Sprintf("- [%s] %s (Confidence: %.2f)\n", res.EntityClass, res.ID, res.ConfidenceScore))
	}

	prompt := fmt.Sprintf(`Analyze these related nodes and explain the logical connection to the query: "%s".
Nodes:
%s`, query, mapBuilder.String())

	explanation, err := c.LLM.Complete(ctx, prompt)
	return explanation, results, err
}

// Validate checks a technical assertion against the Cortex knowledge.
func (c *Chat) Validate(ctx context.Context, assertion string) (bool, string, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Validate")
	defer span.End()

	nodes, err := c.Storage.SearchNodes(ctx, assertion)
	if err != nil {
		return false, "", err
	}

	if len(nodes) == 0 {
		return true, "No contradictory evidence found in Cortex.", nil
	}

	var evidence strings.Builder
	for _, n := range nodes {
		evidence.WriteString(fmt.Sprintf("- %s (Confidence: %.2f)\n", n.Content, n.ConfidenceScore))
	}

	prompt := fmt.Sprintf(`Verify the following assertion: "%s"
Based on this evidence from the Cortex:
%s
Does the evidence support, contradict, or is it neutral to the assertion? Return "SUPPORTED", "CONTRADICTED", or "NEUTRAL" followed by a 1-sentence explanation.`, assertion, evidence.String())

	result, err := c.LLM.Complete(ctx, prompt)
	if err != nil {
		return false, "", err
	}

	isSafe := !strings.Contains(result, "CONTRADICTED")
	return isSafe, result, nil
}

// Ask processes a user query, retrieves relevant context from the Cortex,
// and returns a grounded response from the LLM.
// It returns the generated response, the context nodes used for grounding, and any error.
func (c *Chat) Ask(ctx context.Context, userID string, message string) (string, []string, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Ask",
		trace.WithAttributes(
			attribute.String("user_id", userID),
			attribute.Int("message_length", len(message)),
		))
	defer span.End()

	// 0. Record user query in Hippocampus
	c.Hippocampus.Record(ctx, userID, Interaction{Role: "user", Content: message})

	// 1. Deep Recall via Orchestrator (Phase 2)
	// We use the message as a seed for semantic discovery
	cognition, err := c.Orchestrator.Recall(ctx, userID, message, []string{message}, 2000, 0.5)
	if err != nil {
		return "", nil, fmt.Errorf("recall failed: %w", err)
	}

	// 2. Construct a grounded prompt.
	var contextBuilder strings.Builder
	for _, w := range cognition.Wisdom {
		contextBuilder.WriteString(fmt.Sprintf("%s\n", w))
	}

	prompt := fmt.Sprintf(`You are Wisdom, a Cognitive Memory Substrate.
Retrieved Context:
%s
User Query: %s`, contextBuilder.String(), message)

	// 3. Call the LLM provider for a completion.
	response, err := c.LLM.Complete(ctx, prompt)
	if err != nil {
		return "", nil, fmt.Errorf("llm completion failed: %w", err)
	}

	// 3.1 SCG-Mem Hallucination Guardrail (Phase 2 SOTA)
	grounded, ungrounded := c.Storage.GetTrie().ValidateSentence(response)
	
	// If strictness is STRICT, treat ungrounded terms as high-risk
	threshold := 5
	if cognition.Strictness == StrictRule {
		threshold = 1 // Zero tolerance for ungrounded technical terms in strict mode
	}

	if len(ungrounded) > threshold {
		observability.Logger.Warn("SCG-Mem Hallucination Warning", "strictness", cognition.Strictness, "ungrounded_terms", ungrounded)
		if cognition.Strictness == StrictRule {
			response = fmt.Sprintf("⚠️ [STRICT MODE WARNING] The following terms could not be verified against the knowledge base: %s\n\n%s", strings.Join(ungrounded, ", "), response)
		}
	}
	observability.Logger.Info("SCG-Mem Validation", "strictness", cognition.Strictness, "grounded_words", grounded, "ungrounded_count", len(ungrounded))

	// 4. Record assistant response in Hippocampus
	c.Hippocampus.Record(ctx, userID, Interaction{Role: "assistant", Content: response})

	return response, cognition.Wisdom, nil
}
