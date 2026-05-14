package cortex

// pgvector_engine.go — Semantic search layer for Cortex using pgvector.
//
// Extends PostgresEngine with:
//   1. StoreEmbedding — generate and persist a vector embedding for a node
//   2. SemanticSearch — ANN search using cosine distance (HNSW index)
//   3. HybridSearch   — combines pgvector ANN + full-text ts_content (RRF fusion)
//
// Fallback strategy:
//   If Vertex AI embedding generation fails, SemanticSearch falls back to
//   JSONB full-text search (existing QueryFacts behavior) so the system
//   degrades gracefully without crashing.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// SemanticSearchResult extends Node with a relevance score.
type SemanticSearchResult struct {
	*Node
	Score float64 `json:"score"` // Cosine similarity 0.0–1.0 (higher = more similar).
	Mode  string  `json:"mode"`  // "vector", "fulltext", or "hybrid".
}

// SemanticSearchRequest defines the parameters for a semantic search.
type SemanticSearchRequest struct {
	Query        string            // Natural language query.
	Limit        int               // Max results to return (default 10).
	DomainFilter string            // Optional: restrict to a domain (e.g. "CHESS").
	TypeFilter   string            // Optional: restrict to a node type (e.g. "Algorithm").
	MinScore     float64           // Minimum similarity threshold (0.0–1.0, default 0.5).
	Metadata     map[string]string // Optional JSONB metadata filters.
}

// StoreEmbedding generates and persists an embedding vector for a node.
// Called asynchronously after Memorize — never blocks the write path.
func (e *PostgresEngine) StoreEmbedding(ctx context.Context, nodeID string, embClient *EmbeddingClient) error {
	// Fetch the node to get its text content.
	node, err := e.GetNode(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("StoreEmbedding: GetNode %s: %w", nodeID, err)
	}
	if node == nil {
		return fmt.Errorf("StoreEmbedding: node %s not found", nodeID)
	}

	text := TextFromNode(node)
	if strings.TrimSpace(text) == "" {
		log.Printf("cortex: node %s has no embeddable text, skipping", nodeID)
		return nil
	}

	// Generate embedding via Vertex AI.
	vec, err := embClient.EmbedDocument(ctx, text)
	if err != nil {
		return fmt.Errorf("StoreEmbedding: EmbedDocument: %w", err)
	}

	// Persist vector as pgvector literal: '[0.1, 0.2, ...]'
	vecStr := float32SliceToPgVector(vec)

	_, err = e.db.ExecContext(ctx,
		`UPDATE nodes SET embedding = $1::vector, embedding_model = $2 WHERE id = $3`,
		vecStr, EmbeddingModel, nodeID,
	)
	if err != nil {
		return fmt.Errorf("StoreEmbedding: UPDATE: %w", err)
	}

	return nil
}

// SemanticSearch performs ANN vector search using the HNSW index.
// Falls back to JSONB full-text search if no embeddings exist.
func (e *PostgresEngine) SemanticSearch(ctx context.Context, req SemanticSearchRequest, embClient *EmbeddingClient) ([]*SemanticSearchResult, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.MinScore <= 0 {
		req.MinScore = 0.4
	}

	// Step 1: Try pgvector ANN search.
	vec, err := embClient.EmbedQuery(ctx, req.Query)
	if err != nil {
		log.Printf("cortex: embedding failed for query '%s': %v — falling back to full-text", req.Query, err)
		return e.fullTextFallback(ctx, req)
	}

	results, err := e.vectorSearch(ctx, req, vec)
	if err != nil {
		log.Printf("cortex: vector search failed: %v — falling back to full-text", err)
		return e.fullTextFallback(ctx, req)
	}

	// Step 2: If vector search returns too few results, augment with full-text.
	if len(results) < req.Limit/2 {
		ftResults, ftErr := e.fullTextFallback(ctx, req)
		if ftErr == nil {
			results = reciprocalRankFusion(results, ftResults, req.Limit)
		}
	}

	return results, nil
}

// HybridSearch combines pgvector ANN and full-text search using RRF fusion.
// Best quality for knowledge retrieval — use this in ADK Router experts.
func (e *PostgresEngine) HybridSearch(ctx context.Context, req SemanticSearchRequest, embClient *EmbeddingClient) ([]*SemanticSearchResult, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}

	vec, vecErr := embClient.EmbedQuery(ctx, req.Query)
	var vecResults []*SemanticSearchResult
	if vecErr == nil {
		vecResults, _ = e.vectorSearch(ctx, req, vec)
	}

	ftResults, _ := e.fullTextFallback(ctx, req)

	merged := reciprocalRankFusion(vecResults, ftResults, req.Limit)

	mode := "hybrid"
	if vecErr != nil {
		mode = "fulltext"
	} else if len(ftResults) == 0 {
		mode = "vector"
	}
	for _, r := range merged {
		r.Mode = mode
	}

	return merged, nil
}

// ── Internal: pgvector ANN query ──────────────────────────────────────────────

