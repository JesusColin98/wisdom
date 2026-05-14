// Package integrations implements the Wisdom-Integrations gRPC service.
// It is the single authoritative bridge between Expert Agents and local
// MCP servers (Obsidian, Anki, Logseq). Expert Agents NEVER call MCP
// servers directly — all requests route through this service.
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
	masterv1 "github.com/google/wisdom/pkg/mastery/v1"
	pb "github.com/google/wisdom/pkg/integrations/v1"
)

// Config holds all dependencies for the Integrations service.
type Config struct {
	CortexClient        cortexv1.CortexClient
	MasteryClient       masterv1.MasteryClient
	ObsidianMCPURL      string // e.g., "http://localhost:3333"
	AnkiMCPURL          string // e.g., "http://localhost:3334"
	GCPProject          string
	AnkiPollIntervalSec int
}

// Server implements IntegrationsServer.
type Server struct {
	pb.UnimplementedIntegrationsServer

	cortex    cortexv1.CortexClient
	mastery   masterv1.MasteryClient
	httpCli   *http.Client
	pubsubCli *pubsub.Client
	cfg       Config

	// Pub/Sub topic: wisdom.integrations.sync_ready
	syncReadyTopic *pubsub.Topic
}

// NewServer initializes the Integrations server.
func NewServer(cfg Config) (*Server, error) {
	ctx := context.Background()

	psCli, err := pubsub.NewClient(ctx, cfg.GCPProject)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %w", err)
	}

	return &Server{
		cortex:         cfg.CortexClient,
		mastery:        cfg.MasteryClient,
		httpCli:        &http.Client{Timeout: 10 * time.Second},
		pubsubCli:      psCli,
		cfg:            cfg,
		syncReadyTopic: psCli.Topic("wisdom.integrations.sync_ready"),
	}, nil
}

// CreateNote sends a knowledge note to the Obsidian (or Logseq) MCP server.
// If the MCP server is offline, queues it in Cortex with PENDING_SYNC status.
func (s *Server) CreateNote(ctx context.Context, req *pb.NoteRequest) (*pb.IntegrationResult, error) {
	if req.AgentName == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_name and user_id are required")
	}

	payload := map[string]any{
		"action":        "create_note",
		"path":          req.TargetPath,
		"title":         req.Metadata.GetTitle(),
		"tags":          req.Metadata.GetTags(),
		"content":       req.Content,
		"relationships": req.Relationships,
		"mastery_score": req.Metadata.GetMasteryScore(),
	}

	err := s.callMCPServer(ctx, s.cfg.ObsidianMCPURL+"/tools/create_note", payload)
	if err != nil {
		log.Printf("WARN: Obsidian MCP offline, queuing note for %s: %v", req.TargetPath, err)
		if qErr := s.queuePendingItem(ctx, "NOTE", "OBSIDIAN", req); qErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to queue pending note: %v", qErr)
		}
		s.notifyPortalPending(ctx, req.UserId)
		return &pb.IntegrationResult{Success: false, Status: "PENDING_SYNC"}, nil
	}

	return &pb.IntegrationResult{Success: true, Status: "SYNCED"}, nil
}

// CreateCard sends a flashcard to the Anki MCP server.
// If Anki is offline, queues it in Cortex with PENDING_SYNC status.
func (s *Server) CreateCard(ctx context.Context, req *pb.CardRequest) (*pb.IntegrationResult, error) {
	if req.AgentName == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_name and user_id are required")
	}

	payload := map[string]any{
		"action":    "add_note",
		"deck_name": req.DeckName,
		"model":     cardTypeToModel(req.CardType),
		"front":     req.Front,
		"back":      req.Back,
		"tags":      req.Tags,
	}
	if req.CardType == pb.AnkiCardType_CLOZE {
		payload["text"] = req.ClozeText
		payload["extra"] = req.Extra
	}

	err := s.callMCPServer(ctx, s.cfg.AnkiMCPURL+"/tools/add_note", payload)
	if err != nil {
		log.Printf("WARN: Anki MCP offline, queuing card for deck %s: %v", req.DeckName, err)
		if qErr := s.queuePendingItem(ctx, "CARD", "ANKI", req); qErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to queue pending card: %v", qErr)
		}
		s.notifyPortalPending(ctx, req.UserId)
		return &pb.IntegrationResult{Success: false, Status: "PENDING_SYNC"}, nil
	}

	return &pb.IntegrationResult{Success: true, Status: "SYNCED"}, nil
}

