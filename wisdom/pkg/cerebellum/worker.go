// Package cerebellum implements the Wisdom-Cerebellum background worker.
// It handles all heavy, asynchronous processing that must not block the main
// memory retrieval loops: REM Cycle (GC + consolidation) and conflict detection.
//
// Migration note: Previously used NATS JetStream. Migrated to GCP Pub/Sub
// (architecture decision locked 2026-05-14) for Cloud Run compatibility.
// Pub/Sub push subscriptions make this naturally fit Cloud Run scaling.
package cerebellum

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/protobuf/types/known/structpb"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
)

// IngestedData matches the expected payload from the Researcher service.
type IngestedData struct {
	Title           string   `json:"title"`
	MarkdownContent string   `json:"markdown_content"`
	SourceURL       string   `json:"source_url"`
	SuggestedTags   []string `json:"suggested_tags"`
	Domain          string   `json:"domain"`
}

// Worker handles background tasks for the Wisdom Cognitive Runtime.
type Worker struct {
	db           *sql.DB
	pubsubCli    *pubsub.Client
	cortexClient cortexv1.CortexClient
	project      string

	// Subscriptions
	ingestedSub *pubsub.Subscription

	// Publisher for conflict events
	conflictTopic *pubsub.Topic
}

// NewWorker initializes a new Cerebellum background worker.
// Replaces the former NATS-based constructor.
func NewWorker(dbConnStr string, gcpProject string, cortexClient cortexv1.CortexClient) (*Worker, error) {
	db, err := sql.Open("pgx", dbConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx := context.Background()
	psCli, err := pubsub.NewClient(ctx, gcpProject)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %w", err)
	}

	// Subscribe to wisdom.knowledge.ingested events from Researcher.
	ingestedSub := psCli.Subscription("cerebellum-knowledge-ingested")

	// Topic for publishing conflict detection events.
	conflictTopic := psCli.Topic("wisdom.memory.conflict_detected")

	return &Worker{
		db:            db,
		pubsubCli:     psCli,
		cortexClient:  cortexClient,
		project:       gcpProject,
		ingestedSub:   ingestedSub,
		conflictTopic: conflictTopic,
	}, nil
}

// Start begins processing Pub/Sub events and background cron jobs.
func (w *Worker) Start(ctx context.Context) error {
	log.Println("Cerebellum Worker starting (Pub/Sub mode)...")

	// 1. Subscribe to knowledge.ingested events (replaces NATS subscription).
	go func() {
		log.Println("Cerebellum: listening for wisdom.knowledge.ingested events...")
		if err := w.ingestedSub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			w.handleIngestion(ctx, msg)
		}); err != nil {
			log.Printf("Cerebellum: Pub/Sub receive error: %v", err)
		}
	}()

	// 2. Start the REM Cycle (24-hour cron).
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Cerebellum Worker shutting down...")
			w.db.Close()
			w.pubsubCli.Close()
			return nil
		case <-ticker.C:
			w.runREMCycle(ctx)
		}
	}
}

// handleIngestion processes a Pub/Sub message from the Researcher and saves it to Cortex.
func (w *Worker) handleIngestion(ctx context.Context, msg *pubsub.Message) {
	log.Printf("Cerebellum: received knowledge.ingested event (id=%s)", msg.ID)

	event := cloudevents.NewEvent()
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Error unmarshalling CloudEvent: %v", err)
		msg.Nack() // Negative ack — Pub/Sub will redeliver.
		return
	}

	var data IngestedData
	if err := event.DataAs(&data); err != nil {
		log.Printf("Error decoding event data: %v", err)
		msg.Nack()
		return
	}

	// Construct payload for Cortex.
	payloadMap := map[string]interface{}{
		"title":   data.Title,
		"content": data.MarkdownContent,
		"url":     data.SourceURL,
		"tags":    data.SuggestedTags,
		"domain":  data.Domain,
	}

	payloadStruct, err := structpb.NewStruct(payloadMap)
	if err != nil {
		log.Printf("Error converting payload to structpb: %v", err)
		msg.Nack()
		return
	}

	req := &cortexv1.IngestRequest{
		Id:            uuid.NewString(),
		Type:          "Fact",
		Payload:       payloadStruct,
		Confidence:    1.0,
		RequiresHuman: false,
	}

	_, err = w.cortexClient.Memorize(ctx, req)
	if err != nil {
		log.Printf("Error memorizing to Cortex: %v", err)
		msg.Nack()
		return
	}

	log.Printf("Cerebellum: successfully ingested node %s from %s", req.Id, data.SourceURL)
	msg.Ack() // Acknowledge successful processing.
}

