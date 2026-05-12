package cortex

import (
	"context"
	"time"
)

// NodeStore defines the interface for basic node and metadata management.
type NodeStore interface {
	PutNode(ctx context.Context, node *Node) error
	GetNode(ctx context.Context, id string) (*Node, error)
	DeleteNode(ctx context.Context, id string) error
	SearchNodes(ctx context.Context, query string) ([]Node, error)
	ListNodes(ctx context.Context, filter map[string]any) ([]Node, error)
	PruneNodes(ctx context.Context, lambda float64, threshold float64) (int, error)
	ListDueNodes(ctx context.Context, namespaceID string, limit int) ([]Node, error)
	UpdateConfidence(ctx context.Context, nodeID string, delta float64) error
	UpdateImpact(ctx context.Context, nodeID string, delta float64) error
	MoveToCold(ctx context.Context, nodeID string) error
	RecallToHot(ctx context.Context, nodeID string) error
	CreateNamespace(ctx context.Context, ns *Namespace) error
	ListNamespaces(ctx context.Context) ([]Namespace, error)
}

// VectorStore defines the interface for semantic embedding management and search.
type VectorStore interface {
	PutVector(ctx context.Context, nodeID string, embedding []float32, modelVersion string) error
	GetVector(ctx context.Context, nodeID string) ([]float32, string, error)
	SearchVectors(ctx context.Context, embedding []float32, topK int) ([]ScoredNode, error)
	DeleteVector(ctx context.Context, nodeID string) error
}

// GraphStore defines the interface for managing relationships and graph traversal.
type GraphStore interface {
	LinkNodes(ctx context.Context, link *Link) error
	GetNeighbors(ctx context.Context, nodeID string) ([]*Node, error)
	GetEdges(ctx context.Context, sourceIDs []string) ([]Edge, error)
	ListEdges(ctx context.Context) ([]Link, error)
	ResolvePointer(ctx context.Context, pointer string) (string, error)
}


// SessionStore defines the interface for managing conversational history and logs.
type SessionStore interface {
	AddLog(ctx context.Context, sessionID, role, content string) error
	GetLogs(ctx context.Context, sessionID string) ([]Interaction, error)
	ClearLogs(ctx context.Context, sessionID string) error
	GetInactiveSessions(ctx context.Context, olderThan time.Duration) ([]string, error)
}

// StorageEngine consolidates all memory management capabilities.
type StorageEngine interface {
	NodeStore
	VectorStore
	GraphStore
	SessionStore
	Close() error
}
