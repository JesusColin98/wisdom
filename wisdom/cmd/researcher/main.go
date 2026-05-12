package main

import (
	"log"
	"os"
	"strings"

	"github.com/google/wisdom/pkg/researcher"
)

func main() {
	// The Researcher can be run as a Job (e.g., Cloud Run Job) taking URLs from environment
	// variables or a message queue. For simplicity, we accept a TARGET_URLS env var.
	targetURLsStr := os.Getenv("TARGET_URLS")
	if targetURLsStr == "" {
		log.Fatal("TARGET_URLS environment variable is required (comma-separated)")
	}
	urls := strings.Split(targetURLsStr, ",")

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	publisher, err := researcher.NewPublisher(natsURL)
	if err != nil {
		log.Fatalf("Failed to initialize NATS publisher: %v", err)
	}
	defer publisher.Close()

	scraper := researcher.NewScraper()

	for _, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		log.Printf("Scraping %s...", url)
		markdown, err := scraper.ScrapeURL(url)
		if err != nil {
			log.Printf("Error scraping %s: %v", url, err)
			continue
		}

		// Prepare the data
		data := researcher.IngestedData{
			Title:           "Auto-scraped Content", // In a real app, parse the <title> tag
			MarkdownContent: markdown,
			SourceURL:       url,
			SuggestedTags:   []string{"auto-ingest"}, // Real app: basic keyword extraction
		}

		// Publish to NATS
		if err := publisher.PublishIngestedEvent(data); err != nil {
			log.Printf("Failed to publish event for %s: %v", url, err)
		} else {
			log.Printf("Successfully published CloudEvent for %s", url)
		}
	}
	
	log.Println("Research job completed successfully.")
}
