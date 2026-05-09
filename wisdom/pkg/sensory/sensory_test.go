package sensory

import (
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
	expected := "Proprioception Pulse: 2 active signals. [CRITICAL: 1] [WARNING: 1] "
	if summary != expected {
		t.Errorf("expected %q, got %q", expected, summary)
	}

	// Test overflow
	for i := 0; i < 10; i++ {
		buffer.Ingest(Signal{ID: "overflow", Severity: "INFO"})
	}

	if len(buffer.List()) != 5 {
		t.Errorf("expected size 5, got %d", len(buffer.List()))
	}
}
