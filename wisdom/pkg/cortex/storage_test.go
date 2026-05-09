package cortex_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/wisdom/pkg/cortex"
)

const testDBPath = "test_cortex.db"
const schema = `
CREATE TABLE IF NOT EXISTS namespaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    entity_class TEXT NOT NULL DEFAULT 'OBSERVATION',
    author TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_ref TEXT,
    namespace_id TEXT NOT NULL,
    metadata JSON DEFAULT '{}',
    confidence_score REAL DEFAULT 0.8,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (namespace_id) REFERENCES namespaces(id)
);

CREATE TABLE IF NOT EXISTS links (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relation_type TEXT NOT NULL,
    weight REAL DEFAULT 1.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source_id, target_id, relation_type),
    FOREIGN KEY (source_id) REFERENCES nodes(id),
    FOREIGN KEY (target_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS vectors (
    node_id TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,
    model_version TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS node_history (
    history_id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL,
    content TEXT NOT NULL,
    metadata JSON,
    version_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TRIGGER IF NOT EXISTS archive_node_version
BEFORE UPDATE ON nodes
BEGIN
    INSERT INTO node_history (node_id, content, metadata)
    VALUES (OLD.id, OLD.content, OLD.metadata);
END;
`

func TestCortexTemporalMemory(t *testing.T) {
	ctx := context.Background()
	os.Remove("test_temporal.db")
	defer os.Remove("test_temporal.db")

	db, err := cortex.Open("test_temporal.db")
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx, schema); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	ns := &cortex.Namespace{ID: "ns-test", Name: "testing"}
	db.CreateNamespace(ctx, ns)

	nodeID := "node-v1"
	node := &cortex.Node{
		ID:          nodeID,
		Content:     "Initial logic",
		Author:      "tester",
		SourceType:  "MANUAL",
		NamespaceID: ns.ID,
		Metadata:    map[string]any{"version": 1},
	}

	db.PutNode(ctx, node)

	// Update node to v2 (should trigger archival of v1)
	node.Content = "Improved logic"
	node.Metadata["version"] = 2
	db.PutNode(ctx, node)

	// Update node to v3 (should trigger archival of v2)
	node.Content = "Final logic"
	node.Metadata["version"] = 3
	db.PutNode(ctx, node)

	// Verify current state
	current, _ := db.GetNode(ctx, nodeID)
	if current.Content != "Final logic" {
		t.Errorf("expected Final logic, got %s", current.Content)
	}

	// Verify history
	history, err := db.GetHistory(ctx, nodeID)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("expected 2 historical versions, got %d", len(history))
	}

	// history is ORDER BY timestamp DESC, so first element is v2
	if history[0].Content != "Improved logic" {
		t.Errorf("expected first history item to be v2, got %s", history[0].Content)
	}
	if history[1].Content != "Initial logic" {
		t.Errorf("expected second history item to be v1, got %s", history[1].Content)
	}
}

func TestCortexHybridSearch(t *testing.T) {
	ctx := context.Background()
	os.Remove("test_hybrid.db")
	defer os.Remove("test_hybrid.db")

	db, err := cortex.Open("test_hybrid.db")
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx, schema); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	ns := &cortex.Namespace{ID: "ns-test", Name: "testing"}
	db.CreateNamespace(ctx, ns)

	// Node 1: Matches "SQL" keyword
	node1 := &cortex.Node{ID: "node-1", Content: "GoogleSQL tuning guide", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}
	// Node 2: Matches semantically to "memory" (mocked via vector)
	node2 := &cortex.Node{ID: "node-2", Content: "RAM leak in dremel", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}
	// Node 3: Random node
	node3 := &cortex.Node{ID: "node-3", Content: "Random documentation", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}

	db.PutNode(ctx, node1)
	db.PutNode(ctx, node2)
	db.PutNode(ctx, node3)

	// Mock embeddings: [1, 0] for SQL, [0, 1] for Memory/RAM
	db.PutVector(ctx, "node-1", []float32{1.0, 0.0}, "mock-v1")
	db.PutVector(ctx, "node-2", []float32{0.0, 1.0}, "mock-v1")
	db.PutVector(ctx, "node-3", []float32{0.5, 0.5}, "mock-v1")

	// 1. Vector Search for "Memory" (close to [0, 1])
	vResults, err := db.VectorSearch(ctx, []float32{0.1, 0.9}, 2)
	if err != nil {
		t.Fatalf("VectorSearch failed: %v", err)
	}
	if vResults[0].ID != "node-2" {
		t.Errorf("expected node-2 (RAM leak) to be top vector result, got %s", vResults[0].ID)
	}

	// 2. Hybrid Search for "SQL memory issues"
	// Should find node-1 via keyword "SQL" and node-2 via vector similarity to "memory issues"
	hResults, err := db.HybridSearch(ctx, "SQL", []float32{0.1, 0.9}, 5)
	if err != nil {
		t.Fatalf("HybridSearch failed: %v", err)
	}

	if len(hResults) < 2 {
		t.Fatalf("expected at least 2 hybrid results, got %d", len(hResults))
	}

	// Verify both nodes are present in results
	foundNode1 := false
	foundNode2 := false
	for _, res := range hResults {
		if res.ID == "node-1" {
			foundNode1 = true
		}
		if res.ID == "node-2" {
			foundNode2 = true
		}
	}

	if !foundNode1 || !foundNode2 {
		t.Errorf("expected both node-1 and node-2 in hybrid results. foundNode1=%v, foundNode2=%v", foundNode1, foundNode2)
	}
}

