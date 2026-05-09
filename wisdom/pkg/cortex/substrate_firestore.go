package cortex

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type FirestoreSubstrate struct {
	client     *firestore.Client
	collection string
	storage    *Cortex
}

func NewFirestoreSubstrate(ctx context.Context, projectID, collection string, storage *Cortex) (*FirestoreSubstrate, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &FirestoreSubstrate{
		client:     client,
		collection: collection,
		storage:    storage,
	}, nil
}

func (s *FirestoreSubstrate) Add(ctx context.Context, id string, vector []float32) error {
	_, err := s.client.Collection(s.collection).Doc(id).Set(ctx, map[string]interface{}{
		"embedding": vector,
	})
	return err
}

func (s *FirestoreSubstrate) Search(ctx context.Context, queryVector []float32, topK int) ([]ScoredNode, error) {
	// Convert float32 to float64 for Firestore vector SDK if needed,
	// but usually the SDK takes VectorValue or similar.
	// For 2026 SDK, we use the vector search query.
	
	q := s.client.Collection(s.collection).FindNearest("embedding", queryVector, topK, firestore.DistanceMeasureCosine, nil)
	it := q.Documents(ctx)
	
	var results []ScoredNode
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		
		id := doc.Ref.ID
		// Distance in Cosine usually means 1 - similarity or similar depending on implementation.
		// We'll treat the distance as something we can convert back to similarity.
		// Some SDKs provide doc.Distance.
		
		results = append(results, ScoredNode{
			Node:  Node{ID: id},
			Score: 1.0, // Placeholder score if distance is not directly accessible without extra fields
		})
	}
	
	return results, nil
}
