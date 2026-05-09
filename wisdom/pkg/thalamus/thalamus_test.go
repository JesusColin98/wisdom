package thalamus_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/thalamus"
)

func TestThalamusGate(t *testing.T) {
	v := thalamus.NewValidator()
	v.RegisterSchema("test_method", `{"type": "object", "required": ["id"]}`)

	cache, _ := thalamus.NewCache(10)
	gate := thalamus.NewGate(v, cache)

	sessionID := "sess-123"
	session := thalamus.NewSession(sessionID, "jesuscolin")
	cache.PutSession(session)

	// 1. Valid call
	if err := gate.Admit(sessionID, "test_method", `{"id": "abc"}`); err != nil {
		t.Errorf("expected valid call to pass, got: %v", err)
	}

	// 2. Reactive Gating: Block method
	session.Flags["block_test_method"] = "true"
	if err := gate.Admit(sessionID, "test_method", `{"id": "abc"}`); err == nil {
		t.Error("expected blocked method to fail")
	}

	// 3. Schema failure
	session.Flags["block_test_method"] = "false"
	if err := gate.Admit(sessionID, "test_method", `{}`); err == nil {
		t.Error("expected schema failure to be caught at gate")
	}
}

func TestThalamusOrchestration(t *testing.T) {
	ctx := context.Background()

	// Setup Cortex
	tmpDir, _ := os.MkdirTemp("", "cortex")
	defer os.RemoveAll(tmpDir)
	dbPath := filepath.Join(tmpDir, "cortex.db")
	cx, _ := cortex.Open(dbPath)
	defer cx.Close()

	schema, err := os.ReadFile("../cortex/schema.sql")
	if err != nil {
		t.Fatalf("failed to read schema: %v", err)
	}
	if err := cx.InitSchema(ctx, string(schema)); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Add some nodes
	if err := cx.CreateNamespace(ctx, &cortex.Namespace{ID: "ns-1", Name: "Engineering"}); err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}

	if err := cx.PutNode(ctx, &cortex.Node{ID: "node-1", Content: "High Signal Fact", NamespaceID: "ns-1", Author: "test", SourceType: "MANUAL"}); err != nil {
		t.Fatalf("failed to put node-1: %v", err)
	}
	if err := cx.PutNode(ctx, &cortex.Node{ID: "node-2", Content: "Related Context", NamespaceID: "ns-1", Author: "test", SourceType: "MANUAL"}); err != nil {
		t.Fatalf("failed to put node-2: %v", err)
	}
	if err := cx.LinkNodes(ctx, &cortex.Link{SourceID: "node-1", TargetID: "node-2", RelationType: "related", Weight: 1.0}); err != nil {
		t.Fatalf("failed to link nodes: %v", err)
	}

	// Verify node 1 exists
	n1, _ := cx.GetNode(ctx, "node-1")
	if n1 == nil {
		t.Fatalf("node-1 not found after PutNode")
	}
	t.Logf("Verified node-1: %s", n1.Content)

	// Verify propagation directly
	scores, err := cx.Propagate(ctx, []string{"node-1"}, 0.85, 1)
	if err != nil {
		t.Fatalf("direct propagation failed: %v", err)
	}
	t.Logf("Direct propagation scores: %v", scores)

	// Setup Thalamus
	cache, _ := thalamus.NewCache(10)
	sessionID := "sess-123"
	cache.PutSession(thalamus.NewSession(sessionID, "user"))

	orch := thalamus.NewOrchestrator(cx, cache)

	// Test Aggregation with Token Budgeting
	context, err := orch.Recall(ctx, sessionID, []string{"node-1"}, 150) // Budget allows only 1 node (~100 tokens)
	if err != nil {
		t.Fatalf("aggregation failed: %v", err)
	}

	t.Logf("Aggregated wisdom count: %d", len(context.Wisdom))
	for i, w := range context.Wisdom {
		t.Logf("  Node %d: %s", i, w)
	}

	if len(context.Wisdom) != 1 {
		t.Errorf("expected 1 node within budget, got %d", len(context.Wisdom))
	}

	// Test Aggregation with higher budget
	context, _ = orch.Recall(ctx, sessionID, []string{"node-1"}, 300)
	if len(context.Wisdom) != 2 {
		t.Errorf("expected 2 nodes within budget, got %d", len(context.Wisdom))
	}
}
