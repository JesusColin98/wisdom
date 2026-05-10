package metabolism

import (
	"sync"
	"time"
)

// Usage represents the consumption of resources.

// GatingMode represents the complexity level of processing.
type GatingMode string

const (
	LowCostMode  GatingMode = "LOW_COST"
	HighCostMode GatingMode = "HIGH_COST"
)

// DetermineGating evaluates whether to upgrade to high-cost reasoning.
func DetermineGating(uncertainty float64, isMultiHop bool) GatingMode {
	if uncertainty > 0.7 || isMultiHop {
		return HighCostMode
	}
	return LowCostMode
}

type Usage struct {
	TokensIn     int
	TokensOut    int
	SignalUnits  int
	Duration     time.Duration
	CostEstimate float64
}

// Total returns the additive aggregation of two usage stats.
func (u Usage) Total(other Usage) Usage {
	return Usage{
		TokensIn:     u.TokensIn + other.TokensIn,
		TokensOut:    u.TokensOut + other.TokensOut,
		SignalUnits:  u.SignalUnits + other.SignalUnits,
		Duration:     u.Duration + other.Duration,
		CostEstimate: u.CostEstimate + other.CostEstimate,
	}
}

// Tracker manages thread-safe usage tracking for multiple sessions and global totals.
type Tracker struct {
	mu             sync.RWMutex
	global         Usage
	sessions       map[string]Usage
	sessionBudgets map[string]*Budget
}

// NewTracker initializes a new usage tracker.
func NewTracker() *Tracker {
	return &Tracker{
		sessions:       make(map[string]Usage),
		sessionBudgets: make(map[string]*Budget),
	}
}

// SetBudget assigns a resource limit to a specific session.
func (t *Tracker) SetBudget(sessionID string, limit Limit) {
	t.mu.Lock()
	defer t.mu.Unlock()

	usage := t.sessions[sessionID]
	t.sessionBudgets[sessionID] = &Budget{
		Limit:        limit,
		CurrentUsage: usage,
	}
}

// Record updates the global and session-specific usage metrics.
// Returns an error if a budget is set for the session and adding the usage would exceed it.
func (t *Tracker) Record(sessionID string, usage Usage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if budget, ok := t.sessionBudgets[sessionID]; ok {
		if err := budget.Enforce(usage); err != nil {
			return err
		}
		budget.CurrentUsage = budget.CurrentUsage.Total(usage)
	}

	t.global = t.global.Total(usage)
	t.sessions[sessionID] = t.sessions[sessionID].Total(usage)
	return nil
}

// Global returns the aggregated global usage.
func (t *Tracker) Global() Usage {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.global
}

// Session returns the aggregated usage for a specific session.
func (t *Tracker) Session(sessionID string) Usage {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sessions[sessionID]
}

// Efficiency returns the efficiency report for a specific session.
func (t *Tracker) Efficiency(sessionID string) EfficiencyReport {
	t.mu.RLock()
	usage := t.sessions[sessionID]
	t.mu.RUnlock()

	report := EfficiencyReport{
		TSR:           CalculateTSR(usage),
		MetabolicRate: CalculateMetabolicRate(usage),
		TotalTokens:   usage.TokensIn + usage.TokensOut,
		SignalUnits:   usage.SignalUnits,
	}
	report.HealthStatus = report.GetHealthStatus()
	return report
}

// GlobalEfficiency returns the aggregated efficiency report for all sessions.
func (t *Tracker) GlobalEfficiency() EfficiencyReport {
	t.mu.RLock()
	usage := t.global
	t.mu.RUnlock()

	report := EfficiencyReport{
		TSR:           CalculateTSR(usage),
		MetabolicRate: CalculateMetabolicRate(usage),
		TotalTokens:   usage.TokensIn + usage.TokensOut,
		SignalUnits:   usage.SignalUnits,
	}
	report.HealthStatus = report.GetHealthStatus()
	return report
}