// runREMCycle executes the daily memory garbage collection and consolidation.
// Corresponds to the nightly REM cycle defined in REM_LIFECYCLE.md.
func (w *Worker) runREMCycle(ctx context.Context) {
	log.Println("Cerebellum: initiating REM Cycle...")

	// Phase 1: Garbage Collection — hard delete expired Signal nodes (TTL expired, ref_count = 0).
	res, err := w.db.ExecContext(ctx, "DELETE FROM nodes WHERE type = 'Signal' AND ttl < NOW()")
	if err != nil {
		log.Printf("REM Error during Garbage Collection: %v", err)
	} else {
		rows, _ := res.RowsAffected()
		log.Printf("REM GC: deleted %d expired Signal nodes.", rows)
	}

	// Phase 2: Conflict Detection (Integrity Checker).
	w.detectConflicts(ctx)
}

// detectConflicts scans for duplicate Facts (same URL) and creates CONTRADICTS edges.
func (w *Worker) detectConflicts(ctx context.Context) {
	query := `
		WITH duplicates AS (
			SELECT payload->>'url' as url, COUNT(*) as c
			FROM nodes
			WHERE type = 'Fact' AND payload ? 'url'
			GROUP BY payload->>'url'
			HAVING COUNT(*) > 1
		)
		SELECT n.id, n.payload->>'url'
		FROM nodes n
		JOIN duplicates d ON n.payload->>'url' = d.url
		ORDER BY d.url, n.created_at;
	`

	rows, err := w.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("REM Error checking for conflicts: %v", err)
		return
	}
	defer rows.Close()

	var lastURL, lastID string

	for rows.Next() {
		var id, url string
		if err := rows.Scan(&id, &url); err != nil {
			continue
		}

		if url == lastURL {
			log.Printf("Cerebellum: conflict detected between %s and %s for URL: %s", lastID, id, url)

			// Mark the newer node as requiring human resolution.
			w.db.ExecContext(ctx, `UPDATE nodes SET requires_human = true WHERE id = $1`, id)

			// Create a CONTRADICTS edge between the two nodes.
			w.db.ExecContext(ctx,
				`INSERT INTO edges (source_id, target_id, relation) VALUES ($1, $2, 'CONTRADICTS') ON CONFLICT DO NOTHING`,
				lastID, id,
			)

			// Publish conflict_detected event to Pub/Sub (replaces NATS publish).
			event := cloudevents.NewEvent()
			event.SetID(uuid.NewString())
			event.SetSource("/cerebellum/integrity-checker")
			event.SetType("wisdom.memory.conflict_detected")
			event.SetTime(time.Now())
			event.SetData(cloudevents.ApplicationJSON, map[string]interface{}{
				"winning_node_id": lastID,
				"losing_node_id":  id,
				"reason":          "Duplicate URL",
			})

			eventBytes, _ := json.Marshal(event)
			result := w.conflictTopic.Publish(ctx, &pubsub.Message{Data: eventBytes})
			if _, err := result.Get(ctx); err != nil {
				log.Printf("WARN: failed to publish conflict event: %v", err)
			}
		} else {
			lastURL = url
			lastID = id
		}
	}
}
