package thalamus

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// Scheduler manages the spaced repetition cycle for wisdom nodes.
type Scheduler struct {
	Cortex *cortex.Cortex
}

// NewScheduler initializes a new spaced repetition scheduler.
func NewScheduler(c *cortex.Cortex) *Scheduler {
	return &Scheduler{Cortex: c}
}

// ReviewNode updates a node's repetition metadata based on user feedback (SM-2).
// grade: 0-5 (0: total blackout, 5: perfect response)
func (s *Scheduler) ReviewNode(ctx context.Context, nodeID string, grade int) error {
	ctx, span := observability.Tracer.Start(ctx, "Scheduler.ReviewNode")
	defer span.End()

	node, err := s.Cortex.GetNode(ctx, nodeID)
	if err != nil {
		return err
	}
	if node == nil {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// SM-2 Algorithm Implementation
	if grade >= 3 {
		// Successful review
		if node.RepetitionCount == 0 {
			node.RepetitionCount = 1
			node.NextReviewAt = time.Now().Add(24 * time.Hour) // 1 day
		} else if node.RepetitionCount == 1 {
			node.RepetitionCount = 2
			node.NextReviewAt = time.Now().Add(6 * 24 * time.Hour) // 6 days
		} else {
			// n > 2: I(n) = I(n-1) * EF
			interval := float64(time.Until(node.NextReviewAt).Hours() / 24)
			if interval <= 0 {
				// Fallback if reviewed early or exactly on time
				interval = 6
			}
			newInterval := math.Ceil(interval * node.EasinessFactor)
			node.RepetitionCount++
			node.NextReviewAt = time.Now().Add(time.Duration(newInterval) * 24 * time.Hour)
		}

		// Update Easiness Factor: EF = EF + (0.1 - (5-grade)*(0.08 + (5-grade)*0.02))
		ef := node.EasinessFactor + (0.1 - float64(5-grade)*(0.08+float64(5-grade)*0.02))
		if ef < 1.3 {
			ef = 1.3
		}
		node.EasinessFactor = ef
	} else {
		// Failed review: Reset interval
		node.RepetitionCount = 0
		node.NextReviewAt = time.Now().Add(24 * time.Hour)
	}

	return s.Cortex.PutNode(ctx, node)
}

// GetDueNodes retrieves all nodes that are due for review.
func (s *Scheduler) GetDueNodes(ctx context.Context, namespace string, limit int) ([]cortex.Node, error) {
	return s.Cortex.ListDueNodes(ctx, namespace, limit)
}
