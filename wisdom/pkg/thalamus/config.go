package thalamus

// WisdomConfig centralizes all adjustable parameters for the Knowledge Runtime.
type WisdomConfig struct {
	// --- Retrieval Parameters ---
	
	// DefaultRetrievalDepth (Iterations)
	// Higher (3-5): Better for complex "why" questions, explores more graph connections.
	// Lower (1-2): Faster, less token usage, better for direct facts.
	DefaultRetrievalDepth int `json:"default_retrieval_depth"`

	// UncertaintyThreshold (0.0 - 1.0)
	// Lower (0.3-0.5): More aggressive "High Cost" mode, triggers deep reasoning easily.
	// Higher (0.7-0.9): Conservative, stays in "Low Cost" mode unless very unsure.
	UncertaintyThreshold float64 `json:"uncertainty_threshold"`

	// TokenBudget (Max tokens for context)
	// Higher: Provides more context to the LLM, but increases latency and cost.
	// Lower: Forced density, saves tokens but might miss nuances.
	TokenBudget int `json:"token_budget"`

	// --- Validation & Strictness ---

	// StrictnessPressure (Hallucination Guard)
	// Higher: Zero tolerance for ungrounded terms in STRICT mode.
	// Lower: Allows more creative "leaps" and conceptual blending.
	StrictnessPressure float64 `json:"strictness_pressure"`

	// --- Pruning & Entropy ---

	// EntropyFactor (Lambda)
	// Higher: Facts decay faster. Good for rapidly changing environments (e.g., Oncall schedules).
	// Lower: Facts stay "known" longer. Good for stable documentation.
	EntropyFactor float64 `json:"entropy_factor"`

	// PruningThreshold (Certainty survival)
	// Nodes below this weight are migrated to COLD or deleted.
	PruningThreshold float64 `json:"pruning_threshold"`
}

// DefaultConfig returns the enterprise-standard defaults.
func DefaultConfig() WisdomConfig {
	return WisdomConfig{
		DefaultRetrievalDepth: 2,
		UncertaintyThreshold:  0.6,
		TokenBudget:           2000,
		StrictnessPressure:    0.9,
		EntropyFactor:         0.01,
		PruningThreshold:      0.1,
	}
}
