package thalamus

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
)

func TestCoach_ExtractPatterns(t *testing.T) {
	dbPath := "coach_test.db"
	defer os.Remove(dbPath)

	ctx := context.Background()
	c, err := cortex.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer c.Close()

	// Init schema
	schema, _ := os.ReadFile("../cortex/schema.sql")
	if err := c.InitSchema(ctx, string(schema)); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	scheduler := NewScheduler(c)
	mockLLM := &cerebellum.MockLLM{
		CannedResponse: `[
			{"content": "Sicilian Defense", "entity_class": "CONCEPT", "relation": "MASTERED_BY", "metadata": {"eco": "B20"}},
			{"content": "Knight Fork", "entity_class": "PATTERN", "relation": "STRUGGLES_WITH", "metadata": {"difficulty": "medium"}}
		]`,
	}

	coach := NewCoach(c, scheduler, mockLLM)

	userID := "jesuscolin"
	namespaceID := "chess-coaching"
	_ = c.CreateNamespace(ctx, &cortex.Namespace{ID: namespaceID, Name: "Chess"})

	// Create user node to satisfy foreign key constraints
	_ = c.PutNode(ctx, &cortex.Node{
		ID:          userID,
		Content:     "Jesus Colin",
		EntityClass: "PERSON",
		NamespaceID: namespaceID,
		Author:      "system",
	})

	nodeIDs, err := coach.ExtractPatterns(ctx, userID, namespaceID, "User successfully played Sicilian but missed a Knight Fork.")
	if err != nil {
		t.Fatalf("ExtractPatterns failed: %v", err)
	}

	if len(nodeIDs) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodeIDs))
	}

	// Verify links
	weaknesses, err := coach.DiscoverWeaknesses(ctx, userID)
	if err != nil {
		t.Fatalf("DiscoverWeaknesses failed: %v", err)
	}

	foundFork := false
	for _, w := range weaknesses {
		if w.Content == "Knight Fork" {
			foundFork = true
		}
	}

	if !foundFork {
		t.Errorf("expected to find Knight Fork in weaknesses")
	}

	// 2. Missing Prerequisite Test
	// Create "Basic Tactics" as prerequisite for "Sicilian Defense"
	basicTactics := &cortex.Node{
		ID:          "basic_tactics",
		Content:     "Basic Tactics",
		EntityClass: "CONCEPT",
		NamespaceID: namespaceID,
		Author:      "system",
	}
	_ = c.PutNode(ctx, basicTactics)

	// Sicilian is "sicilian_defense" (from ExtractPatterns)
	_ = c.LinkNodes(ctx, &cortex.Link{
		SourceID:     "basic_tactics",
		TargetID:     "sicilian_defense",
		RelationType: "PREREQUISITE_OF",
		Weight:       1.0,
	})

	weaknesses, _ = coach.DiscoverWeaknesses(ctx, userID)
	foundPrereq := false
	for _, w := range weaknesses {
		if w.NodeID == "basic_tactics" {
			foundPrereq = true
		}
	}

	if !foundPrereq {
		t.Errorf("expected to find Basic Tactics as a weakness due to missing prerequisite link")
	}
}

func TestCoach_GenerateCurriculum(t *testing.T) {
	dbPath := "curriculum_test.db"
	defer os.Remove(dbPath)

	ctx := context.Background()
	c, err := cortex.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer c.Close()

	// Init schema
	schema, _ := os.ReadFile("../cortex/schema.sql")
	_ = c.InitSchema(ctx, string(schema))

	scheduler := NewScheduler(c)
	coach := NewCoach(c, scheduler, &cerebellum.MockLLM{})

	userID := "jesuscolin"
	namespaceID := "lang-coaching"
	_ = c.CreateNamespace(ctx, &cortex.Namespace{ID: namespaceID, Name: "Languages"})

	// Create a node due for review
	node := &cortex.Node{
		ID:           "verb_ir",
		Content:      "Ir (to go)",
		NamespaceID:  namespaceID,
		Author:       userID,
		NextReviewAt: time.Now().Add(-1 * time.Hour), // Overdue
	}
	_ = c.PutNode(ctx, node)

	curriculum, err := coach.GenerateCurriculum(ctx, userID, namespaceID)
	if err != nil {
		t.Fatalf("GenerateCurriculum failed: %v", err)
	}

	if len(curriculum) == 0 {
		t.Errorf("expected at least 1 item in curriculum")
	}

	if curriculum[0].NodeID != "verb_ir" {
		t.Errorf("expected verb_ir, got %s", curriculum[0].NodeID)
	}
}
