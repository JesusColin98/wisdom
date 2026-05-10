package cortex_test

import (
        "os"
        "context"
        "testing"

        "github.com/google/wisdom/pkg/cortex"
)
func TestThalamicGating(t *testing.T) {
	ctx := context.Background()
	os.Remove("test_scalability.db")
	defer os.Remove("test_scalability.db")

	db, err := cortex.Open("test_scalability.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Initialize Schema
	schemaBytes, _ := os.ReadFile("schema.sql")
	if len(schemaBytes) > 0 {
		_ = db.InitSchema(ctx, string(schemaBytes))
	}

	// 1. Setup Namespace
	ns := &cortex.Namespace{ID: "ns-test", Name: "Test"}
	db.CreateNamespace(ctx, ns)

	// 2. Add initial node with vector
	node1 := &cortex.Node{ID: "original", Content: "Standard SRE rule", NamespaceID: ns.ID, ConfidenceScore: 0.8}
	db.PutNode(ctx, node1)
	vec1 := []float32{1.0, 0.0, 0.0}
	db.PutVector(ctx, "original", vec1, "v1")

	// 3. Try to add highly similar node (should be gated)
	similarVec := []float32{0.95, 0.05, 0.0} // > 0.92 similarity

	similar, err := db.FindSimilar(ctx, similarVec, 0.92)
	if err != nil {
		t.Fatal(err)
	}
	if similar == nil {
		t.Fatal("Expected to find similar node")
	}

	if err := db.StrengthenSynapse(ctx, similar.ID); err != nil {
		t.Fatal(err)
	}

	// 4. Verify original node confidence increased
	got, _ := db.GetNode(ctx, "original")
	if got.ConfidenceScore < 0.849 || got.ConfidenceScore > 0.851 {
		t.Errorf("Expected confidence 0.85, got %f", got.ConfidenceScore)
	}
}

func TestSubstratePromotion(t *testing.T) {
	ctx := context.Background()
	os.Remove("test_promotion.db")
	os.Remove("test_promotion.db.rpforest")
	defer os.Remove("test_promotion.db")
	defer os.Remove("test_promotion.db.rpforest")

	db, err := cortex.Open("test_promotion.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Initialize Schema
	schemaBytes, _ := os.ReadFile("schema.sql")
	if len(schemaBytes) > 0 {
		_ = db.InitSchema(ctx, string(schemaBytes))
	}

	// Manual promotion
	if err := db.PromoteSubstrate(ctx); err != nil {
		t.Fatal(err)
	}
}
