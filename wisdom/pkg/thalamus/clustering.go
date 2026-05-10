package thalamus

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// ClusteringService handles automatic namespace organization.
type ClusteringService struct {
	Cortex *cortex.Cortex
	Chat   *Chat
}

// NewClusteringService initializes a new clustering service.
func NewClusteringService(c *cortex.Cortex, chat *Chat) *ClusteringService {
	return &ClusteringService{Cortex: c, Chat: chat}
}

// ReorganizeNSGeneral scans ns-general and clusters nodes into specific namespaces.
func (s *ClusteringService) ReorganizeNSGeneral(ctx context.Context) (int, error) {
	ctx, span := observability.Tracer.Start(ctx, "Clustering.Reorganize")
	defer span.End()

	nodes, err := s.Cortex.ListNodes(ctx, "ns-general")
	if err != nil {
		return 0, err
	}
	if len(nodes) < 5 {
		return 0, nil // Not enough nodes to cluster effectively
	}

	processed := make(map[string]bool)
	clustersMoved := 0

	for _, node := range nodes {
		if processed[node.ID] {
			continue
		}

		// 1. Get embedding for current node
		embedding, _, err := s.Cortex.GetVector(ctx, node.ID)
		if err != nil || embedding == nil {
			continue
		}

		// 2. Find similar nodes in ns-general
		results, err := s.Cortex.VectorSearch(ctx, embedding, 10)
		if err != nil {
			continue
		}

		var clusterNodes []cortex.Node
		var clusterContents []string
		for _, res := range results {
			if res.NamespaceID == "ns-general" && res.Score > 0.85 {
				clusterNodes = append(clusterNodes, res.Node)
				clusterContents = append(clusterContents, res.Content)
				processed[res.ID] = true
			}
		}

		if len(clusterNodes) < 3 {
			// Not a significant cluster, release processed status for others to try
			for _, n := range clusterNodes {
				if n.ID != node.ID {
					delete(processed, n.ID)
				}
			}
			continue
		}

		// 3. Use LLM to name the cluster
		name, desc, err := s.identifyCluster(ctx, clusterContents)
		if err != nil {
			continue
		}

		// 4. Create Namespace and Move Nodes
		nsID := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		if !strings.HasPrefix(nsID, "ns-") {
			nsID = "ns-" + nsID
		}

		err = s.Cortex.CreateNamespace(ctx, &cortex.Namespace{
			ID:          nsID,
			Name:        name,
			Description: desc,
		})
		if err != nil {
			continue
		}

		for _, n := range clusterNodes {
			n.NamespaceID = nsID
			_ = s.Cortex.PutNode(ctx, &n)
		}
		clustersMoved++
	}

	return clustersMoved, nil
}

func (s *ClusteringService) identifyCluster(ctx context.Context, contents []string) (string, string, error) {
	prompt := fmt.Sprintf(`Analyze the following pieces of information and provide a concise "Namespace Name" (2-3 words) and a "Description" that characterizes this cluster.
The name should be a high-level category (e.g., "Chess Tactics", "Golang Concurrency", "Project Deployment").

Contents:
- %s`, strings.Join(contents, "\n- "))

	resp, err := s.Chat.LLM.Complete(ctx, prompt)
	if err != nil {
		return "", "", err
	}

	// Simple parsing of "Name: ... Description: ..."
	lines := strings.Split(resp, "\n")
	name := "Dynamic Cluster"
	desc := "Automatically organized by Wisdom REM"
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "name:") {
			name = strings.TrimSpace(line[5:])
		} else if strings.HasPrefix(strings.ToLower(line), "description:") {
			desc = strings.TrimSpace(line[12:])
		}
	}

	return name, desc, nil
}
