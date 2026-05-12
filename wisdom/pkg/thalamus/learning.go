package thalamus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// LearningPath represents a structured curriculum.
type LearningPath struct {
	Topic       string   `json:"topic"`
	Description string   `json:"description"`
	Modules     []Module `json:"modules"`
}

// Module represents a logical unit of learning.
type Module struct {
	Title         string   `json:"title"`
	Concepts      []string `json:"concepts"`
	Prerequisites []string `json:"prerequisites"`
}

// LearningEngine orchestrates proactive learning path generation.
type LearningEngine struct {
	Cortex   *cortex.Cortex
	Cerebellum *cerebellum.Runner
	Coach    *Coach
	LLM      cerebellum.LLMProvider
}

// NewLearningEngine initializes the learning engine.
func NewLearningEngine(cx *cortex.Cortex, cb *cerebellum.Runner, coach *Coach, llm cerebellum.LLMProvider) *LearningEngine {
	return &LearningEngine{
		Cortex:   cx,
		Cerebellum: cb,
		Coach:    coach,
		LLM:      llm,
	}
}

// GenerateFromTopic creates a learning path by searching the web.
func (e *LearningEngine) GenerateFromTopic(ctx context.Context, topic string, userID string) (*LearningPath, error) {
	ctx, span := observability.Tracer.Start(ctx, "LearningEngine.GenerateFromTopic")
	defer span.End()

	// 1. Get user context from Coach (what do they already know/struggle with?)
	weaknesses, _ := e.Coach.DiscoverWeaknesses(ctx, userID)
	mastered, _ := e.Coach.ListMasteredNodes(ctx, userID)

	contextSummary := e.buildContextSummary(weaknesses, mastered)

	// 2. Proactive Search using cerebellum tool
	params, _ := json.Marshal(map[string]string{"query": fmt.Sprintf("learning roadmap for %s", topic)})
	job, err := e.Cerebellum.Run(ctx, "web_search", params)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	researchData, _ := json.Marshal(job.Result.Output)

	// 3. Formulate Path
	pathPrompt := fmt.Sprintf(`Generate a structured JSON learning path for "%s".
User Context: %s

Guidelines:
- Modules should be sequential.
- Concepts should be atomic.
- Adapt the depth based on the user context (e.g., if they struggle with basics, add more foundational modules).

Respond ONLY with JSON matching this schema:
{
  "topic": string,
  "description": string,
  "modules": [
    { "title": string, "concepts": [string], "prerequisites": [string] }
  ]
}

Research Data from Web Search:
%s`, topic, contextSummary, string(researchData))

	response, err := e.LLM.Complete(ctx, pathPrompt)
	if err != nil {
		return nil, err
	}

	var path LearningPath
	if err := json.Unmarshal([]byte(e.extractJSON(response)), &path); err != nil {
		return nil, fmt.Errorf("failed to parse path JSON: %w", err)
	}

	// 4. Anchor to Cortex
	if err := e.anchorPathToCortex(ctx, &path, userID); err != nil {
		return nil, err
	}

	return &path, nil
}

func (e *LearningEngine) buildContextSummary(weaknesses []Weakness, mastered []cortex.Node) string {
	var sb strings.Builder
	sb.WriteString("User struggles with: ")
	for _, w := range weaknesses {
		sb.WriteString(w.Content + ", ")
	}
	sb.WriteString(". User has mastered: ")
	for _, m := range mastered {
		sb.WriteString(m.Content + ", ")
	}
	return sb.String()
}

// GenerateFromYouTube creates a learning path from a video transcript.
func (e *LearningEngine) GenerateFromYouTube(ctx context.Context, url string, userID string) (*LearningPath, error) {
	ctx, span := observability.Tracer.Start(ctx, "LearningEngine.GenerateFromYouTube")
	defer span.End()

	// 1. Extract transcript using cerebellum tool
	params, _ := json.Marshal(map[string]string{"url": url})
	job, err := e.Cerebellum.Run(ctx, "youtube_transcript", params)
	if err != nil {
		return nil, fmt.Errorf("transcript extraction failed: %w", err)
	}

	transcriptData := job.Result.Output.(map[string]string)["transcript"]
	
	// 2. Formulate Path
	return e.generatePathFromContent(ctx, "Video Resource", transcriptData, userID)
}

// GenerateFromDocument creates a learning path from a document.
func (e *LearningEngine) GenerateFromDocument(ctx context.Context, title string, content string, userID string) (*LearningPath, error) {
	ctx, span := observability.Tracer.Start(ctx, "LearningEngine.GenerateFromDocument")
	defer span.End()

	return e.generatePathFromContent(ctx, title, content, userID)
}

func (e *LearningEngine) generatePathFromContent(ctx context.Context, sourceName string, content string, userID string) (*LearningPath, error) {
	weaknesses, _ := e.Coach.DiscoverWeaknesses(ctx, userID)
	mastered, _ := e.Coach.ListMasteredNodes(ctx, userID)
	contextSummary := e.buildContextSummary(weaknesses, mastered)

	prompt := fmt.Sprintf(`Generate a structured JSON learning path based on this content: "%s"
User Context: %s

Respond ONLY with JSON matching this schema:
{
  "topic": string,
  "description": string,
  "modules": [
    { "title": string, "concepts": [string], "prerequisites": [string] }
  ]
}

Content:
%s`, sourceName, contextSummary, content)

	response, err := e.LLM.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var path LearningPath
	if err := json.Unmarshal([]byte(e.extractJSON(response)), &path); err != nil {
		return nil, err
	}

	_ = e.anchorPathToCortex(ctx, &path, userID)
	return &path, nil
}

func (e *LearningEngine) anchorPathToCortex(ctx context.Context, path *LearningPath, userID string) error {
	// Create Topic Node
	topicNode := &cortex.Node{
		ID:          fmt.Sprintf("path-%s", strings.ReplaceAll(strings.ToLower(path.Topic), " ", "-")),
		Content:     path.Topic,
		EntityClass: "LEARNING_PATH",
		NamespaceID: "ns-learning",
		Metadata: map[string]any{
			"description": path.Description,
		},
	}
	_ = e.Cortex.PutNode(ctx, topicNode)

	for i, mod := range path.Modules {
		modNode := &cortex.Node{
			ID:          fmt.Sprintf("%s-mod-%d", topicNode.ID, i),
			Content:     mod.Title,
			EntityClass: "LEARNING_MODULE",
			NamespaceID: "ns-learning",
		}
		_ = e.Cortex.PutNode(ctx, modNode)
		_ = e.Cortex.PutLink(ctx, topicNode.ID, modNode.ID, "HAS_MODULE", 1.0)

		for _, concept := range mod.Concepts {
			conceptNode := &cortex.Node{
				ID:          fmt.Sprintf("%s-concept-%s", modNode.ID, strings.ReplaceAll(strings.ToLower(concept), " ", "-")),
				Content:     concept,
				EntityClass: "LEARNING_CONCEPT",
				NamespaceID: "ns-learning",
			}
			_ = e.Cortex.PutNode(ctx, conceptNode)
			_ = e.Cortex.PutLink(ctx, modNode.ID, conceptNode.ID, "HAS_CONCEPT", 1.0)
		}
	}
	return nil
}
