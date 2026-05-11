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
	engine StorageEngine
}

func (s *FlatSubstrate) Add(ctx context.Context, id string, vector []float32) error {
	return nil
}

func (s *FlatSubstrate) Search(ctx context.Context, query []float32, topK int) ([]ScoredNode, error) {
	// Re-route to engine's linear search logic
	if sqlite, ok := s.engine.(*SQLiteEngine); ok {
		return sqlite.linearVectorSearch(ctx, query, topK)
	}
	return nil, fmt.Errorf("flat search only supported for SQLiteEngine")
}

// RPForestSubstrate implements a Random Projection Forest for scalable ANN search.
type RPForestSubstrate struct {
	engine StorageEngine
	Trees  []*rpTree
	mu     sync.RWMutex
	Dim    int
}

func (s *RPForestSubstrate) SetEngine(e StorageEngine) {
	s.engine = e
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

func NewRPForestSubstrate(engine StorageEngine, numTrees int, dim int) *RPForestSubstrate {
	forest := &RPForestSubstrate{
		engine: engine,
		Trees:  make([]*rpTree, numTrees),
		Dim:    dim,
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
		// Split leaf
		if len(n.NodeIDs) >= 2 {
			idx1 := rand.Intn(len(n.NodeIDs))
			idx2 := rand.Intn(len(n.NodeIDs))
			for idx1 == idx2 {
				idx2 = rand.Intn(len(n.NodeIDs))
			}
			
			vec1, _, _ := s.engine.GetVector(context.Background(), n.NodeIDs[idx1])
			vec2, _, _ := s.engine.GetVector(context.Background(), n.NodeIDs[idx2])
			
			if len(vec1) > 0 && len(vec2) > 0 {
				n.Hyperplane = make([]float32, s.Dim)
				for i := range n.Hyperplane {
					n.Hyperplane[i] = vec1[i] - vec2[i]
				}
			}
		}

		if n.Hyperplane == nil {
			n.Hyperplane = make([]float32, s.Dim)
			for i := range n.Hyperplane {
				n.Hyperplane[i] = rand.Float32()*2 - 1
			}
		}
		oldIDs := n.NodeIDs
		n.NodeIDs = nil
		
		for _, oldID := range oldIDs {
			vec, _, err := s.engine.GetVector(context.Background(), oldID)
			if err != nil || len(vec) == 0 {
				continue 
			}
			if dotProduct(vec, n.Hyperplane) > 0 {
				n.Right = s.addToNode(n.Right, oldID, vec, depth+1)
			} else {
				n.Left = s.addToNode(n.Left, oldID, vec, depth+1)
			}
		}
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

	if sqlite, ok := s.engine.(*SQLiteEngine); ok {
		if len(candidates) < topK {
			return sqlite.linearVectorSearch(ctx, query, topK)
		}

		var ids []string
		for id := range candidates {
			ids = append(ids, id)
		}
		return sqlite.scoreSpecificNodes(ctx, query, ids, topK)
	}
	
	return nil, fmt.Errorf("RPForest search only supported for SQLiteEngine")
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
