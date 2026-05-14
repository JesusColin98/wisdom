// Package curriculum implements the Wisdom-Curriculum gRPC service.
// It is the "Academic Dean" — transforms raw Researcher signals into
// structured, sequential learning paths ordered by prerequisites.
package curriculum

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
	masterv1 "github.com/google/wisdom/pkg/mastery/v1"
	pb "github.com/google/wisdom/pkg/curriculum/v1"
)

// Server implements the CurriculumServer gRPC interface.
type Server struct {
	pb.UnimplementedCurriculumServer

	cortex    cortexv1.CortexClient
	mastery   masterv1.MasteryClient
	pubsubCli *pubsub.Client
	project   string

	// Pub/Sub topic: wisdom.learning.path_created
	pathCreatedTopic *pubsub.Topic
}

// NewServer creates a new Curriculum server.
func NewServer(cortex cortexv1.CortexClient, mastery masterv1.MasteryClient, gcpProject string) (*Server, error) {
	ctx := context.Background()

	psCli, err := pubsub.NewClient(ctx, gcpProject)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %w", err)
	}

	return &Server{
		cortex:           cortex,
		mastery:          mastery,
		pubsubCli:        psCli,
		project:          gcpProject,
		pathCreatedTopic: psCli.Topic("wisdom.learning.path_created"),
	}, nil
}

// GeneratePath creates a personalized learning path for a topic.
// Filters out concepts already dominated by the user (MasteryScore > 0.8).
func (s *Server) GeneratePath(ctx context.Context, req *pb.PathRequest) (*pb.LearningPath, error) {
	if req.Topic == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "topic and user_id are required")
	}

	// TODO: Query Cortex for all nodes related to req.Topic via PART_OF / THEORY_OF edges.
	// TODO: Fetch user's strengths from Mastery to filter out already mastered concepts.
	// TODO: Order modules by prerequisite dependencies (topological sort).
	// TODO: Assign difficulty tiers 1-5.

	path := &pb.LearningPath{
		TopicId:    fmt.Sprintf("topic:%s", req.Topic),
		TopicTitle: req.Topic,
		Domain:     req.Domain,
		Modules:    []*pb.Module{},
	}

	// Publish event so Mastery can initialize progress markers.
	if err := s.publishPathCreated(ctx, req.UserId, path); err != nil {
		log.Printf("WARN: failed to publish path_created event: %v", err)
	}

	return path, nil
}

// MapDependencies returns the prerequisite graph for a list of concept nodes.
func (s *Server) MapDependencies(ctx context.Context, req *pb.NodeList) (*pb.DependencyGraph, error) {
	if len(req.NodeIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "node_ids must not be empty")
	}

	// TODO: For each node_id, call cortex.Recall with depth=1 and extract
	// PREREQUISITE_OF / PART_OF edges.
	return &pb.DependencyGraph{Edges: []*pb.Dependency{}}, nil
}

// AssignDifficulty analyzes raw content and assigns a difficulty tier (1-5).
func (s *Server) AssignDifficulty(ctx context.Context, req *pb.DifficultyRequest) (*pb.DifficultyResult, error) {
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	// TODO: Use heuristics (vocabulary complexity, concept density) or call
	// an LLM classifier via Thalamus to determine difficulty tier.
	return &pb.DifficultyResult{
		NodeId:    req.NodeId,
		Tier:      2, // Default: intermediate
		Rationale: "placeholder — difficulty analysis not yet implemented",
	}, nil
}

// ReprioritizeForUser re-orders a user's learning path based on struggle signals.
// Called when Pub/Sub delivers a wisdom.user.struggle_detected event.
func (s *Server) ReprioritizeForUser(ctx context.Context, req *pb.ReprioritizeRequest) (*pb.LearningPath, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// TODO: For each node in struggle_node_ids, find the weakest prerequisites
	// and insert "reinforcement" modules at the front of the path.
	return &pb.LearningPath{
		Domain:  "unknown",
		Modules: []*pb.Module{},
	}, nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func (s *Server) publishPathCreated(ctx context.Context, userID string, path *pb.LearningPath) error {
	moduleIDs := make([]string, 0, len(path.Modules))
	for _, m := range path.Modules {
		moduleIDs = append(moduleIDs, m.Id)
	}

	msg := fmt.Sprintf(`{"path_id":%q,"topic":%q,"user_id":%q,"module_count":%d}`,
		path.TopicId, path.TopicTitle, userID, len(moduleIDs))

	result := s.pathCreatedTopic.Publish(ctx, &pubsub.Message{Data: []byte(msg)})
	_, err := result.Get(ctx)
	return err
}
