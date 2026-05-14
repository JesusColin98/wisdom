// queue.go — PENDING_SYNC queue backed by Cortex.
// When an MCP server (Obsidian, Anki) is offline, items are stored in Cortex
// as Signal nodes with status="PENDING_SYNC". The Integrations service retries
// them every 5 minutes until successful.
package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
	pb "github.com/google/wisdom/pkg/integrations/v1"
)

const (
	pendingSyncRetryInterval = 5 * time.Minute
	maxRetryCount            = 10 // After 10 failures, mark as ERROR and stop retrying.
)

// queuePendingItem persists a failed MCP sync attempt to Cortex.
// The item is stored as a Signal node with type "PendingSync" and will be
// retried by the background retry loop.
func (s *Server) queuePendingItem(ctx context.Context, itemType, targetApp string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal payload: %w", err)
	}

	payloadMap := map[string]interface{}{
		"item_type":    itemType,
		"target_app":   targetApp,
		"payload_json": string(data),
		"status":       "PENDING_SYNC",
		"retry_count":  0,
		"queued_at":    time.Now().UTC().Format(time.RFC3339),
	}

	payloadStruct, err := structpb.NewStruct(payloadMap)
	if err != nil {
		return fmt.Errorf("structpb.NewStruct: %w", err)
	}

	_, err = s.cortex.Memorize(ctx, &cortexv1.IngestRequest{
		Type:          "PendingSync",
		Payload:       payloadStruct,
		Confidence:    1.0,
		RequiresHuman: false,
	})
	if err != nil {
		return fmt.Errorf("cortex.Memorize: %w", err)
	}

	log.Printf("Integrations: queued %s for %s (PENDING_SYNC)", itemType, targetApp)
	return nil
}

// GetPendingQueue returns all items awaiting sync from Cortex.
func (s *Server) GetPendingQueue(ctx context.Context, req *pb.QueueRequest) (*pb.PendingQueue, error) {
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	// Query Cortex for all PendingSync Signal nodes.
	factList, err := s.cortex.QueryFacts(ctx, &cortexv1.FactRequest{
		Query: "PendingSync",
		MetadataFilters: map[string]string{
			"type":   "PendingSync",
			"status": "PENDING_SYNC",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cortex.QueryFacts: %w", err)
	}

	items := make([]*pb.PendingItem, 0, len(factList.Facts))
	for _, node := range factList.Facts {
		if node.Payload == nil {
			continue
		}
		fields := node.Payload.GetFields()

		items = append(items, &pb.PendingItem{
			ItemId:      node.Id,
			ItemType:    stringField(fields, "item_type"),
			TargetApp:   stringField(fields, "target_app"),
			PayloadJson: stringField(fields, "payload_json"),
			RetryCount:  int32(numberField(fields, "retry_count")),
		})
	}

	return &pb.PendingQueue{
		Items:      items,
		TotalCount: int32(len(items)),
	}, nil
}

// RetryPendingSync attempts to re-send all queued items to their MCP servers.
// Called by the Portal "Retry All" button or the background retry loop.
func (s *Server) RetryPendingSync(ctx context.Context, req *pb.QueueRequest) (*pb.RetryResult, error) {
	queue, err := s.GetPendingQueue(ctx, req)
	if err != nil {
		return nil, err
	}

	var succeeded, failed int32
	for _, item := range queue.Items {
		if item.RetryCount >= maxRetryCount {
			log.Printf("Integrations: item %s exceeded max retries, marking ERROR", item.ItemId)
			failed++
			continue
		}

		var retryErr error
		switch item.TargetApp {
		case "OBSIDIAN":
			retryErr = s.retryObsidianNote(ctx, item)
		case "ANKI":
			retryErr = s.retryAnkiCard(ctx, item)
		default:
			log.Printf("Integrations: unknown target app %q for item %s", item.TargetApp, item.ItemId)
			failed++
			continue
		}

		if retryErr != nil {
			log.Printf("Integrations: retry failed for %s: %v", item.ItemId, retryErr)
			failed++
		} else {
			log.Printf("Integrations: retry succeeded for item %s", item.ItemId)
			succeeded++
			// TODO: Delete from Cortex on success (mark status="SYNCED").
		}
	}

	return &pb.RetryResult{
		Succeeded:    succeeded,
		Failed:       failed,
		StillPending: int32(len(queue.Items)) - succeeded - failed,
	}, nil
}

// StartRetryLoop runs the background 5-minute retry loop for PENDING_SYNC items.
// Must be called as a goroutine from cmd/integrations/main.go alongside StartAnkiPoller.
func (s *Server) StartRetryLoop(userID string) {
	log.Printf("Integrations: PENDING_SYNC retry loop started (interval: %v)", pendingSyncRetryInterval)
	ticker := time.NewTicker(pendingSyncRetryInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		result, err := s.RetryPendingSync(ctx, &pb.QueueRequest{UserId: userID})
		if err != nil {
			log.Printf("Integrations: retry loop error: %v", err)
			continue
		}
		if result.Succeeded > 0 || result.Failed > 0 {
			log.Printf("Integrations: retry loop — synced=%d failed=%d pending=%d",
				result.Succeeded, result.Failed, result.StillPending)
		}
	}
}

// retryObsidianNote deserializes a pending note and re-sends it to the Obsidian MCP.
func (s *Server) retryObsidianNote(ctx context.Context, item *pb.PendingItem) error {
	var req pb.NoteRequest
	if err := json.Unmarshal([]byte(item.PayloadJson), &req); err != nil {
		return fmt.Errorf("unmarshal NoteRequest: %w", err)
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
	return s.callMCPServer(ctx, s.cfg.ObsidianMCPURL+"/tools/create_note", payload)
}

// retryAnkiCard deserializes a pending card and re-sends it to the Anki MCP.
func (s *Server) retryAnkiCard(ctx context.Context, item *pb.PendingItem) error {
	var req pb.CardRequest
	if err := json.Unmarshal([]byte(item.PayloadJson), &req); err != nil {
		return fmt.Errorf("unmarshal CardRequest: %w", err)
	}
	payload := map[string]any{
		"action":    "add_note",
		"deck_name": req.DeckName,
		"model":     cardTypeToModel(req.CardType),
		"front":     req.Front,
		"back":      req.Back,
		"tags":      req.Tags,
	}
	return s.callMCPServer(ctx, s.cfg.AnkiMCPURL+"/tools/add_note", payload)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func stringField(fields map[string]*structpb.Value, key string) string {
	v, ok := fields[key]
	if !ok {
		return ""
	}
	return v.GetStringValue()
}

func numberField(fields map[string]*structpb.Value, key string) float64 {
	v, ok := fields[key]
	if !ok {
		return 0
	}
	return v.GetNumberValue()
}