func TestCortexRelationalIntelligence(t *testing.T) {
	ctx := context.Background()
	os.Remove("test_relational.db")
	defer os.Remove("test_relational.db")

	db, err := cortex.Open("test_relational.db")
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx, schema); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	ns := &cortex.Namespace{ID: "ns-1", Name: "test"}
	db.CreateNamespace(ctx, ns)

	// 1. Test Entity Class & Attributes
	svc := &cortex.Node{
		ID:          "bq-engine",
		Content:     "BigQuery Query Engine",
		EntityClass: "SERVICE",
		Author:      "tester",
		NamespaceID: ns.ID,
		Metadata:    map[string]any{"tier": 1},
	}
	db.PutNode(ctx, svc)

	// 2. Test Synonym Pointer
	alias := &cortex.Node{ID: "bq", Content: "Alias for BigQuery", Author: "tester", NamespaceID: ns.ID}
	db.PutNode(ctx, alias)
	db.LinkNodes(ctx, &cortex.Link{SourceID: "bq", TargetID: "bq-engine", RelationType: "SYNONYM_OF", Weight: 1.0})

	resolved, err := db.ResolvePointer(ctx, "bq")
	if err != nil {
		t.Fatalf("ResolvePointer failed: %v", err)
	}
	if resolved != "bq-engine" {
		t.Errorf("expected bq-engine, got %s", resolved)
	}

	// 3. Test Confidence Update
	db.UpdateConfidence(ctx, "bq-engine", -0.2) // User correction/Mistake
	node, _ := db.GetNode(ctx, "bq-engine")
	if node.ConfidenceScore > 0.7 { // Default 0.8 - 0.2 = 0.6
		t.Errorf("expected confidence around 0.6, got %f", node.ConfidenceScore)
	}
}

func TestCortexGraph(t *testing.T) {
	ctx := context.Background()
	os.Remove("test_graph.db")
	defer os.Remove("test_graph.db")

	db, err := cortex.Open("test_graph.db")
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx, schema); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	ns := &cortex.Namespace{ID: "ns-test", Name: "testing"}
	db.CreateNamespace(ctx, ns)

	nodeA := &cortex.Node{ID: "A", Content: "Fact A", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}
	nodeB := &cortex.Node{ID: "B", Content: "Fact B", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}

	db.PutNode(ctx, nodeA)
	db.PutNode(ctx, nodeB)

	// Link A -> B
	link := &cortex.Link{SourceID: "A", TargetID: "B", RelationType: "RELATED_TO", Weight: 1.0}
	if err := db.LinkNodes(ctx, link); err != nil {
		t.Fatalf("failed to link nodes: %v", err)
	}

	// Verify neighbors
	neighbors, err := db.GetNeighbors(ctx, "A")
	if err != nil {
		t.Fatalf("failed to get neighbors: %v", err)
	}

	if len(neighbors) != 1 {
		t.Errorf("expected 1 neighbor, got %d", len(neighbors))
	}

	if neighbors[0].ID != "B" {
		t.Errorf("expected neighbor B, got %s", neighbors[0].ID)
	}
}

