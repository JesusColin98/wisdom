package cortex

import (
	"context"
	"encoding/gob"
	"math/rand"
	"os"
	"sync"
)

func init() {
	gob.Register(&RPForestSubstrate{})
	gob.Register(&rpTree{})
	gob.Register(&rpNode{})
}

// ... rest of imports ...

// VectorSubstrate defines the interface for different vector search implementations.
type VectorSubstrate interface {
	// Add adds a vector to the index.
	Add(ctx context.Context, id string, vector []float32) error
	// Search finds the topK closest vectors.
	Search(ctx context.Context, query []float32, topK int) ([]ScoredNode, error)
}

// FlatSubstrate is the Tier 1 implementation using simple linear scan.
type FlatSubstrate struct {
	storage *Cortex
}

func (s *FlatSubstrate) Add(ctx context.Context, id string, vector []float32) error {
	return nil
}

func (s *FlatSubstrate) Search(ctx context.Context, query []float32, topK int) ([]ScoredNode, error) {
	return s.storage.linearVectorSearch(ctx, query, topK)
}

// RPForestSubstrate implements a Random Projection Forest for scalable ANN search.
type RPForestSubstrate struct {
	storage *Cortex
	Trees   []*rpTree
	mu      sync.RWMutex
	Dim     int
}

func (s *RPForestSubstrate) SetStorage(c *Cortex) {
	s.storage = c
}

type rpTree struct {
	Root *rpNode
}

type rpNode struct {
	Hyperplane []float32
	Left       *rpNode
	Right      *rpNode
	NodeIDs    []string // Only for leaf nodes
}

func NewRPForestSubstrate(storage *Cortex, numTrees int, dim int) *RPForestSubstrate {
	forest := &RPForestSubstrate{
		storage: storage,
		Trees:   make([]*rpTree, numTrees),
		Dim:     dim,
	}
	for i := range forest.Trees {
		forest.Trees[i] = &rpTree{}
	}
	return forest
}

func (s *RPForestSubstrate) Add(ctx context.Context, id string, vector []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tree := range s.Trees {
		tree.Root = s.addToNode(tree.Root, id, vector, 0)
	}
	return nil
}

func (s *RPForestSubstrate) addToNode(n *rpNode, id string, vector []float32, depth int) *rpNode {
	if n == nil {
		return &rpNode{NodeIDs: []string{id}}
	}

	// Leaf node behavior: if small enough or too deep, just append
	const maxLeafSize = 10
	const maxDepth = 20
	if n.Hyperplane == nil {
		if len(n.NodeIDs) < maxLeafSize || depth >= maxDepth {
			n.NodeIDs = append(n.NodeIDs, id)
			return n
		}
		// Split leaf into internal node
		n.Hyperplane = make([]float32, s.Dim)
		for i := range n.Hyperplane {
			n.Hyperplane[i] = rand.Float32()*2 - 1
		}
		oldIDs := n.NodeIDs
		n.NodeIDs = nil
		
		// Redistribute old IDs
		for _, oldID := range oldIDs {
			vec, _, err := s.storage.GetVector(context.Background(), oldID)
			if err != nil || len(vec) == 0 {
				continue // Should not happen if data is consistent
			}
			if dotProduct(vec, n.Hyperplane) > 0 {
				n.Right = s.addToNode(n.Right, oldID, vec, depth+1)
			} else {
				n.Left = s.addToNode(n.Left, oldID, vec, depth+1)
			}
		}
		// Finally, add the new one
		if dotProduct(vector, n.Hyperplane) > 0 {
			n.Right = s.addToNode(n.Right, id, vector, depth+1)
		} else {
			n.Left = s.addToNode(n.Left, id, vector, depth+1)
		}
		return n
	}

	if dotProduct(vector, n.Hyperplane) > 0 {
		n.Right = s.addToNode(n.Right, id, vector, depth+1)
	} else {
		n.Left = s.addToNode(n.Left, id, vector, depth+1)
	}
	return n
}

func (s *RPForestSubstrate) Search(ctx context.Context, query []float32, topK int) ([]ScoredNode, error) {
	s.mu.RLock()
	candidates := make(map[string]struct{})
	for _, tree := range s.Trees {
		s.collectCandidates(tree.Root, query, candidates)
	}
	s.mu.RUnlock()

	// If too few candidates, fallback to flat search or just return what we have
	if len(candidates) < topK {
		return s.storage.linearVectorSearch(ctx, query, topK)
	}

	// Rank candidates
	var ids []string
	for id := range candidates {
		ids = append(ids, id)
	}
	
	// We need to fetch vectors for these IDs and score them.
	// For simplicity, we'll use a specialized SQL query for these specific IDs.
	return s.storage.scoreSpecificNodes(ctx, query, ids, topK)
}

func (s *RPForestSubstrate) Save(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(s)
}

func (s *RPForestSubstrate) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewDecoder(f).Decode(s)
}

func (s *RPForestSubstrate) collectCandidates(n *rpNode, query []float32, candidates map[string]struct{}) {
	if n == nil {
		return
	}
	if n.Hyperplane == nil {
		for _, id := range n.NodeIDs {
			candidates[id] = struct{}{}
		}
		return
	}
	if dotProduct(query, n.Hyperplane) > 0 {
		s.collectCandidates(n.Right, query, candidates)
	} else {
		s.collectCandidates(n.Left, query, candidates)
	}
}

func dotProduct(a, b []float32) float32 {
	var res float32
	for i := range a {
		res += a[i] * b[i]
	}
	return res
}
