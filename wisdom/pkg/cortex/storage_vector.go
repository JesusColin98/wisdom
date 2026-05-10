package cortex

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/google/wisdom/pkg/observability"
	"golang.org/x/sync/errgroup"
)

// PutVector stores a semantic embedding for a node using Direct Insert logic.
func (c *Cortex) PutVector(ctx context.Context, nodeID string, embedding []float32, modelVersion string) error {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.PutVector")
	defer span.End()

	blob := floatsToBytes(embedding)
	query := `
		INSERT INTO vectors (node_id, embedding, model_version, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(node_id) DO UPDATE SET
			embedding = excluded.embedding,
			model_version = excluded.model_version,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := c.db.ExecContext(ctx, query, nodeID, blob, modelVersion)
	if err != nil {
		return fmt.Errorf("failed to persist vector: %w", err)
	}

	// OdinANN Mandate: Real-time update of the in-memory/on-disk substrate
	if c.substrate != nil {
		// Use Direct Insert logic to avoid full index rebuild
		err = c.substrate.Add(ctx, nodeID, embedding)
		if err != nil {
			observability.Logger.Warn("Substrate Add failed, falling back to lazy rebuild", "error", err)
		}

		// Stability Check: Save periodically to disk
		if forest, ok := c.substrate.(*RPForestSubstrate); ok {
			var count int
			_ = c.db.QueryRow("SELECT COUNT(*) FROM vectors").Scan(&count)
			if count > 0 && count%50 == 0 {
				go func() {
					_ = forest.Save(c.indexPath)
					observability.Logger.Info("Substrate index checkpoint saved", "count", count)
				}()
			}
		}
	}

	// Async promotion check
	go func() {
		var count int
		_ = c.db.QueryRow("SELECT COUNT(*) FROM vectors").Scan(&count)
		if count >= HNSWThreshold {
			if _, ok := c.substrate.(*FlatSubstrate); ok {
				_ = c.PromoteSubstrate(context.Background())
			}
		}
	}()

	return nil
}
// GetVector retrieves the semantic embedding for a node.
func (c *Cortex) GetVector(ctx context.Context, nodeID string) ([]float32, string, error) {
	query := `SELECT embedding, model_version FROM vectors WHERE node_id = ?`
	row := c.db.QueryRowContext(ctx, query, nodeID)

	var blob []byte
	var modelVersion string
	if err := row.Scan(&blob, &modelVersion); err != nil {
		if err == sql.ErrNoRows {
			return nil, "", nil
		}
		return nil, "", err
	}

	return bytesToFloats(blob), modelVersion, nil
}

// VectorSearch performs a semantic similarity search across all nodes.
func (c *Cortex) VectorSearch(ctx context.Context, queryEmbedding []float32, topK int) ([]ScoredNode, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.VectorSearch")
	defer span.End()

	if queryEmbedding == nil {
		return nil, nil
	}

	if c.substrate != nil {
		return c.substrate.Search(ctx, queryEmbedding, topK)
	}

	return c.linearVectorSearch(ctx, queryEmbedding, topK)
}

func (c *Cortex) linearVectorSearch(ctx context.Context, queryEmbedding []float32, topK int) ([]ScoredNode, error) {
	query := `
		SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.impact_score, n.stratum, n.source_mime_type, n.external_links, n.created_at, n.updated_at, v.embedding
		FROM nodes n
		JOIN vectors v ON n.id = v.node_id
	`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ScoredNode
	for rows.Next() {
		var sn ScoredNode
		var metadataRaw, linksRaw []byte
		var embeddingRaw []byte
		err := rows.Scan(
			&sn.ID, &sn.Content, &sn.EntityClass, &sn.Author, &sn.SourceType,
			&sn.SourceRef, &sn.NamespaceID, &metadataRaw, &sn.ConfidenceScore,
			&sn.ImpactScore, &sn.Stratum, &sn.SourceMimeType, &linksRaw, &sn.CreatedAt, &sn.UpdatedAt, &embeddingRaw,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataRaw, &sn.Metadata); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(linksRaw, &sn.ExternalLinks); err != nil {
			return nil, err
		}

		nodeEmbedding := bytesToFloats(embeddingRaw)
		sn.Score = cosineSimilarity(queryEmbedding, nodeEmbedding)
		results = append(results, sn)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// HybridSearch combines Keyword and Vector search using Reciprocal Rank Fusion (RRF).
func (c *Cortex) HybridSearch(ctx context.Context, keywordQuery string, queryEmbedding []float32, topK int) ([]ScoredNode, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.HybridSearch")
	defer span.End()

	var keywordResults []Node
	var vectorResults []ScoredNode
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		keywordResults, err = c.SearchNodes(gCtx, keywordQuery)
		return err
	})

	g.Go(func() error {
		var err error
		vectorResults, err = c.VectorSearch(gCtx, queryEmbedding, topK*2)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	const k = 60.0
	fusedScores := make(map[string]float64)
	nodesMap := make(map[string]Node)

	for rank, node := range keywordResults {
		fusedScores[node.ID] += 1.0 / (k + float64(rank+1))
		nodesMap[node.ID] = node
	}

	for rank, sn := range vectorResults {
		fusedScores[sn.ID] += 1.0 / (k + float64(rank+1))
		nodesMap[sn.ID] = sn.Node
	}

	var finalResults []ScoredNode
	for id, score := range fusedScores {
		finalResults = append(finalResults, ScoredNode{
			Node:  nodesMap[id],
			Score: score,
		})
	}

	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].Score > finalResults[j].Score
	})

	if len(finalResults) > topK {
		finalResults = finalResults[:topK]
	}

	return finalResults, nil
}

func (c *Cortex) scoreSpecificNodes(ctx context.Context, queryEmbedding []float32, ids []string, topK int) ([]ScoredNode, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := ""
	args := make([]any, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.impact_score, n.stratum, n.source_mime_type, n.external_links, n.created_at, n.updated_at, v.embedding
		FROM nodes n
		JOIN vectors v ON n.id = v.node_id
		WHERE n.id IN (%s)
	`, placeholders)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ScoredNode
	for rows.Next() {
		var sn ScoredNode
		var metadataRaw, linksRaw []byte
		var embeddingRaw []byte
		err := rows.Scan(
			&sn.ID, &sn.Content, &sn.EntityClass, &sn.Author, &sn.SourceType,
			&sn.SourceRef, &sn.NamespaceID, &metadataRaw, &sn.ConfidenceScore,
			&sn.ImpactScore, &sn.Stratum, &sn.SourceMimeType, &linksRaw, &sn.CreatedAt, &sn.UpdatedAt, &embeddingRaw,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataRaw, &sn.Metadata); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(linksRaw, &sn.ExternalLinks); err != nil {
			return nil, err
		}

		nodeEmbedding := bytesToFloats(embeddingRaw)
		sn.Score = cosineSimilarity(queryEmbedding, nodeEmbedding)
		results = append(results, sn)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

func (c *Cortex) PromoteSubstrate(ctx context.Context) error {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.PromoteSubstrate")
	defer span.End()

	if _, ok := c.substrate.(*RPForestSubstrate); ok {
		return nil
	}

	observability.Logger.Info("Promoting substrate to RPForest Tier")

	query := `SELECT node_id, embedding FROM vectors`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	forest := NewRPForestSubstrate(c, 10, 768) 
	
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			continue
		}
		vec := bytesToFloats(blob)
		if len(vec) > 0 {
			forest.Dim = len(vec)
			_ = forest.Add(ctx, id, vec)
		}
	}

	c.substrate = forest
	return forest.Save(c.indexPath)
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (c *Cortex) FindSimilar(ctx context.Context, embedding []float32, threshold float64) (*ScoredNode, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.FindSimilar")
	defer span.End()

	results, err := c.VectorSearch(ctx, embedding, 1)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 && results[0].Score >= threshold {
		return &results[0], nil
	}

	return nil, nil
}
