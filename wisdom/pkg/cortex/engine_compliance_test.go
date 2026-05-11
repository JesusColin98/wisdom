package cortex

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// EngineComplianceTestSuite defines a set of tests that every StorageEngine must pass.
func EngineComplianceTestSuite(t *testing.T, engine StorageEngine) {
	ctx := context.Background()

	t.Run("NodeStore: Put and Get", func(t *testing.T) {
		node := &Node{
			ID:      "test-node-1",
			Content: "Compliance test content",
			Author:  "Tester",
			Metadata: map[string]any{"key": "value"},
		}
		if err := engine.PutNode(ctx, node); err != nil {
			t.Fatalf("PutNode failed: %v", err)
		}

		got, err := engine.GetNode(ctx, "test-node-1")
		if err != nil {
			t.Fatalf("GetNode failed: %v", err)
		}
		if got == nil {
			t.Fatal("GetNode returned nil")
		}

		if diff := cmp.Diff(node.Content, got.Content); diff != "" {
			t.Errorf("Content mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("VectorStore: Put and Search", func(t *testing.T) {
		nodeID := "vec-node-1"
		embedding := make([]float32, 768)
		embedding[0] = 1.0 // Simple unit vector

		if err := engine.PutVector(ctx, nodeID, embedding, "v1"); err != nil {
			t.Fatalf("PutVector failed: %v", err)
		}

		results, err := engine.SearchVectors(ctx, embedding, 1)
		if err != nil {
			t.Fatalf("SearchVectors failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("SearchVectors returned no results")
		}
		if results[0].ID != nodeID {
			t.Errorf("Expected node %s, got %s", nodeID, results[0].ID)
		}
	})

	t.Run("GraphStore: Link and Neighbors", func(t *testing.T) {
		n1 := &Node{ID: "node-a", Content: "A"}
		n2 := &Node{ID: "node-b", Content: "B"}
		engine.PutNode(ctx, n1)
		engine.PutNode(ctx, n2)

		link := &Link{
			SourceID:     "node-a",
			TargetID:     "node-b",
			RelationType: "TEST_LINK",
			Weight:       0.8,
		}
		if err := engine.LinkNodes(ctx, link); err != nil {
			t.Fatalf("LinkNodes failed: %v", err)
		}

		neighbors, err := engine.GetNeighbors(ctx, "node-a")
		if err != nil {
			t.Fatalf("GetNeighbors failed: %v", err)
		}
		if len(neighbors) != 1 || neighbors[0].ID != "node-b" {
			t.Errorf("Expected neighbor node-b, got %v", neighbors)
		}
	})

	t.Run("SessionStore: Logs", func(t *testing.T) {
		sessionID := "sess-123"
		if err := engine.AddLog(ctx, sessionID, "user", "Hello"); err != nil {
			t.Fatalf("AddLog failed: %v", err)
		}

		logs, err := engine.GetLogs(ctx, sessionID)
		if err != nil {
			t.Fatalf("GetLogs failed: %v", err)
		}
		if len(logs) != 1 || logs[0].Content != "Hello" {
			t.Errorf("Expected log 'Hello', got %v", logs)
		}
	})
}
