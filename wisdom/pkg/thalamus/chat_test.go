package thalamus_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/thalamus"
)

func TestChatAsk(t *testing.T) {
	ctx := context.Background()
	testDB := "test_chat.db"
	os.Remove(testDB)
	defer os.Remove(testDB)

	storage, err := cortex.Open(testDB)
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer storage.Close()

	// Read schema from file
	schemaPath := filepath.Join("..", "cortex", "schema.sql")
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}

	if err := storage.InitSchema(ctx, string(schemaSQL)); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Setup context
	ns := &cortex.Namespace{ID: "ns-1", Name: "test"}
	storage.CreateNamespace(ctx, ns)
	node := &cortex.Node{
		ID:          "node-1",
		Content:     "Wisdom is a Cognitive SRE engine.",
		NamespaceID: ns.ID,
		Author:      "tester",
		SourceType:  "MANUAL",
	}
	storage.PutNode(ctx, node)

	mockLLM := &cerebellum.MockLLM{Echo: true}
	chat := &thalamus.Chat{
		Storage:     storage,
		LLM:         mockLLM,
		Hippocampus: thalamus.NewHippocampus(storage),
	}

	response, nodes, err := chat.Ask(ctx, "user-1", "Wisdom")
	if err != nil {
		t.Fatalf("Ask failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("expected 1 context node, got %d", len(nodes))
	}

	if !strings.Contains(response, "Wisdom is a Cognitive SRE engine.") {
		t.Errorf("response missing context: %s", response)
	}

	if !strings.Contains(response, "Wisdom") {
		t.Errorf("response missing query: %s", response)
	}
}
