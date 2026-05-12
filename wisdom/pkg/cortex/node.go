package cortex

import (
	"time"
)

// NodeType represents the type of a node in the universal graph.
type NodeType string

const (
	NodeFact    NodeType = "Fact"
	NodeSignal  NodeType = "Signal"
	NodeConcept NodeType = "Concept"
	NodeUser    NodeType = "User"
)

// RelationType represents the type of a relation between nodes.
type RelationType string

const (
	RelationTheoryOf       RelationType = "THEORY_OF"
	RelationContradicts    RelationType = "CONTRADICTS"
	RelationPrerequisiteOf RelationType = "PREREQUISITE_OF"
	RelationMasteredBy     RelationType = "MASTERED_BY"
)

// Node represents a discrete unit of wisdom in the Cortex universal graph.
type Node struct {
	ID            string         `json:"id"`
	Type          NodeType       `json:"type"`
	Payload       map[string]any `json:"payload"`
	Confidence    float64        `json:"confidence"`
	RequiresHuman bool           `json:"requires_human"`
	TTL           *time.Time     `json:"ttl,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// Edge represents a relationship between two nodes.
type Edge struct {
	SourceID  string       `json:"source_id"`
	TargetID  string       `json:"target_id"`
	Relation  RelationType `json:"relation"`
	CreatedAt time.Time    `json:"created_at"`
}

// CognitionResponse represents a node and its direct neighbors.
type CognitionResponse struct {
	Center   *Node   `json:"center"`
	OutEdges []*Edge `json:"out_edges"`
	InEdges  []*Edge `json:"in_edges"`
	Nodes    []*Node `json:"nodes"` // Neighbors
}
