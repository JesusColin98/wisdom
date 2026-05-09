package metabolism

import (
	"testing"
	"time"
)

func TestCalculateTSR(t *testing.T) {
	tests := []struct {
		name  string
		usage Usage
		want  float64
	}{
		{
			name:  "zero tokens",
			usage: Usage{TokensIn: 0, TokensOut: 0, SignalUnits: 10},
			want:  0,
		},
		{
			name:  "normal usage",
			usage: Usage{TokensIn: 100, TokensOut: 100, SignalUnits: 50},
			want:  0.25, // 50 / 200
		},
		{
			name:  "zero signal",
			usage: Usage{TokensIn: 10, TokensOut: 10, SignalUnits: 0},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateTSR(tt.usage); got != tt.want {
				t.Errorf("CalculateTSR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateMetabolicRate(t *testing.T) {
	tests := []struct {
		name  string
		usage Usage
		want  float64
	}{
		{
			name:  "zero duration",
			usage: Usage{TokensIn: 100, TokensOut: 100, Duration: 0},
			want:  0,
		},
		{
			name:  "normal usage",
			usage: Usage{TokensIn: 100, TokensOut: 100, Duration: 2 * time.Second},
			want:  100, // 200 / 2
		},
		{
			name:  "zero tokens",
			usage: Usage{TokensIn: 0, TokensOut: 0, Duration: time.Second},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateMetabolicRate(tt.usage); got != tt.want {
				t.Errorf("CalculateMetabolicRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTracker_Efficiency(t *testing.T) {
	tracker := NewTracker()
	usage1 := Usage{TokensIn: 50, TokensOut: 50, SignalUnits: 25, Duration: time.Second}
	usage2 := Usage{TokensIn: 50, TokensOut: 50, SignalUnits: 25, Duration: time.Second}

	_ = tracker.Record("sess1", usage1)
	_ = tracker.Record("sess2", usage2)

	// Session 1 Efficiency
	eff1 := tracker.Efficiency("sess1")
	if eff1.TSR != 0.25 {
		t.Errorf("sess1 TSR = %v, want 0.25", eff1.TSR)
	}
	if eff1.MetabolicRate != 100 {
		t.Errorf("sess1 MetabolicRate = %v, want 100", eff1.MetabolicRate)
	}
	if eff1.TotalTokens != 100 {
		t.Errorf("sess1 TotalTokens = %v, want 100", eff1.TotalTokens)
	}

	// Global Efficiency
	effGlob := tracker.GlobalEfficiency()
	if effGlob.TSR != 0.25 {
		t.Errorf("global TSR = %v, want 0.25", effGlob.TSR)
	}
	if effGlob.MetabolicRate != 100 {
		t.Errorf("global MetabolicRate = %v, want 100", effGlob.MetabolicRate)
	}
	if effGlob.TotalTokens != 200 {
		t.Errorf("global TotalTokens = %v, want 200", effGlob.TotalTokens)
	}
	if effGlob.SignalUnits != 50 {
		t.Errorf("global SignalUnits = %v, want 50", effGlob.SignalUnits)
	}
}
