// Package mastery implements the unified Wisdom-Mastery gRPC service.
// It combines Trace (mastery tracking) and Metabolism (SRS scheduling)
// into a single cohesive binary, as they share the same state.
package mastery

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
	pb "github.com/google/wisdom/pkg/mastery/v1"
)

// Server implements the MasteryServer gRPC interface.
type Server struct {
	pb.UnimplementedMasteryServer

	cortex    cortexv1.CortexClient
	pubsubCli *pubsub.Client
	project   string

	// Pub/Sub topic: wisdom.user.struggle_detected
	struggleTopic *pubsub.Topic
}

// NewServer initializes the Mastery gRPC server.
func NewServer(cortex cortexv1.CortexClient, gcpProject string) (*Server, error) {
	ctx := context.Background()

	psCli, err := pubsub.NewClient(ctx, gcpProject)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %w", err)
	}

	topic := psCli.Topic("wisdom.user.struggle_detected")

	return &Server{
		cortex:        cortex,
		pubsubCli:     psCli,
		project:       gcpProject,
		struggleTopic: topic,
	}, nil
}

// RecordEngagement records a study event and updates the MasteryScore.
// Publishes a wisdom.user.struggle_detected event if score drops below 0.3.
func (s *Server) RecordEngagement(ctx context.Context, req *pb.TraceEvent) (*pb.TraceUpdate, error) {
	if req.UserId == "" || req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and node_id are required")
	}

	// Map score (1-5) to mastery delta using the Integrations grade mapping.
	delta := gradeToDelta(req.Score)

	// TODO: Fetch current MasteryScore from Cortex, apply delta, clamp to [0.0, 1.0],
	// and write back to Cortex via Memorize RPC.
	// Placeholder implementation:
	newScore := clamp(0.5+delta, 0.0, 1.0)
	newStatus := scoreToStatus(newScore)

	// Publish struggle event if score is FRAGILE.
	if newScore < 0.3 {
		if err := s.publishStruggleEvent(ctx, req.UserId, req.NodeId, newScore); err != nil {
			log.Printf("WARN: failed to publish struggle event for node %s: %v", req.NodeId, err)
		}
	}

	return &pb.TraceUpdate{
		NodeId:       req.NodeId,
		MasteryScore: newScore,
		NewStatus:    newStatus,
	}, nil
}

// GetWeaknesses returns the concepts with the lowest MasteryScore for a user.
func (s *Server) GetWeaknesses(ctx context.Context, req *pb.UserRequest) (*pb.ConceptList, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	// TODO: Query Cortex for concepts where MasteryScore < 0.4, ordered ascending.
	return &pb.ConceptList{Concepts: []*pb.Concept{}}, nil
}

// GetStrengths returns the concepts with the highest MasteryScore for a user.
func (s *Server) GetStrengths(ctx context.Context, req *pb.UserRequest) (*pb.ConceptList, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	// TODO: Query Cortex for concepts where MasteryScore > 0.8, ordered descending.
	return &pb.ConceptList{Concepts: []*pb.Concept{}}, nil
}

// SyncAnkiReviews processes a batch of Anki review grades.
// Deduplicated via review_id — prevents double-counting replayed events.
func (s *Server) SyncAnkiReviews(ctx context.Context, req *pb.AnkiReviewBatch) (*pb.SyncResult, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var synced, skipped, errCount int32
	for _, review := range req.Reviews {
		// TODO: Check if review_id already processed in Cortex (deduplication).
		// TODO: Call RecordEngagement for each review.
		_ = review
		synced++
	}

	return &pb.SyncResult{
		SyncedCount:  synced,
		SkippedCount: skipped,
		ErrorCount:   errCount,
	}, nil
}

// GetDueCards returns all concepts due for review (Metabolism engine).
func (s *Server) GetDueCards(ctx context.Context, req *pb.UserRequest) (*pb.DueCardList, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	// TODO: Query Cortex for concepts where next_review_at <= now(), ordered by due date.
	return &pb.DueCardList{Cards: []*pb.DueCard{}}, nil
}

// ScheduleNextReview computes and stores the next review interval (Metabolism engine).
// If scheduler is "ANKI", Wisdom defers to Anki's FSRS date; otherwise uses AI-driven SRS.
func (s *Server) ScheduleNextReview(ctx context.Context, req *pb.ReviewOutcome) (*pb.ScheduleResult, error) {
	if req.UserId == "" || req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and node_id are required")
	}

	// Metabolism: compute next interval. Placeholder SM-2 approximation.
	intervalDays := computeInterval(req.Grade)
	nextReview := time.Now().AddDate(0, 0, intervalDays)

	// TODO: Write next_review_at back to Cortex node metadata.

	// TODO: assign NextReviewAt field once timestamppb import is added.
	_ = nextReview

	return &pb.ScheduleResult{
		NodeId:       req.NodeId,
		IntervalDays: int32(intervalDays),
	}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// gradeToDelta maps Anki grade (1-4) to MasteryScore delta per INTEGRATIONS spec.
func gradeToDelta(grade int32) float64 {
	switch grade {
	case 1: // Again
		return -0.30
	case 2: // Hard
		return +0.05
	case 3: // Good
		return +0.15
	case 4: // Easy
		return +0.30
	default:
		return 0.0
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func scoreToStatus(score float64) string {
	switch {
	case score < 0.3:
		return "FRAGILE"
	case score >= 0.8:
		return "DOMINATED"
	default:
		return "LEARNING"
	}
}

// computeInterval is a placeholder SM-2 approximation.
// Will be replaced by the AI-driven Metabolism algorithm.
func computeInterval(grade int32) int {
	switch grade {
	case 1:
		return 1
	case 2:
		return 3
	case 3:
		return 7
	case 4:
		return 14
	default:
		return 1
	}
}

func (s *Server) publishStruggleEvent(ctx context.Context, userID, nodeID string, score float64) error {
	msg := fmt.Sprintf(`{"user_id":%q,"node_id":%q,"mastery_score":%f}`, userID, nodeID, score)
	result := s.struggleTopic.Publish(ctx, &pubsub.Message{
		Data: []byte(msg),
	})
	_, err := result.Get(ctx)
	return err
}
