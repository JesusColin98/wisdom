// anki_poller.go — Background Anki review polling loop.
// Polls AnkiConnect via the local Anki MCP server every 15 minutes (per spec).
// Deduplicates reviews using review_id (Anki timestamp) checked against Cortex.
// Forwards review batches to the Mastery service for MasteryScore updates.
package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	masterv1 "github.com/google/wisdom/pkg/mastery/v1"
	pb "github.com/google/wisdom/pkg/integrations/v1"
)

// AnkiReview represents a single review fetched from AnkiConnect.
type AnkiReview struct {
	CardID     int64  `json:"cardId"`
	NoteID     int64  `json:"noteId"`
	Ease       int32  `json:"ease"`       // 1=Again, 2=Hard, 3=Good, 4=Easy
	ReviewTime int64  `json:"reviewTime"` // Unix timestamp ms — used as review_id for dedup
	ReviewDuration int32 `json:"reviewDuration"` // ms
}

// ankiConnectRequest is the standard AnkiConnect API request format.
type ankiConnectRequest struct {
	Action  string `json:"action"`
	Version int    `json:"version"`
	Params  any    `json:"params,omitempty"`
}

// ankiConnectResponse is the standard AnkiConnect API response format.
type ankiConnectResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}

// StartAnkiPoller runs the background polling loop at a configurable interval.
// Must be called as a goroutine from cmd/integrations/main.go.
func (s *Server) StartAnkiPoller() {
	interval := time.Duration(s.cfg.AnkiPollIntervalSec) * time.Second
	log.Printf("Integrations: Anki polling loop started (interval: %v)", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// The Anki MCP server communicates with AnkiConnect (localhost:8765).
	// We poll the MCP server's proxy endpoint which abstracts AnkiConnect.
	ankiConnectURL := s.cfg.AnkiMCPURL + "/tools/get_reviews"

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		log.Println("Integrations: Anki poller — fetching reviews...")

		if err := s.pollAndSync(ctx, ankiConnectURL); err != nil {
			log.Printf("Integrations: Anki poll error: %v", err)
		}

		cancel()
	}
}

// pollAndSync fetches reviews tagged "Wisdom" from AnkiConnect and syncs them
// to the Mastery service. Uses review_id for deduplication.
func (s *Server) pollAndSync(ctx context.Context, ankiMCPURL string) error {
	// Request reviews for all cards tagged with "Wisdom" (the Wisdom namespace).
	payload := map[string]any{
		"query": "tag:Wisdom",
		"since": time.Now().Add(-16 * time.Minute).UnixMilli(), // overlap for safety
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	resp, err := doHTTPPost(ctx, s.httpCli, ankiMCPURL, body)
	if err != nil {
		// Anki is offline — this is expected if the user hasn't opened Anki yet.
		log.Printf("Integrations: Anki MCP unreachable (Anki likely closed): %v", err)
		return nil // Not fatal.
	}

	var reviews []AnkiReview
	if err := json.Unmarshal(resp, &reviews); err != nil {
		return fmt.Errorf("unmarshal reviews: %w", err)
	}

	if len(reviews) == 0 {
		log.Println("Integrations: Anki poll — no new reviews.")
		return nil
	}

	log.Printf("Integrations: Anki poll — %d reviews fetched.", len(reviews))

	// Build a batch of Anki reviews for the Mastery service.
	// TODO: For each review, look up the Wisdom node ID from Cortex using the Anki card ID.
	batch := &pb.AnkiReviewBatch{
		UserId:  s.cfg.DefaultUserID,
		Reviews: make([]*pb.AnkiReviewEntry, 0, len(reviews)),
	}

	for _, r := range reviews {
		batch.Reviews = append(batch.Reviews, &pb.AnkiReviewEntry{
			AnkiCardId:   fmt.Sprintf("%d", r.CardID),
			WisdomNodeId: "", // TODO: resolve from Cortex card-to-node mapping
			Grade:        r.Ease,
			ReviewId:     fmt.Sprintf("%d", r.ReviewTime), // dedup key
		})
	}

	result, err := s.ProcessAnkiReviews(ctx, batch)
	if err != nil {
		return fmt.Errorf("ProcessAnkiReviews: %w", err)
	}

	log.Printf("Integrations: Anki sync — synced=%d skipped=%d errors=%d",
		result.SyncedCount, result.SkippedCount, result.ErrorCount)

	// Also forward to Mastery service for MasteryScore updates.
	masterBatch := &masterv1.AnkiReviewBatch{
		UserId:  s.cfg.DefaultUserID,
		Reviews: make([]*masterv1.AnkiReview, 0, len(batch.Reviews)),
	}
	for _, r := range batch.Reviews {
		masterBatch.Reviews = append(masterBatch.Reviews, &masterv1.AnkiReview{
			AnkiCardId:   r.AnkiCardId,
			WisdomNodeId: r.WisdomNodeId,
			Grade:        r.Grade,
			ReviewId:     r.ReviewId,
			ReviewedAt:   r.ReviewedAt,
		})
	}

	if _, err := s.mastery.SyncAnkiReviews(ctx, masterBatch); err != nil {
		log.Printf("WARN: mastery.SyncAnkiReviews failed: %v", err)
	}

	return nil
}

// doHTTPPost is a helper that performs a POST request and returns the response body.
func doHTTPPost(ctx context.Context, cli *http.Client, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url,
		newBytesReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}

// newBytesReader wraps a byte slice in an io.Reader.
func newBytesReader(b []byte) io.Reader {
	return &bytesReader{data: b}
}

type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
