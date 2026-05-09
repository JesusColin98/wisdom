package cerebellum

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNeurogenesis(t *testing.T) {
	def := ToolDefinition{
		ID:   "dynamic_adder",
		Name: "Dynamic Adder",
		Description: "Adds two numbers dynamically",
	}

	source := `
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"wisdom"
)

func Execute(ctx context.Context, params json.RawMessage) (*wisdom.Result, error) {
	var p struct {
		A int ` + "`" + `json:"a"` + "`" + `
		B int ` + "`" + `json:"b"` + "`" + `
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	return &wisdom.Result{
		Success: true,
		Output:  fmt.Sprintf("%d", p.A + p.B),
	}, nil
}
`

	dynamicTool, err := Synthesize(def, source)
	if err != nil {
		t.Fatalf("failed to synthesize tool: %v", err)
	}

	params := json.RawMessage(`{"a": 10, "b": 20}`)
	result, err := dynamicTool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("failed to execute dynamic tool: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}

	if result.Output != "30" {
		t.Errorf("expected 30, got %v", result.Output)
	}
}