func (e *PostgresEngine) vectorSearch(ctx context.Context, req SemanticSearchRequest, vec []float32) ([]*SemanticSearchResult, error) {
	vecStr := float32SliceToPgVector(vec)

	var conditions []string
	var args []any
	args = append(args, vecStr) // $1 — query vector

	conditions = append(conditions, "embedding IS NOT NULL")

	argIdx := 2
	if req.DomainFilter != "" {
		conditions = append(conditions, fmt.Sprintf("payload->>'domain' = $%d", argIdx))
		args = append(args, req.DomainFilter)
		argIdx++
	}
	if req.TypeFilter != "" {
		conditions = append(conditions, fmt.Sprintf("type::text = $%d", argIdx))
		args = append(args, req.TypeFilter)
		argIdx++
	}
	if len(req.Metadata) > 0 {
		filterJSON, err := json.Marshal(req.Metadata)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("payload @> $%d", argIdx))
			args = append(args, string(filterJSON))
			argIdx++
		}
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Cosine distance: 1 - (embedding <=> query_vec) gives similarity.
	query := fmt.Sprintf(`
		SELECT
			id, type, payload, confidence, requires_human, ttl, created_at, updated_at,
			1 - (embedding <=> $1::vector) AS score
		FROM nodes
		%s
		AND 1 - (embedding <=> $1::vector) >= %g
		ORDER BY embedding <=> $1::vector
		LIMIT %d
	`, where, req.MinScore, req.Limit*2)

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("vectorSearch query: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows, "vector")
}

// ── Internal: full-text fallback ──────────────────────────────────────────────

func (e *PostgresEngine) fullTextFallback(ctx context.Context, req SemanticSearchRequest) ([]*SemanticSearchResult, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, nil
	}

	tsQuery := strings.Join(strings.Fields(req.Query), " & ")

	var conditions []string
	var args []any
	args = append(args, tsQuery) // $1

	conditions = append(conditions, "ts_content @@ to_tsquery('english', $1)")

	argIdx := 2
	if req.DomainFilter != "" {
		conditions = append(conditions, fmt.Sprintf("payload->>'domain' = $%d", argIdx))
		args = append(args, req.DomainFilter)
		argIdx++
	}
	if req.TypeFilter != "" {
		conditions = append(conditions, fmt.Sprintf("type::text = $%d", argIdx))
		args = append(args, req.TypeFilter)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT
			id, type, payload, confidence, requires_human, ttl, created_at, updated_at,
			ts_rank(ts_content, to_tsquery('english', $1)) AS score
		FROM nodes
		%s
		ORDER BY score DESC
		LIMIT %d
	`, where, req.Limit*2)

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("fullTextFallback query: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows, "fulltext")
}

// ── Internal: result scanning ─────────────────────────────────────────────────

func scanSearchResults(rows *sql.Rows, mode string) ([]*SemanticSearchResult, error) {
	var results []*SemanticSearchResult
	for rows.Next() {
		var node Node
		var payloadRaw []byte
		var ttl sql.NullTime
		var score float64

		if err := rows.Scan(
			&node.ID, &node.Type, &payloadRaw, &node.Confidence, &node.RequiresHuman,
			&ttl, &node.CreatedAt, &node.UpdatedAt, &score,
		); err != nil {
			return nil, err
		}
		if ttl.Valid {
			node.TTL = &ttl.Time
		}
		if err := json.Unmarshal(payloadRaw, &node.Payload); err != nil {
			continue
		}
		results = append(results, &SemanticSearchResult{
			Node:  &node,
			Score: score,
			Mode:  mode,
		})
	}
	return results, rows.Err()
}

// ── Reciprocal Rank Fusion ────────────────────────────────────────────────────

// reciprocalRankFusion merges two ranked lists using the RRF formula.
// k=60 is the standard constant from the original RRF paper.
func reciprocalRankFusion(vecResults, ftResults []*SemanticSearchResult, limit int) []*SemanticSearchResult {
	const k = 60.0

	scores := make(map[string]float64)
	byID := make(map[string]*SemanticSearchResult)

	for i, r := range vecResults {
		scores[r.ID] += 1.0 / (k + float64(i+1))
		byID[r.ID] = r
	}
	for i, r := range ftResults {
		scores[r.ID] += 1.0 / (k + float64(i+1))
		if _, ok := byID[r.ID]; !ok {
			byID[r.ID] = r
		}
	}

	// Collect and sort by RRF score.
	type scored struct {
		id    string
		score float64
	}
	var sorted []scored
	for id, s := range scores {
		sorted = append(sorted, scored{id, s})
	}
	// Insertion sort (small N, typically ≤20).
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].score > sorted[j-1].score; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	var results []*SemanticSearchResult
	for _, s := range sorted {
		if len(results) >= limit {
			break
		}
		r := byID[s.id]
		r.Score = s.score
		r.Mode = "hybrid"
		results = append(results, r)
	}
	return results
}

// ── Utility ───────────────────────────────────────────────────────────────────

// float32SliceToPgVector converts a float32 slice to pgvector literal format.
// e.g. []float32{0.1, 0.2} → "[0.1,0.2]"
func float32SliceToPgVector(vec []float32) string {
	sb := strings.Builder{}
	sb.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, "%g", v)
	}
	sb.WriteByte(']')
	return sb.String()
}