// GetPendingQueue returns all items awaiting sync for a user.
func (s *Server) GetPendingQueue(ctx context.Context, req *pb.QueueRequest) (*pb.PendingQueue, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	// TODO: Query Cortex for nodes of type "Signal" with status="PENDING_SYNC" and user_id match.
	return &pb.PendingQueue{Items: []*pb.PendingItem{}, TotalCount: 0}, nil
}

// RetryPendingSync attempts to re-send all queued items to their MCP servers.
func (s *Server) RetryPendingSync(ctx context.Context, req *pb.QueueRequest) (*pb.RetryResult, error) {
	queue, err := s.GetPendingQueue(ctx, req)
	if err != nil {
		return nil, err
	}

	var succeeded, failed int32
	for _, item := range queue.Items {
		_ = item
		// TODO: Deserialize payload_json, determine target app, re-call MCP.
		succeeded++
	}

	return &pb.RetryResult{
		Succeeded:    succeeded,
		Failed:       failed,
		StillPending: 0,
	}, nil
}

// ProcessAnkiReviews is called by the internal Anki polling goroutine.
// It forwards the batch to the Mastery service for MasteryScore updates.
func (s *Server) ProcessAnkiReviews(ctx context.Context, req *pb.AnkiReviewBatch) (*pb.AnkiSyncResult, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	masterBatch := &masterv1.AnkiReviewBatch{
		UserId:  req.UserId,
		Reviews: make([]*masterv1.AnkiReview, 0, len(req.Reviews)),
	}
	for _, r := range req.Reviews {
		masterBatch.Reviews = append(masterBatch.Reviews, &masterv1.AnkiReview{
			AnkiCardId:   r.AnkiCardId,
			WisdomNodeId: r.WisdomNodeId,
			Grade:        r.Grade,
			ReviewId:     r.ReviewId,
			ReviewedAt:   r.ReviewedAt,
		})
	}

	result, err := s.mastery.SyncAnkiReviews(ctx, masterBatch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "mastery sync failed: %v", err)
	}

	return &pb.AnkiSyncResult{
		SyncedCount:  result.SyncedCount,
		SkippedCount: result.SkippedCount,
		ErrorCount:   result.ErrorCount,
	}, nil
}

// StartAnkiPoller runs the background polling loop (every 15 minutes by spec).
// Must be called as a goroutine from cmd/integrations/main.go.
func (s *Server) StartAnkiPoller() {
	interval := time.Duration(s.cfg.AnkiPollIntervalSec) * time.Second
	log.Printf("Anki polling loop started (interval: %v)", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		log.Println("Anki poller: fetching reviews...")

		// TODO: Call AnkiConnect via Anki MCP server to get recent reviews.
		// Call s.ProcessAnkiReviews with the batch.
		// Use review_id (Anki timestamp) to deduplicate via Cortex.
		_ = ctx
	}
}

// ─── Internal Helpers ────────────────────────────────────────────────────────

func (s *Server) callMCPServer(ctx context.Context, url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpCli.Do(req)
	if err != nil {
		return fmt.Errorf("MCP server unreachable (ECONNREFUSED): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("MCP server returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *Server) queuePendingItem(ctx context.Context, itemType, targetApp string, payload any) error {
	data, _ := json.Marshal(payload)
	_ = data
	// TODO: Store in Cortex with type="Signal", status="PENDING_SYNC".
	return nil
}

func (s *Server) notifyPortalPending(ctx context.Context, userID string) {
	msg := fmt.Sprintf(`{"user_id":%q,"message":"Items pending sync. Please open your local apps."}`, userID)
	result := s.syncReadyTopic.Publish(ctx, &pubsub.Message{Data: []byte(msg)})
	if _, err := result.Get(ctx); err != nil {
		log.Printf("WARN: failed to publish sync_ready event: %v", err)
	}
}

func cardTypeToModel(ct pb.AnkiCardType) string {
	if ct == pb.AnkiCardType_CLOZE {
		return "Wisdom-Cloze"
	}
	return "Wisdom-Basic"
}
