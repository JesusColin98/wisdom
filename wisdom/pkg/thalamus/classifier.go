package thalamus

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/observability"
)

// Intent represents the detected cognitive category of a query.
type Intent string

const (
	IntentCode       Intent = "CODE"      // Lexical search
	IntentRelational Intent = "RELATIONAL" // Mesh/Graph crawl
	IntentHierarchy  Intent = "HIERARCHY"  // Tree traversal (Org charts, Sam Morgan)
	IntentStar       Intent = "STAR"       // Entity-centric (Risk, Incident analysis)
	IntentKG         Intent = "KG"         // Knowledge Graph (Complex factual)
	IntentGeneral    Intent = "GENERAL"    // Flat RAG
)

// Strictness defines the grounding requirements for a query.
type Strictness string

const (
	StrictRule     Strictness = "STRICT"  // Must be grounded in SCG Trie (Policies, Specs)
	DynamicContext Strictness = "DYNAMIC" // Allows new conceptual combinations (Brainstorming)
)

// IntentResult combines intent and strictness.
type IntentResult struct {
	Intent     Intent
	Strictness Strictness
	Confidence float64
}

// IntentClassifierV2 automatically assigns retrieval patterns and strictness.
type IntentClassifierV2 struct {
	LLM cerebellum.LLMProvider
}

// NewIntentClassifierV2 creates a new classifier.
func NewIntentClassifierV2(llm cerebellum.LLMProvider) *IntentClassifierV2 {
	return &IntentClassifierV2{LLM: llm}
}

// Classify determines the best retrieval pattern and strictness for the query.
func (c *IntentClassifierV2) Classify(ctx context.Context, query string) (IntentResult, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.IntentClassifier.Classify")
	defer span.End()

	prompt := fmt.Sprintf(`Analyze the user query and classify it into:
- INTENT: CODE (files/symbols), RELATIONAL (mesh/causal), HIERARCHY (tree/org), STAR (entity-centric/risk), KG (complex factual), GENERAL.
- STRICTNESS: STRICT (facts/rules) or DYNAMIC (creative/brainstorming).

Query: "%s"

Respond ONLY with: INTENT STRICTNESS CONFIDENCE`, query)

	response, err := c.LLM.Complete(ctx, prompt)
	if err != nil {
		return IntentResult{Intent: IntentGeneral, Strictness: DynamicContext, Confidence: 0.0}, err
	}

	parts := strings.Fields(strings.TrimSpace(response))
	res := IntentResult{Intent: IntentGeneral, Strictness: DynamicContext, Confidence: 0.5}

	if len(parts) >= 1 {
		res.Intent = Intent(strings.ToUpper(parts[0]))
	}
	if len(parts) >= 2 {
		res.Strictness = Strictness(strings.ToUpper(parts[1]))
	}
	if len(parts) >= 3 {
		fmt.Sscanf(parts[2], "%f", &res.Confidence)
	}

	return res, nil
}
