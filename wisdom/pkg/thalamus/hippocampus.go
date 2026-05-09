package thalamus

import (
	"context"
	"github.com/google/wisdom/pkg/cortex"
)

// Interaction represents a single turn in a cognitive session.
type Interaction = cortex.Interaction

// Hippocampus manages transient session memory (logs).
// It delegates persistence to the Cortex (SQLite) to ensure logs survive restarts.
type Hippocampus struct {
	storage *cortex.Cortex
}

// NewHippocampus initializes a new persistent hippocampus.
func NewHippocampus(storage *cortex.Cortex) *Hippocampus {
	return &Hippocampus{
		storage: storage,
	}
}

// Record appends an interaction to a session's log.
func (h *Hippocampus) Record(ctx context.Context, sessionID string, i Interaction) error {
	return h.storage.AddLog(ctx, sessionID, i.Role, i.Content)
}

// GetLogs retrieves the full interaction history for a session.
func (h *Hippocampus) GetLogs(ctx context.Context, sessionID string) ([]Interaction, error) {
	return h.storage.GetLogs(ctx, sessionID)
}

// Clear removes logs for a session after consolidation.
func (h *Hippocampus) Clear(ctx context.Context, sessionID string) error {
	return h.storage.ClearLogs(ctx, sessionID)
}
