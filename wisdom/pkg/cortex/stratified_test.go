package cortex_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/wisdom/pkg/cortex"
)

func TestStratifiedStorageAndPruning(t *testing.T) {
	ctx := context.Background()
	dbPath := "test_stratified.db"
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	db, err := cortex.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer db.Close()

	// Initialize Schema
	schemaBytes, _ := os.ReadFile("schema.sql")
	if len(schemaBytes) > 0 {
		_ = db.InitSchema(ctx, string(schemaBytes))
	}

	ns := &cortex.Namespace{ID: "ns-strat", Name: "stratified"}
	db.CreateNamespace(ctx, ns)

	// 1. Test Put and Get with new fields
	node := &cortex.Node{
		ID:              "node-strat",
		Content:         "Stratified logic",
		Author:          "tester",
		SourceType:      "MANUAL",
		NamespaceID:     ns.ID,
		ConfidenceScore: 0.95,
		ImpactScore:     0.88,
		ExternalLinks:   []cortex.ExternalLink{{URI: "https://google.com", Type: "URL"}, {URI: "b/123", Type: "BUG"}},
		}

		if err := db.PutNode(ctx, node); err != nil {
		t.Fatalf("PutNode failed: %v", err)
		}

		got, err := db.GetNode(ctx, "node-strat")
		if err != nil {
		t.Fatalf("GetNode failed: %v", err)
		}

		if got.ImpactScore != 0.88 {
		t.Errorf("expected ImpactScore 0.88, got %f", got.ImpactScore)
		}
		if len(got.ExternalLinks) != 2 || got.ExternalLinks[1].URI != "b/123" {
		t.Errorf("expected ExternalLinks URI b/123, got %v", got.ExternalLinks)
		}
	// 2. Test Synaptic Pruning (Decay)
	// Add a low-impact node
	nodeLow := &cortex.Node{
		ID:              "node-low",
		Content:         "Noise",
		Author:          "tester",
		SourceType:      "MANUAL",
		NamespaceID:     ns.ID,
		ConfidenceScore: 0.1,
		ImpactScore:     0.01,
	}
	db.PutNode(ctx, nodeLow)

	// Manually set confidence score to a very low value to test deletion
	_, _ = db.DB().Exec("UPDATE nodes SET confidence_score = 0.01 WHERE id = 'node-low'")

	// Prune with threshold 0.05
	pruned, err := db.PruneNodes(ctx, 0.0, 0.05)
	if err != nil {
		t.Fatalf("PruneNodes failed: %v", err)
	}

	if pruned < 1 {
		t.Errorf("expected at least 1 node pruned, got %d", pruned)
	}

	dead, _ := db.GetNode(ctx, "node-low")
	if dead != nil {
		t.Error("expected node-low to be pruned")
	}

	survivor, _ := db.GetNode(ctx, "node-strat")
	if survivor == nil {
		t.Fatal("expected node-strat to survive")
	}
}
