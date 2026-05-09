package metabolism

import (
	"fmt"
	"time"

	"github.com/google/wisdom/pkg/errors"
)

// ErrBudgetExceeded is returned when a resource limit is breached.
var ErrBudgetExceeded = errors.New(errors.CodeResourceExhausted, "resource budget exceeded")

// Limit defines the maximum allowed resources for a session.
type Limit struct {
	MaxTokensIn  int
	MaxTokensOut int
	MaxCost      float64
	MaxDuration  time.Duration
}

// Budget tracks current usage against a set of limits.
type Budget struct {
	Limit        Limit
	CurrentUsage Usage
}

// Enforce checks if adding the provided usage would exceed the budget's limits.
func (b *Budget) Enforce(usage Usage) error {
	total := b.CurrentUsage.Total(usage)

	if b.Limit.MaxTokensIn > 0 && total.TokensIn > b.Limit.MaxTokensIn {
		return fmt.Errorf("%w: tokens in (%d) exceeds limit (%d)", ErrBudgetExceeded, total.TokensIn, b.Limit.MaxTokensIn)
	}
	if b.Limit.MaxTokensOut > 0 && total.TokensOut > b.Limit.MaxTokensOut {
		return fmt.Errorf("%w: tokens out (%d) exceeds limit (%d)", ErrBudgetExceeded, total.TokensOut, b.Limit.MaxTokensOut)
	}
	if b.Limit.MaxCost > 0 && total.CostEstimate > b.Limit.MaxCost {
		return fmt.Errorf("%w: cost (%.4f) exceeds limit (%.4f)", ErrBudgetExceeded, total.CostEstimate, b.Limit.MaxCost)
	}
	if b.Limit.MaxDuration > 0 && total.Duration > b.Limit.MaxDuration {
		return fmt.Errorf("%w: duration (%v) exceeds limit (%v)", ErrBudgetExceeded, total.Duration, b.Limit.MaxDuration)
	}

	return nil
}
