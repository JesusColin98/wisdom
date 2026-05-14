// mastery_calculator.go — Grade-to-MasteryScore delta mapping.
// Centralizes the arithmetic defined in SERVICE_INTEGRATIONS.md (Gap R2 Fix).
// Rule: score = clamp(current + delta, 0.0, 1.0)
package integrations

import "fmt"

// GradeConfig maps an Anki review grade to its MasteryScore delta and status.
type GradeConfig struct {
	Delta     float64
	StatusTag string // status applied to the Cortex node
}

// AnkiGradeMap is the authoritative grade-to-delta table from the spec.
// Grade 1 (Again) = -0.30 | Grade 2 (Hard) = +0.05
// Grade 3 (Good)  = +0.15 | Grade 4 (Easy) = +0.30
var AnkiGradeMap = map[int32]GradeConfig{
	1: {Delta: -0.30, StatusTag: "FRAGILE"},
	2: {Delta: +0.05, StatusTag: ""},
	3: {Delta: +0.15, StatusTag: ""},
	4: {Delta: +0.30, StatusTag: "DOMINATED"},
}

// ApplyGrade calculates a new MasteryScore given a current score and an Anki grade.
// The result is clamped to [0.0, 1.0] per spec.
func ApplyGrade(currentScore float64, grade int32) (newScore float64, status string, err error) {
	cfg, ok := AnkiGradeMap[grade]
	if !ok {
		return currentScore, "", fmt.Errorf("unknown Anki grade: %d (valid: 1-4)", grade)
	}

	newScore = clampMastery(currentScore + cfg.Delta)

	// Derive status from score if the grade doesn't mandate one.
	if cfg.StatusTag != "" {
		status = cfg.StatusTag
	} else {
		status = scoreToStatusTag(newScore)
	}

	return newScore, status, nil
}

// clampMastery clamps a score to the [0.0, 1.0] range.
func clampMastery(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// scoreToStatusTag maps a MasteryScore to a human-readable status tag.
func scoreToStatusTag(score float64) string {
	switch {
	case score < 0.3:
		return "FRAGILE"
	case score >= 0.8:
		return "DOMINATED"
	default:
		return "LEARNING"
	}
}
