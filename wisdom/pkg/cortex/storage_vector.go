package cortex

import (
	"context"
	"sort"

	"github.com/google/wisdom/pkg/observability"
	"golang.org/x/sync/errgroup"
)

// PutVector stores a semantic embedding for a node.
func (c *Cortex) PutVector(ctx context.Context, nodeID string, embedding []float32, modelVersion string) error {
	return c.engine.PutVector(ctx, nodeID, embedding, modelVersion)
}

// GetVector retrieves the semantic embedding for a node.
func (c *Cortex) GetVector(ctx context.Context, nodeID string) ([]float32, string, error) {
	return c.engine.GetVector(ctx, nodeID)
}

// VectorSearch performs a semantic similarity search across all nodes.
func (c *Cortex) VectorSearch(ctx context.Context, queryEmbedding []float32, topK int) ([]ScoredNode, error) {
	return c.engine.SearchVectors(ctx, queryEmbedding, topK)
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

func (c *Cortex) PromoteSubstrate(ctx context.Context) error {
	// This is engine specific, but for now we delegate it if possible
	if sqlite, ok := c.engine.(*SQLiteEngine); ok {
		return sqlite.PromoteSubstrate(ctx)
	}
	return nil
}

func (c *Cortex) FindSimilar(ctx context.Context, embedding []float32, threshold float64) (*ScoredNode, error) {
	results, err := c.VectorSearch(ctx, embedding, 1)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 && results[0].Score >= threshold {
		return &results[0], nil
	}
	return nil, nil
}

