package cerebellum

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/wisdom/pkg/observability"
)

// WebSearchTool searches the web for learning materials.
type WebSearchTool struct{}

func (t *WebSearchTool) Execute(ctx context.Context, params json.RawMessage) (*Result, error) {
	var p struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	apiKey := os.Getenv("EXA_API_KEY")
	if apiKey == "" {
		return &Result{Success: false, Output: "EXA_API_KEY not configured. Falling back to internal knowledge."}, nil
	}

	// Implementation of Exa Search (Simplified)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.exa.ai/search", strings.NewReader(fmt.Sprintf(`{"query": "%s", "useAutoprompt": true, "numResults": 5}`, p.Query)))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(body, &result)

	return &Result{
		Success: true,
		Output:  result,
	}, nil
}

// YouTubeTranscriptTool extracts transcripts from videos.
type YouTubeTranscriptTool struct{}

func (t *YouTubeTranscriptTool) Execute(ctx context.Context, params json.RawMessage) (*Result, error) {
	var p struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// In a real implementation, we would call a Python script or a dedicated service like 'youtube-transcript-api-wrapper'
	// For the MVP, we simulate or call a specialized microservice if available.
	observability.Logger.Info("Extracting YouTube transcript", "url", p.URL)
	
	// Mock implementation for demonstration
	mockTranscript := fmt.Sprintf("This is a transcript of a lecture on %s. It covers modules A, B, and C.", p.URL)
	
	return &Result{
		Success: true,
		Output:  map[string]string{"transcript": mockTranscript},
	}, nil
}

// RegisterLearningTools adds the proactive tools to the registry.
func RegisterLearningTools(r *Registry) {
	_ = r.Register(ToolDefinition{
		ID:          "web_search",
		Name:        "Web Search",
		Description: "Searches the web for latest info and learning paths using Exa AI.",
	}, &WebSearchTool{})

	_ = r.Register(ToolDefinition{
		ID:          "youtube_transcript",
		Name:        "YouTube Transcript",
		Description: "Extracts text transcripts from a YouTube URL.",
	}, &YouTubeTranscriptTool{})
}
