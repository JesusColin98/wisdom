package cerebellum

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/types/known/structpb"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
)

// IngestedData matches the expected payload from the researcher scraper.
type IngestedData struct {
	Title           string   `json:"title"`
	MarkdownContent string   `json:"markdown_content"`
	SourceURL       string   `json:"source_url"`
	SuggestedTags   []string `json:"suggested_tags"`
}

// Worker handles background tasks for the Wisdom system.
type Worker struct {
	db           *sql.DB
	nc           *nats.Conn
	js           nats.JetStreamContext
	cortexClient cortexv1.CortexClient
}

// NewWorker initializes a new Cerebellum worker.
func NewWorker(dbConnStr string, natsURL string, cortexClient cortexv1.CortexClient) (*Worker, error) {
	db, err := sql.Open("pgx", dbConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Ensure the stream exists
	_, err = js.StreamInfo("WISDOM")
	if err != nil {
		log.Printf("Stream WISDOM not found, attempting to create it...")
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     "WISDOM",
			Subjects: []string{"wisdom.>"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create NATS stream: %w", err)
		}
	}

	return &Worker{
		db:           db,
		nc:           nc,
		js:           js,
		cortexClient: cortexClient,
	}, nil
}

// Start begins processing events and background jobs.
func (w *Worker) Start(ctx context.Context) error {
	log.Println("Cerebellum Worker starting...")

	// 1. Subscribe to Ingestion Events
	sub, err := w.js.Subscribe("wisdom.knowledge.ingested", func(msg *nats.Msg) {
		w.handleIngestion(ctx, msg)
	}, nats.Durable("cerebellum-ingester"), nats.ManualAck())
	if err != nil {
		return fmt.Errorf("failed to subscribe to ingestion topic: %w", err)
	}
	defer sub.Unsubscribe()

	// 2. Start the REM Cycle (e.g., runs every 24 hours, but we'll use a shorter ticker for testing/concept)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Cerebellum Worker shutting down...")
			w.nc.Close()
			w.db.Close()
			return nil
		case <-ticker.C:
			w.runREMCycle(ctx)
		}
	}
}

// handleIngestion processes a CloudEvent and saves it to Cortex.
func (w *Worker) handleIngestion(ctx context.Context, msg *nats.Msg) {
	log.Printf("Received ingestion event: %s", string(msg.Data))

	event := cloudevents.NewEvent()
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Error unmarshalling CloudEvent: %v", err)
		msg.Nak() // Negative ack to retry or dead-letter
		return
	}

	var data IngestedData
	if err := event.DataAs(&data); err != nil {
		log.Printf("Error decoding event data: %v", err)
		msg.Nak()
		return
	}

	// Construct payload for Cortex
	payloadMap := map[string]interface{}{
		"title":   data.Title,
		"content": data.MarkdownContent,
		"url":     data.SourceURL,
		"tags":    data.SuggestedTags,
	}

	payloadStruct, err := structpb.NewStruct(payloadMap)
	if err != nil {
		log.Printf("Error converting payload to structpb: %v", err)
		msg.Nak()
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
		msg.Nak()
		return
	}

	log.Printf("Successfully ingested node: %s", req.Id)
	msg.Ack() // Acknowledge successful processing
}

// runREMCycle executes garbage collection and consolidation.
func (w *Worker) runREMCycle(ctx context.Context) {
	log.Println("Initiating REM Cycle...")

	// 1. Garbage Collection (Hard delete expired Signals)
	res, err := w.db.ExecContext(ctx, "DELETE FROM nodes WHERE type = 'Signal' AND ttl < NOW()")
	if err != nil {
		log.Printf("REM Error during Garbage Collection: %v", err)
	} else {
		rows, _ := res.RowsAffected()
		log.Printf("REM GC: Deleted %d expired Signal nodes.", rows)
	}

	// 2. Conflict Detection (Integrity Checker)
	w.detectConflicts(ctx)
}

func (w *Worker) detectConflicts(ctx context.Context) {
	// A basic implementation: Find Facts with the exact same 'url' in their JSONB payload
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

	var lastURL string
	var lastID string

	for rows.Next() {
		var id string
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			continue
		}

		if url == lastURL {
			// We found a conflict between `lastID` and `id`
			log.Printf("Conflict detected between %s and %s for URL: %s", lastID, id, url)

			// Mark the newer one as requiring human review and link them
			updateQuery := `UPDATE nodes SET requires_human = true WHERE id = $1`
			w.db.ExecContext(ctx, updateQuery, id)

			edgeQuery := `INSERT INTO edges (source_id, target_id, relation) VALUES ($1, $2, 'CONTRADICTS') ON CONFLICT DO NOTHING`
			w.db.ExecContext(ctx, edgeQuery, lastID, id)

			// Publish event to NATS
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
			w.nc.Publish("wisdom.memory.conflict_detected", eventBytes)
		} else {
			lastURL = url
			lastID = id
		}
	}
}
