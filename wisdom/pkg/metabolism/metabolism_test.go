package metabolism

import (
	"testing"
	"time"
)

func TestTracker_Record(t *testing.T) {
	tests := []struct {
		name     string
		records  map[string][]Usage
		wantSess map[string]Usage
		wantGlob Usage
	}{
		{
			name: "single session single record",
			records: map[string][]Usage{
				"sess1": {
					{TokensIn: 10, TokensOut: 20, Duration: time.Second, CostEstimate: 0.1},
				},
			},
			wantSess: map[string]Usage{
				"sess1": {TokensIn: 10, TokensOut: 20, Duration: time.Second, CostEstimate: 0.1},
			},
			wantGlob: Usage{TokensIn: 10, TokensOut: 20, Duration: time.Second, CostEstimate: 0.1},
		},
		{
			name: "single session multiple records",
			records: map[string][]Usage{
				"sess1": {
					{TokensIn: 10, TokensOut: 20, Duration: time.Second, CostEstimate: 0.1},
					{TokensIn: 5, TokensOut: 5, Duration: time.Second, CostEstimate: 0.05},
				},
			},
			wantSess: map[string]Usage{
				"sess1": {TokensIn: 15, TokensOut: 25, Duration: 2 * time.Second, CostEstimate: 0.15},
			},
			wantGlob: Usage{TokensIn: 15, TokensOut: 25, Duration: 2 * time.Second, CostEstimate: 0.15},
		},
		{
			name: "multiple sessions",
			records: map[string][]Usage{
				"sess1": {
					{TokensIn: 10, TokensOut: 20, Duration: time.Second, CostEstimate: 0.1},
				},
				"sess2": {
					{TokensIn: 5, TokensOut: 5, Duration: time.Second, CostEstimate: 0.05},
				},
			},
			wantSess: map[string]Usage{
				"sess1": {TokensIn: 10, TokensOut: 20, Duration: time.Second, CostEstimate: 0.1},
				"sess2": {TokensIn: 5, TokensOut: 5, Duration: time.Second, CostEstimate: 0.05},
			},
			wantGlob: Usage{TokensIn: 15, TokensOut: 25, Duration: 2 * time.Second, CostEstimate: 0.15},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewTracker()
			for sessID, usages := range tt.records {
				for _, u := range usages {
					if err := tracker.Record(sessID, u); err != nil {
						t.Errorf("Record(%q, %+v) returned unexpected error: %v", sessID, u, err)
					}
				}
			}

			for sessID, want := range tt.wantSess {
				got := tracker.Session(sessID)
				if got.TokensIn != want.TokensIn || got.TokensOut != want.TokensOut || got.Duration != want.Duration || !almostEqual(got.CostEstimate, want.CostEstimate) {
					t.Errorf("Session(%q) = %+v, want %+v", sessID, got, want)
				}
			}

			gotGlob := tracker.Global()
			if gotGlob.TokensIn != tt.wantGlob.TokensIn || gotGlob.TokensOut != tt.wantGlob.TokensOut || gotGlob.Duration != tt.wantGlob.Duration || !almostEqual(gotGlob.CostEstimate, tt.wantGlob.CostEstimate) {
				t.Errorf("Global() = %+v, want %+v", gotGlob, tt.wantGlob)
			}
		})
	}
}

func almostEqual(f1, f2 float64) bool {
	const epsilon = 1e-9
	diff := f1 - f2
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func TestTracker_EmptySession(t *testing.T) {
	tracker := NewTracker()
	got := tracker.Session("non-existent")
	want := Usage{}
	if got != want {
		t.Errorf("Session(\"non-existent\") = %+v, want %+v", got, want)
	}
}
