// Package researcher implements the Wisdom-Researcher gRPC service.
// It handles autonomous, deterministic content gathering from web sources,
// RSS feeds, and book/PDF sources. Publishes wisdom.knowledge.ingested
// events to GCP Pub/Sub upon job completion.
//
// Migration note: Previously used NATS JetStream. Migrated to GCP Pub/Sub
// (architecture decision locked 2026-05-14) for Cloud Run compatibility.
package researcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	md "github.com/JohannesKaufmann/html-to-markdown"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/google/wisdom/pkg/researcher/v1"
)

// IngestedData is the payload published to Pub/Sub when a scrape completes.
type IngestedData struct {
	Title          string   `json:"title"`
	MarkdownContent string  `json:"markdown_content"`
	SourceURL      string   `json:"source_url"`
	SuggestedTags  []string `json:"suggested_tags"`
	Domain         string   `json:"domain"`
	NodeCount      int      `json:"node_count"`
}

// Server implements the ResearcherServer gRPC interface.
type Server struct {
	pb.UnimplementedResearcherServer

	scraper    *Scraper
	pubsubCli  *pubsub.Client
	gcsBucket  string
	project    string

	// Pub/Sub topic: wisdom.knowledge.ingested
	ingestedTopic *pubsub.Topic
}

// NewServer initializes the Researcher gRPC server with Pub/Sub publisher.
func NewServer(gcsBucket, gcpProject string) (*Server, error) {
	ctx := context.Background()

	psCli, err := pubsub.NewClient(ctx, gcpProject)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %w", err)
	}

	return &Server{
		scraper:       NewScraper(),
		pubsubCli:     psCli,
		gcsBucket:     gcsBucket,
		project:       gcpProject,
		ingestedTopic: psCli.Topic("wisdom.knowledge.ingested"),
	}, nil
}

// Investigate triggers a research job on a given topic (streaming RPC).
func (s *Server) Investigate(req *pb.InvestigateRequest, stream pb.Researcher_InvestigateServer) error {
	if req.Topic == "" || req.UserId == "" {
		return status.Error(codes.InvalidArgument, "topic and user_id are required")
	}

	ctx := stream.Context()
	jobID := uuid.NewString()
	log.Printf("Researcher job %s started: topic=%q domain=%q", jobID, req.Topic, req.Domain)

	// TODO: Implement multi-source research pipeline:
	// 1. Query the web for top N sources about req.Topic.
	// 2. Scrape each source, convert to Markdown.
	// 3. Stream each signal back via the gRPC stream.
	// 4. Publish wisdom.knowledge.ingested event at completion.

	// Placeholder: scrape a basic web query result.
	signal := &pb.ResearchSignal{
		JobId:     jobID,
		Source:    fmt.Sprintf("https://en.wikipedia.org/wiki/%s", req.Topic),
		Content:   fmt.Sprintf("# %s\n\nResearch placeholder — implementation pending.", req.Topic),
		NodeType:  "Signal",
		Tags:      []string{req.Domain, req.Topic},
		ScrapedAt: nil, // TODO: use timestamppb.Now()
	}
	if err := stream.Send(signal); err != nil {
		return status.Errorf(codes.Internal, "stream.Send: %v", err)
	}

	// Publish Pub/Sub event when job completes.
	data := IngestedData{
		Title:          req.Topic,
		MarkdownContent: signal.Content,
		SourceURL:      signal.Source,
		SuggestedTags:  signal.Tags,
		Domain:         req.Domain,
		NodeCount:      1,
	}
	if err := s.publishIngestedEvent(ctx, data); err != nil {
		log.Printf("WARN: failed to publish knowledge.ingested for job %s: %v", jobID, err)
	}

	return nil
}

// SubscribeFeed adds an RSS/Atom URL to the autonomous crawler.
func (s *Server) SubscribeFeed(ctx context.Context, req *pb.FeedRequest) (*pb.FeedAck, error) {
	if req.Url == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "url and user_id are required")
	}
	// TODO: Store feed URL in Cortex; the crawler daemon polls subscribed feeds.
	feedID := uuid.NewString()
	log.Printf("Feed subscribed: %s (id=%s)", req.Url, feedID)
	return &pb.FeedAck{FeedId: feedID, Success: true}, nil
}

// IngestBook queues a PDF/book for background processing.
func (s *Server) IngestBook(ctx context.Context, req *pb.BookRequest) (*pb.BookAck, error) {
	if req.SourceUri == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "source_uri and user_id are required")
	}
	jobID := uuid.NewString()
	log.Printf("Book ingestion queued: %s (job=%s)", req.SourceUri, jobID)
	// TODO: Upload to GCS ingestion bucket, trigger Cloud Run Job for OCR/extraction.
	return &pb.BookAck{JobId: jobID, Queued: true}, nil
}

// GetJobStatus returns the current status of a research job.
func (s *Server) GetJobStatus(ctx context.Context, req *pb.JobStatusRequest) (*pb.JobStatus, error) {
	if req.JobId == "" {
		return nil, status.Error(codes.InvalidArgument, "job_id is required")
	}
	// TODO: Query Cortex for job metadata node.
	return &pb.JobStatus{
		JobId:       req.JobId,
		Status:      "UNKNOWN",
		ProgressPct: 0,
	}, nil
}

// ─── Pub/Sub Publisher ────────────────────────────────────────────────────────

func (s *Server) publishIngestedEvent(ctx context.Context, data IngestedData) error {
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource("/researcher/web-scraper")
	event.SetType("wisdom.knowledge.ingested")
	event.SetTime(time.Now())
	if err := event.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return fmt.Errorf("event.SetData: %w", err)
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	result := s.ingestedTopic.Publish(ctx, &pubsub.Message{Data: eventBytes})
	_, err = result.Get(ctx)
	return err
}

// ─── Scraper (unchanged, no NATS dependency) ─────────────────────────────────

// Scraper handles deterministic extraction of web content to Markdown.
type Scraper struct {
	converter *md.Converter
	httpCli   *http.Client
}

// NewScraper initializes a new HTML-to-Markdown scraper.
func NewScraper() *Scraper {
	return &Scraper{
		converter: md.NewConverter("", true, nil),
		httpCli:   &http.Client{Timeout: 15 * time.Second},
	}
}

// ScrapeURL fetches a URL and converts its HTML body to Markdown.
func (s *Scraper) ScrapeURL(url string) (string, error) {
	resp, err := s.httpCli.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	markdown, err := s.converter.ConvertString(string(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to Markdown: %w", err)
	}

	return markdown, nil
}
