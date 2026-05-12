package researcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// IngestedData represents the payload to be sent over NATS.
type IngestedData struct {
	Title           string   `json:"title"`
	MarkdownContent string   `json:"markdown_content"`
	SourceURL       string   `json:"source_url"`
	SuggestedTags   []string `json:"suggested_tags"`
}

// Scraper handles deterministic extraction of web content.
type Scraper struct {
	converter *md.Converter
}

// NewScraper initializes a new HTML-to-Markdown scraper.
func NewScraper() *Scraper {
	return &Scraper{
		converter: md.NewConverter("", true, nil),
	}
}

// ScrapeURL fetches a URL and converts its body to Markdown.
func (s *Scraper) ScrapeURL(url string) (string, error) {
	resp, err := http.Get(url)
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

// Publisher handles sending CloudEvents to NATS JetStream.
type Publisher struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

// NewPublisher initializes a NATS publisher.
func NewPublisher(natsURL string) (*Publisher, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return &Publisher{
		nc: nc,
		js: js,
	}, nil
}

// PublishIngestedEvent constructs and sends a CloudEvent.
func (p *Publisher) PublishIngestedEvent(data IngestedData) error {
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource("/researcher/web-scraper")
	event.SetType("wisdom.knowledge.ingested")
	event.SetTime(time.Now())

	if err := event.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return fmt.Errorf("failed to set event data: %w", err)
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal CloudEvent: %w", err)
	}

	_, err = p.js.Publish("wisdom.knowledge.ingested", eventBytes)
	if err != nil {
		return fmt.Errorf("failed to publish to NATS: %w", err)
	}

	return nil
}

// Close closes the NATS connection.
func (p *Publisher) Close() {
	p.nc.Close()
}
