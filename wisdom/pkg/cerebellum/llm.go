package cerebellum

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/wisdom/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LLMProvider defines the interface for interacting with a Large Language Model.
type LLMProvider interface {
	// Complete generates a response from the LLM based on the provided prompt.
	Complete(ctx context.Context, prompt string) (string, error)
	// Embed generates a semantic embedding for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
}
// ResilientLLM wraps an LLMProvider with a CircuitBreaker to handle failures gracefully.
type ResilientLLM struct {
	provider LLMProvider
	circuit  *CircuitBreaker
}

// NewResilientLLM creates a new ResilientLLM wrapper.
func NewResilientLLM(provider LLMProvider, threshold int, timeout time.Duration) *ResilientLLM {
	return &ResilientLLM{
		provider: provider,
		circuit:  NewCircuitBreaker(threshold, timeout),
	}
}

func (r *ResilientLLM) Complete(ctx context.Context, prompt string) (string, error) {
	if !r.circuit.Allow() {
		return "", fmt.Errorf("LLM circuit is open")
	}

	res, err := r.provider.Complete(ctx, prompt)
	if err != nil {
		r.circuit.RecordFailure()
		return "", err
	}

	r.circuit.RecordSuccess()
	return res, nil
}

func (r *ResilientLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	if !r.circuit.Allow() {
		return nil, fmt.Errorf("LLM circuit is open")
	}

	res, err := r.provider.Embed(ctx, text)
	if err != nil {
		r.circuit.RecordFailure()
		return nil, err
	}

	r.circuit.RecordSuccess()
	return res, nil
}

// MockLLM is a mock implementation of LLMProvider for testing purposes.
type MockLLM struct {
	// CannedResponse if set, will be returned by Complete.
	CannedResponse string
	// Echo if true, Complete will echo back the prompt.
	Echo bool
	// FailNext if true, the next call to Complete will return an error.
	FailNext bool
	// CannedEmbedding if set, will be returned by Embed.
	CannedEmbedding []float32
}

// Embed implements the LLMProvider interface.
func (m *MockLLM) Embed(ctx context.Context, text string) ([]float32, error) {
	_, span := observability.Tracer.Start(ctx, "cerebellum.MockLLM.Embed",
		trace.WithAttributes(attribute.Int("text_length", len(text))))
	defer span.End()

	if m.CannedEmbedding != nil {
		return m.CannedEmbedding, nil
	}

	// Default mock embedding (size 3 for testing)
	return []float32{0.1, 0.2, 0.3}, nil
}

// Complete implements the LLMProvider interface.
func (m *MockLLM) Complete(ctx context.Context, prompt string) (string, error) {
	_, span := observability.Tracer.Start(ctx, "cerebellum.MockLLM.Complete",
		trace.WithAttributes(attribute.Int("prompt_length", len(prompt))))
	defer span.End()

	if m.FailNext {
		m.FailNext = false
		return "", fmt.Errorf("mock llm failure")
	}

	if m.CannedResponse != "" {
		return m.CannedResponse, nil
	}

	// Specialized canned response for REM cycle verification
	if strings.Contains(prompt, "Analyze the following SRE session logs") {
		return "Finding 1: Hybrid Search enables conceptual retrieval.\n\nFinding 2: Neurogenesis allows autonomous tool evolution.\n\nFinding 3: Synaptic Layering preserves temporal history.", nil
	}

	if m.Echo {
		return fmt.Sprintf("Echo: %s", prompt), nil
	}

	return "I am a mock LLM response.", nil
}
