package metabolism

// EfficiencyReport provides detailed performance and throughput metrics.
type EfficiencyReport struct {
	TSR           float64
	MetabolicRate float64
	TotalTokens   int
	SignalUnits   int
}

// CalculateTSR computes the Token-to-Signal Ratio.
// TSR = SignalUnits / (TokensIn + TokensOut)
func CalculateTSR(u Usage) float64 {
	totalTokens := u.TokensIn + u.TokensOut
	if totalTokens == 0 {
		return 0
	}
	return float64(u.SignalUnits) / float64(totalTokens)
}

// CalculateMetabolicRate computes the number of tokens processed per second.
func CalculateMetabolicRate(u Usage) float64 {
	seconds := u.Duration.Seconds()
	if seconds == 0 {
		return 0
	}
	totalTokens := u.TokensIn + u.TokensOut
	return float64(totalTokens) / seconds
}
