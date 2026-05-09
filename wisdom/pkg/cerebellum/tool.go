// Package cerebellum handles the execution and action layer of Wisdom.
// It is responsible for managing the lifecycle of tool calls, ensuring thread-safety,
// and providing a resilient environment for autonomous actions.
package cerebellum

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// ... existing code ...

// DynamicTool implements the Tool interface using interpreted Go code.
type DynamicTool struct {
	Definition ToolDefinition
	Source     string
	ExecuteFn  func(context.Context, json.RawMessage) (*Result, error)
}

// Execute runs the dynamic tool's logic.
func (t *DynamicTool) Execute(ctx context.Context, params json.RawMessage) (*Result, error) {
	if t.ExecuteFn == nil {
		return nil, fmt.Errorf("dynamic tool %s not properly synthesized", t.Definition.ID)
	}
	return t.ExecuteFn(ctx, params)
}

// Synthesize creates a DynamicTool from source code.
func Synthesize(def ToolDefinition, source string) (*DynamicTool, error) {
	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, fmt.Errorf("failed to load stdlib: %w", err)
	}

	// Export cerebellum types to the interpreter
	i.Use(interp.Exports{
		"wisdom/wisdom": {
			"Result":         reflect.ValueOf((*Result)(nil)),
			"ToolDefinition": reflect.ValueOf((*ToolDefinition)(nil)),
		},
	})

	if _, err := i.Eval(source); err != nil {
		return nil, fmt.Errorf("failed to evaluate source: %w", err)
	}

	v, err := i.Eval("tool.Execute")
	if err != nil {
		return nil, fmt.Errorf("source must define a 'tool.Execute' function: %w", err)
	}

	fn, ok := v.Interface().(func(context.Context, json.RawMessage) (*Result, error))
	if !ok {
		return nil, fmt.Errorf("invalid signature for tool.Execute. Expected func(context.Context, json.RawMessage) (*Result, error)")
	}

	return &DynamicTool{
		Definition: def,
		Source:     source,
		ExecuteFn:  fn,
	}, nil
}

// Result represents the outcome of a tool execution.
// It is the standard interface for reporting success or failure back to the engine.
type Result struct {
	// Success indicates if the tool performed its primary action without critical failure.
	Success bool `json:"success"`
	// Output contains the tool-specific return data. Must be JSON-serializable.
	Output interface{} `json:"output,omitempty"`
	// Metadata stores execution-specific details like latency, backend IDs, or retry counts.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ToolDefinition provides the metadata and schema required to register and validate a tool.
// Developers should ensure the ID is unique and the Parameters schema is strictly defined.
type ToolDefinition struct {
	// ID is the unique identifier for the tool (e.g., "shell_execute").
	ID string `json:"id"`
	// Name is a human-readable name for the tool.
	Name string `json:"name"`
	// Description explains what the tool does, intended for LLM/Orchestrator context.
	Description string `json:"description"`
	// Parameters is a JSON Schema (json.RawMessage) used to validate incoming arguments.
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

// Tool is the core interface for all Wisdom actions.
// To implement a new tool:
// 1. Create a struct that implements this interface.
// 2. Register it in the Cerebellum Registry.
// 3. (Optional) Implement the CircuitBreaker interface for failure tracking.
type Tool interface {
	// Execute performs the action. It MUST respect the context for cancellation/timeouts.
	// params is guaranteed to be validated against the ToolDefinition.Parameters schema
	// if called via the standard Thalamic/Cerebellum pipeline.
	Execute(ctx context.Context, params json.RawMessage) (*Result, error)
}