func TestCortexPropagation(t *testing.T) {
	ctx := context.Background()
	os.Remove("test_prop.db")
	defer os.Remove("test_prop.db")

	db, err := cortex.Open("test_prop.db")
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx, schema); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	ns := &cortex.Namespace{ID: "ns-test", Name: "testing"}
	db.CreateNamespace(ctx, ns)

	// Setup small graph: A -> B -> C
	nodeA := &cortex.Node{ID: "A", Content: "Fact A", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}
	nodeB := &cortex.Node{ID: "B", Content: "Fact B", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}
	nodeC := &cortex.Node{ID: "C", Content: "Fact C", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}

	db.PutNode(ctx, nodeA)
	db.PutNode(ctx, nodeB)
	db.PutNode(ctx, nodeC)

	db.LinkNodes(ctx, &cortex.Link{SourceID: "A", TargetID: "B", RelationType: "CAUSED_BY", Weight: 1.0})
	db.LinkNodes(ctx, &cortex.Link{SourceID: "B", TargetID: "C", RelationType: "CAUSED_BY", Weight: 1.0})

	// Propagate from seed A
	scores, err := db.Propagate(ctx, []string{"A"}, 0.85, 2)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Verify scores are present for connected nodes
	if scores["A"] == 0 {
		t.Error("expected non-zero score for seed A")
	}
	if scores["B"] == 0 {
		t.Error("expected non-zero score for neighbor B")
	}
	if scores["C"] == 0 {
		t.Error("expected non-zero score for 2-hop neighbor C")
	}

	// A should have highest score due to teleportation in this small example
	if scores["A"] <= scores["B"] {
		t.Errorf("expected A score > B score, got A:%f, B:%f", scores["A"], scores["B"])
	}
}

func TestCortexProvenance(t *testing.T) {
	ctx := context.Background()
	os.Remove(testDBPath)
	defer os.Remove(testDBPath)

	db, err := cortex.Open(testDBPath)
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx, schema); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// 1. Create a namespace
	ns := &cortex.Namespace{ID: "ns-eng", Name: "engineering"}
	if err := db.CreateNamespace(ctx, ns); err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}

	// 2. Insert a node with strict provenance
	originalNode := &cortex.Node{
		ID:          "node-1",
		Content:     "GoogleSQL is not F1 SQL.",
		Author:      "jesuscolin",
		SourceType:  "G3DOC",
		SourceRef:   "http://go/sql-diff",
		NamespaceID: ns.ID,
		Metadata:    map[string]any{"confidence": 0.95},
	}

	if err := db.PutNode(ctx, originalNode); err != nil {
		t.Fatalf("failed to put node: %v", err)
	}

	// 3. Retrieve and verify
	retrieved, err := db.GetNode(ctx, "node-1")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if retrieved == nil {
		t.Fatal("node not found")
	}

	if retrieved.Author != "jesuscolin" {
		t.Errorf("expected author jesuscolin, got %s", retrieved.Author)
	}

	if retrieved.SourceType != "G3DOC" {
		t.Errorf("expected source_type G3DOC, got %s", retrieved.SourceType)
	}

	if retrieved.SourceRef != "http://go/sql-diff" {
		t.Errorf("expected source_ref http://go/sql-diff, got %s", retrieved.SourceRef)
	}

	if retrieved.Metadata["confidence"] != 0.95 {
		t.Errorf("expected confidence 0.95, got %v", retrieved.Metadata["confidence"])
	}
}

func TestCortexSearch(t *testing.T) {
	ctx := context.Background()
	os.Remove("test_search.db")
	defer os.Remove("test_search.db")

	db, err := cortex.Open("test_search.db")
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx, schema); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	ns := &cortex.Namespace{ID: "ns-test", Name: "testing"}
	db.CreateNamespace(ctx, ns)

	node1 := &cortex.Node{ID: "node-id-123", Content: "GoogleSQL is powerful", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}
	node2 := &cortex.Node{ID: "node-2", Content: "F1 is a database", Author: "tester", SourceType: "MANUAL", NamespaceID: ns.ID}

	db.PutNode(ctx, node1)
	db.PutNode(ctx, node2)

	// Search by content
	results, err := db.SearchNodes(ctx, "SQL")
	if err != nil {
		t.Fatalf("SearchNodes failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'SQL', got %d", len(results))
	} else if results[0].ID != "node-id-123" {
		t.Errorf("expected node-id-123, got %s", results[0].ID)
	}

	// Search by ID
	results, err = db.SearchNodes(ctx, "123")
	if err != nil {
		t.Fatalf("SearchNodes failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for '123', got %d", len(results))
	} else if results[0].ID != "node-id-123" {
		t.Errorf("expected node-id-123, got %s", results[0].ID)
	}

	// Search with no results
	results, err = db.SearchNodes(ctx, "missing")
	if err != nil {
		t.Fatalf("SearchNodes failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
