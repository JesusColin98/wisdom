package sensory

import (
	"strings"
	"testing"
	"time"
)

func TestSensorySummary(t *testing.T) {
	buffer := NewBuffer(5)

	// Test stable
	if buffer.GetSummary() != "Production is stable. No active alerts." {
		t.Errorf("expected stable summary, got %s", buffer.GetSummary())
	}

	// Test signals
	buffer.Ingest(Signal{ID: "s1", Severity: "CRITICAL", Message: "Storage full", Timestamp: time.Now()})
	buffer.Ingest(Signal{ID: "s2", Severity: "WARNING", Message: "High latency", Timestamp: time.Now()})

	summary := buffer.GetSummary()
	if !strings.Contains(summary, "[CRITICAL: 1]") || !strings.Contains(summary, "[WARNING: 1]") {
		t.Errorf("expected summary to contain CRITICAL and WARNING counts, got %q", summary)
	}

	// Test overflow
	for i := 0; i < 10; i++ {
		buffer.Ingest(Signal{ID: "overflow", Severity: "INFO"})
	}

	if len(buffer.List()) != 5 {
		t.Errorf("expected size 5, got %d", len(buffer.List()))
	}
}
