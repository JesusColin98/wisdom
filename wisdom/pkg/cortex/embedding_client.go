package cortex

// embedding_client.go — Vertex AI text-embedding-004 client for Cortex.
//
// Generates 768-dimensional dense embeddings for semantic search (pgvector).
// Uses the Vertex AI textembedding-gecko / text-embedding-004 model.
//
// Fallback chain:
//   Vertex AI text-embedding-004 → error (embedding stored as NULL, JSONB search used instead)

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2/google"
)

const (
	// EmbeddingModel is the Vertex AI embedding model.
	// text-embedding-004 outputs 768-dimensional vectors (same as pgvector column).
	EmbeddingModel = "text-embedding-004"

	// EmbeddingDim must match the vector(768) column in schema_v3_pgvector.sql.
	EmbeddingDim = 768

	// embeddingMaxChars is the max input length for text-embedding-004.
	embeddingMaxChars = 2048
)

// EmbeddingClient generates text embeddings via Vertex AI.
type EmbeddingClient struct {
	project    string
	region     string
	httpClient *http.Client
}

// vertexEmbedRequest is the Vertex AI prediction request payload.
type vertexEmbedRequest struct {
	Instances []vertexEmbedInstance `json:"instances"`
}

type vertexEmbedInstance struct {
	Content  string `json:"content"`
	TaskType string `json:"task_type"` // RETRIEVAL_DOCUMENT or RETRIEVAL_QUERY
}

type vertexEmbedResponse struct {
	Predictions []struct {
		Embeddings struct {
			Values []float32 `json:"values"`
		} `json:"embeddings"`
	} `json:"predictions"`
}

// NewEmbeddingClient creates an embedding client using Application Default Credentials.
func NewEmbeddingClient(project, region string) (*EmbeddingClient, error) {
	if project == "" {
		project = os.Getenv("GCP_PROJECT_ID")
	}
	if region == "" {
		region = os.Getenv("GCP_REGION")
		if region == "" {
			region = "us-central1"
		}
	}

	// Use ADC (Application Default Credentials) — works in Cloud Run and local dev.
	ts, err := google.DefaultTokenSource(context.Background(),
		"https://www.googleapis.com/auth/cloud-platform",
	)
	if err != nil {
		return nil, fmt.Errorf("google.DefaultTokenSource: %w", err)
	}

	return &EmbeddingClient{
		project: project,
		region:  region,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: &oauth2Transport{base: http.DefaultTransport, ts: ts},
		},
	}, nil
}

// EmbedDocument embeds a document for storage (RETRIEVAL_DOCUMENT task type).
// Use for nodes being stored in Cortex.
func (c *EmbeddingClient) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return c.embed(ctx, text, "RETRIEVAL_DOCUMENT")
}

// EmbedQuery embeds a query for similarity search (RETRIEVAL_QUERY task type).
// Use when searching Cortex by semantic similarity.
func (c *EmbeddingClient) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return c.embed(ctx, text, "RETRIEVAL_QUERY")
}

func (c *EmbeddingClient) embed(ctx context.Context, text, taskType string) ([]float32, error) {
	// Truncate to model limit.
	if len(text) > embeddingMaxChars {
		text = text[:embeddingMaxChars]
	}

	endpoint := fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:predict",
		c.region, c.project, c.region, EmbeddingModel,
	)

	reqBody := vertexEmbedRequest{
		Instances: []vertexEmbedInstance{
			{Content: text, TaskType: taskType},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vertex embed request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vertex embed HTTP %d: %s", resp.StatusCode, string(body))
	}

	var embResp vertexEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}

	if len(embResp.Predictions) == 0 || len(embResp.Predictions[0].Embeddings.Values) == 0 {
		return nil, fmt.Errorf("vertex embed returned no predictions")
	}

	vec := embResp.Predictions[0].Embeddings.Values
	if len(vec) != EmbeddingDim {
		return nil, fmt.Errorf("unexpected embedding dimension: got %d, want %d", len(vec), EmbeddingDim)
	}

	return vec, nil
}

// TextFromNode extracts a string suitable for embedding from a Node's payload.
func TextFromNode(node *Node) string {
	if node.Payload == nil {
		return ""
	}

	var buf bytes.Buffer
	for _, key := range []string{"title", "content", "name", "question", "answer", "description"} {
		if v, ok := node.Payload[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				buf.WriteString(s)
				buf.WriteByte(' ')
			}
		}
	}
	// Append domain for domain-scoped retrieval.
	if domain, ok := node.Payload["domain"].(string); ok {
		buf.WriteString(domain)
	}

	return buf.String()
}

// ── oauth2Transport ────────────────────────────────────────────────────────────

type oauth2Transport struct {
	base http.RoundTripper
	ts   interface {
		Token() (*oauth2Token, error)
	}
}

// We import oauth2 token type inline to avoid full dependency.
type oauth2Token struct {
	AccessToken string
}

func (t *oauth2Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone to avoid mutating caller's request.
	r := req.Clone(req.Context())
	if tok, err := t.ts.Token(); err == nil {
		r.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	} else {
		log.Printf("WARN: failed to get OAuth2 token: %v", err)
	}
	return t.base.RoundTrip(r)
}
