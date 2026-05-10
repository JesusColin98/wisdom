package cortex

import (
	"time"
)

// ExternalLink represents a structured pointer to multimodal data.
type ExternalLink struct {
	URI      string         `json:"uri"`
	Type     string         `json:"type"`                // BUG, VIDEO, DOC, IMAGE, TRACE
	Title    string         `json:"title,omitempty"`     // Human-friendly name
	Metadata map[string]any `json:"metadata,omitempty"`  // Extra context (e.g., timestamps for video)
}

// Node represents a discrete unit of wisdom in the Cortex.
type Node struct {
	ID              string         `json:"id"`
	Content         string         `json:"content"`
	EntityClass     string         `json:"entity_class"` // PERSON, ROLE, CONCEPT, ERROR_PATTERN, etc.
	Author          string         `json:"author"`       // Who registered (e.g., jesuscolin)
	SourceType      string         `json:"source_type"`  // BUGANIZER, TABLE, URL, MANUAL, REM_CYCLE
	SourceRef       string         `json:"source_ref"`   // b/123, table_name, session_id, etc.
	NamespaceID     string         `json:"namespace_id"`
	Metadata        map[string]any `json:"metadata"`
	ConfidenceScore float64        `json:"confidence_score"`
	ImpactScore     float64        `json:"impact_score"`   // Determines if node survives pruning
	Stratum         string         `json:"stratum"`        // HOT (Fast/SQLite), COLD (Deep/GCS-Archive)
	SourceMimeType  string         `json:"source_mime_type"` // text/plain, application/pdf, etc.
	ExternalLinks   []ExternalLink `json:"external_links"` // Structured multimodal links
	SupersededByID  string         `json:"superseded_by_id"` // Traceable Neurogenesis: Link to the newer version
	ValidFrom       time.Time      `json:"valid_from"`       // Temporal logic
	ValidUntil      time.Time      `json:"valid_until"`      // Temporal logic (zero for current truths)
	RepetitionCount int            `json:"repetition_count"` // SM-2
	EasinessFactor  float64        `json:"easiness_factor"`  // SM-2
	NextReviewAt    time.Time      `json:"next_review_at"`   // SM-2
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Link represents a relationship between two wisdom nodes.
type Link struct {
	SourceID     string    `json:"source_id"`
	TargetID     string    `json:"target_id"`
	RelationType string    `json:"relation_type"` // IS_A, CAUSED_BY, etc.
	Weight       float64   `json:"weight"`
	CreatedAt    time.Time `json:"created_at"`
}

// Namespace provides logical isolation.
type Namespace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// ScoredNode represents a node with a search relevance score.
type ScoredNode struct {
	Node
	Score float64 `json:"score"`
}

// NodeHistory represents a previous version of a wisdom node.
type NodeHistory struct {
	NodeID           string         `json:"node_id"`
	Content          string         `json:"content"`
	Metadata         map[string]any `json:"metadata"`
	ExternalLinks    []ExternalLink `json:"external_links"`
	VersionTimestamp time.Time      `json:"version_timestamp"`
}
