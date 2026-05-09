package sensory

import (
	"fmt"
	"sync"
	"time"
)

// Signal represents an external observability event (alert, incident).
type Signal struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"` // IRM, Monarch, etc.
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]string `json:"metadata"`
}

// Buffer manages real-time signals in memory.
type Buffer struct {
	mu      sync.RWMutex
	signals []Signal
	maxSize int
}

// NewBuffer creates a new sensory buffer.
func NewBuffer(maxSize int) *Buffer {
	return &Buffer{
		maxSize: maxSize,
	}
}

// Ingest adds a new signal to the buffer.
func (b *Buffer) Ingest(s Signal) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.signals = append(b.signals, s)
	if len(b.signals) > b.maxSize {
		b.signals = b.signals[1:]
	}
}

// GetSummary returns a concise description of active signals for the context window.
func (b *Buffer) GetSummary() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.signals) == 0 {
		return "Production is stable. No active alerts."
	}

	counts := make(map[string]int)
	for _, s := range b.signals {
		counts[s.Severity]++
	}

	summary := fmt.Sprintf("Proprioception Pulse: %d active signals. ", len(b.signals))
	for sev, count := range counts {
		summary += fmt.Sprintf("[%s: %d] ", sev, count)
	}

	return summary
}

// List returns all signals in the buffer.
func (b *Buffer) List() []Signal {
	b.mu.RLock()
	defer b.mu.RUnlock()

	res := make([]Signal, len(b.signals))
	copy(res, b.signals)
	return res
}
