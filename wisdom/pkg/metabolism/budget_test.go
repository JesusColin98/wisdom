package metabolism

import (
	"errors"
	"testing"
	"time"
)

func TestBudget_Enforce(t *testing.T) {
	tests := []struct {
		name    string
		limit   Limit
		initial Usage
		usage   Usage
		wantErr bool
	}{
		{
			name:    "under limit - tokens in",
			limit:   Limit{MaxTokensIn: 100},
			initial: Usage{TokensIn: 50},
			usage:   Usage{TokensIn: 40},
			wantErr: false,
		},
		{
			name:    "at limit - tokens in",
			limit:   Limit{MaxTokensIn: 100},
			initial: Usage{TokensIn: 50},
			usage:   Usage{TokensIn: 50},
			wantErr: false,
		},
		{
			name:    "over limit - tokens in",
			limit:   Limit{MaxTokensIn: 100},
			initial: Usage{TokensIn: 50},
			usage:   Usage{TokensIn: 51},
			wantErr: true,
		},
		{
			name:    "over limit - tokens out",
			limit:   Limit{MaxTokensOut: 100},
			initial: Usage{TokensOut: 50},
			usage:   Usage{TokensOut: 51},
			wantErr: true,
		},
		{
			name:    "over limit - cost",
			limit:   Limit{MaxCost: 1.0},
			initial: Usage{CostEstimate: 0.5},
			usage:   Usage{CostEstimate: 0.6},
			wantErr: true,
		},
		{
			name:    "over limit - duration",
			limit:   Limit{MaxDuration: time.Minute},
			initial: Usage{Duration: 30 * time.Second},
			usage:   Usage{Duration: 31 * time.Second},
			wantErr: true,
		},
		{
			name:    "unlimited",
			limit:   Limit{},
			initial: Usage{TokensIn: 1000},
			usage:   Usage{TokensIn: 1000},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Budget{
				Limit:        tt.limit,
				CurrentUsage: tt.initial,
			}
			err := b.Enforce(tt.usage)
			if (err != nil) != tt.wantErr {
				t.Errorf("Enforce() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrBudgetExceeded) {
				t.Errorf("Enforce() error should wrap ErrBudgetExceeded, got %v", err)
			}
		})
	}
}

func TestTracker_BudgetIntegration(t *testing.T) {
	t.Run("fails when budget exceeded", func(t *testing.T) {
		tracker := NewTracker()
		sessID := "sess1"
		limit := Limit{MaxTokensIn: 100}
		tracker.SetBudget(sessID, limit)

		// First record is fine
		err := tracker.Record(sessID, Usage{TokensIn: 50})
		if err != nil {
			t.Fatalf("first Record failed: %v", err)
		}

		// Second record exceeds limit
		err = tracker.Record(sessID, Usage{TokensIn: 51})
		if !errors.Is(err, ErrBudgetExceeded) {
			t.Errorf("expected ErrBudgetExceeded, got %v", err)
		}

		// Verify usage was NOT updated after failure
		got := tracker.Session(sessID)
		if got.TokensIn != 50 {
			t.Errorf("expected usage to remain at 50, got %d", got.TokensIn)
		}
	})

	t.Run("unlimited session works", func(t *testing.T) {
		tracker := NewTracker()
		sessID := "unlimited"

		err := tracker.Record(sessID, Usage{TokensIn: 1000000})
		if err != nil {
			t.Errorf("unlimited session Record failed: %v", err)
		}
	})
}
