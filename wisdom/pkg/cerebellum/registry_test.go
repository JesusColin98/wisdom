package cerebellum

import (
	"context"
	"encoding/json"
	"testing"
)

type mockTool struct{}

func (m *mockTool) Execute(ctx context.Context, params json.RawMessage) (*Result, error) {
	return &Result{Success: true}, nil
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	def := ToolDefinition{
		ID:   "test-tool",
		Name: "Test Tool",
	}
	impl := &mockTool{}

	// Test Register
	if err := registry.Register(def, impl); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Test Duplicate Register
	if err := registry.Register(def, impl); err == nil {
		t.Error("Expected error for duplicate registration, got nil")
	}

	// Test Get
	gotDef, gotImpl, err := registry.Get(ctx, "test-tool")
	if err != nil {
		t.Fatalf("Failed to get tool: %v", err)
	}
	if gotDef.ID != def.ID {
		t.Errorf("Expected tool ID %s, got %s", def.ID, gotDef.ID)
	}
	if gotImpl != impl {
		t.Error("Expected same implementation pointer")
	}

	// Test Get Not Found
	_, _, err = registry.Get(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent tool, got nil")
	}

	// Test List
	list := registry.List()
	if len(list) != 1 {
		t.Errorf("Expected list size 1, got %d", len(list))
	}
	if list[0].ID != def.ID {
		t.Errorf("Expected list[0].ID %s, got %s", def.ID, list[0].ID)
	}
}
