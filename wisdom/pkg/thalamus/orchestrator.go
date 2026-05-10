package thalamus

import (
	"context"
	"sort"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// GatingMode defines the retrieval strategy.
type GatingMode string

const (
	LowCostMode  GatingMode = "LOW_COST"
	HighCostMode GatingMode = "HIGH_COST"
)

// Orchestrator aggregates context from various substrates.
type Orchestrator struct {
	cortex     *cortex.Cortex
	cache      *Cache
	inquirer   *InquirerService
	reinforce  *ReinforcementService
	classifier *IntentClassifierV2
	grepRAG    *cerebellum.GrepRAGAgent
	identity   *IdentityService
	hierarchy  *HierarchyManager
	risk       *RiskEngine
	sre        *SREAssistant
	Config     WisdomConfig
}

// NewOrchestrator creates a new Thalamic Orchestrator.
func NewOrchestrator(cx *cortex.Cortex, c *Cache, inquirer *InquirerService, reinforce *ReinforcementService, classifier *IntentClassifierV2, grepRAG *cerebellum.GrepRAGAgent, identity *IdentityService, hierarchy *HierarchyManager, risk *RiskEngine, sre *SREAssistant) *Orchestrator {
	return &Orchestrator{
		cortex:     cx,
		cache:      c,
		inquirer:   inquirer,
		reinforce:  reinforce,
		classifier: classifier,
		grepRAG:    grepRAG,
		identity:   identity,
		hierarchy:  hierarchy,
		risk:       risk,
		sre:        sre,
		Config:     DefaultConfig(),
	}
}

// UpdateConfig allows dynamic adjustment of parameters from UI/Chatbot.
func (o *Orchestrator) UpdateConfig(newConfig WisdomConfig) {
	o.Config = newConfig
}

// ResolvePointer resolves a human-friendly alias to a node ID.
func (o *Orchestrator) ResolvePointer(ctx context.Context, pointer string) (string, error) {
	return o.cortex.ResolvePointer(ctx, pointer)
}

// GetImpactGraph performs a breadth-first traversal of dependencies.
func (o *Orchestrator) GetImpactGraph(ctx context.Context, nodeID string, maxDepth int) ([]cortex.Node, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.GetImpactGraph")
	defer span.End()

	rootID, err := o.cortex.ResolvePointer(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	impacted := make(map[string]cortex.Node)
	queue := []string{rootID}
	depths := map[string]int{rootID: 0}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if depths[curr] >= maxDepth {
			continue
		}

		query := `
			SELECT target_id 
			FROM links 
			WHERE source_id = ? AND relation_type IN ('DEPENDS_ON', 'PARENT_OF')
		`
		rows, err := o.cortex.DB().QueryContext(ctx, query, curr)
		if err != nil {
			continue
		}

		var neighborIDs []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err == nil {
				neighborIDs = append(neighborIDs, id)
			}
		}
		rows.Close()

		for _, id := range neighborIDs {
			if _, seen := impacted[id]; !seen {
				node, err := o.cortex.GetNode(ctx, id)
				if err == nil && node != nil {
					impacted[id] = *node
					depths[id] = depths[curr] + 1
					queue = append(queue, id)
				}
			}
		}
	}

	var results []cortex.Node
	for _, n := range impacted {
		results = append(results, n)
	}
	return results, nil
}

// GetNearbyWisdom explores the graph using SOTA MCMI (Minimum Cost Maximum Influence) heuristics.
func (o *Orchestrator) GetNearbyWisdom(ctx context.Context, seedIDs []string, iterations int) ([]cortex.ScoredNode, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.GetNearbyWisdom")
	defer span.End()

	// 1. Semantic Propagation (Personalized PageRank)
	scores, err := o.cortex.Propagate(ctx, seedIDs, 0.85, iterations)
	if err != nil {
		return nil, err
	}

	// 2. Hydrate and Apply MCMI Heuristic
	var results []cortex.ScoredNode
	for id, influence := range scores {
		node, err := o.cortex.GetNode(ctx, id)
		if err != nil || node == nil {
			continue
		}
		if node.SupersededByID != "" {
			continue
		}

		// MCMI Heuristic: Score = Influence / log(Cost + 1)
		cost := float64(len(node.Content)) / 4.0
		if cost < 1 {
			cost = 1
		}

		mcmiScore := influence * (node.ConfidenceScore / (cost / 100.0))

		results = append(results, cortex.ScoredNode{
			Node:  *node,
			Score: mcmiScore,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// DetermineRetrieveMode implements metabolic gating based on uncertainty and token availability.
func (o *Orchestrator) DetermineRetrieveMode(uncertainty float64, tokensUsed int, budget int) GatingMode {
	remainingFraction := 1.0 - (float64(tokensUsed) / float64(budget))
	if budget > 0 && remainingFraction < 0.2 {
		return LowCostMode
	}

	if uncertainty > o.Config.UncertaintyThreshold || remainingFraction > 0.8 {
		return HighCostMode
	}

	return LowCostMode
}

// Recall fetches context for the LLM using Intent-Driven Retrieval.
func (o *Orchestrator) Recall(ctx context.Context, userID string, query string, seeds []string, budget int, uncertainty float64) (*Context, error) {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Recall")
	defer span.End()

	if budget <= 0 {
		budget = o.Config.TokenBudget
	}

	session, ok := o.cache.GetSession(userID)
	if !ok {
		session = NewSession(userID, userID)
		o.cache.PutSession(session)
	}

	// 1. Intent Detection
	intentRes := IntentResult{Intent: IntentGeneral, Strictness: DynamicContext, Confidence: 0.5}
	if o.classifier != nil {
		intentRes, _ = o.classifier.Classify(ctx, query)
		observability.Logger.Info("Query Intent Classified", "intent", intentRes.Intent, "strictness", intentRes.Strictness, "confidence", intentRes.Confidence)
	}

	var nearby []cortex.ScoredNode
	iterations := o.Config.DefaultRetrievalDepth

	// 2. Pattern-Based Retrieval (SOTA)
	switch intentRes.Intent {
	case IntentCode:
		if o.grepRAG != nil {
			code, err := o.grepRAG.Search(ctx, query, "")
			if err == nil && code != "" {
				nearby = append(nearby, cortex.ScoredNode{
					Node: cortex.Node{
						ID:      "grep_result",
						Content: code,
						Stratum: "HOT",
					},
					Score: 1.0,
				})
			}
		}
	case IntentHierarchy:
		// Phase 4 Tree-RAG logic
		if o.hierarchy != nil && len(seeds) > 0 {
			lineage, _ := o.hierarchy.GetLineage(ctx, seeds[0], "DOWN")
			for _, n := range lineage {
				nearby = append(nearby, cortex.ScoredNode{Node: n, Score: 1.0})
			}
		}
		nearbySeeds, _ := o.GetNearbyWisdom(ctx, seeds, iterations+1)
		nearby = append(nearby, nearbySeeds...)
	case IntentStar:
		// Phase 4 Risk Engine logic
		if o.risk != nil && len(seeds) > 0 {
			risks, _ := o.risk.CalculateEntityRisk(ctx, seeds[0], 2)
			for _, r := range risks {
				node, _ := o.cortex.GetNode(ctx, r.NodeID)
				if node != nil {
					nearby = append(nearby, cortex.ScoredNode{Node: *node, Score: r.Score})
				}
			}
		}
	case IntentKG:
		// Phase 4 SRE causal logic
		if o.sre != nil && len(seeds) > 0 {
			chains, _ := o.sre.TraceCausalPath(ctx, seeds[0])
			for _, c := range chains {
				for _, n := range c.Nodes {
					nearby = append(nearby, cortex.ScoredNode{Node: n, Score: c.Score})
				}
			}
		}
	default:
		mode := o.DetermineRetrieveMode(uncertainty, 0, budget)
		if mode == HighCostMode {
			iterations++
		}

		effectiveSeeds := make([]string, 0, len(seeds))
		for _, s := range seeds {
			id, err := o.cortex.ResolvePointer(ctx, s)
			if err == nil {
				effectiveSeeds = append(effectiveSeeds, id)
			} else {
				nodes, _ := o.cortex.SearchNodes(ctx, s)
				for _, n := range nodes {
					effectiveSeeds = append(effectiveSeeds, n.ID)
				}
			}
		}
		nearby, _ = o.GetNearbyWisdom(ctx, effectiveSeeds, iterations)
	}

	// 3. Metabolic Auto-Adjustment (Phase 3 final)
	// Calculate current TSR (Token-to-Signal Ratio)
	totalSignal := 0.0
	totalTokens := 0
	for _, sn := range nearby {
		totalSignal += sn.Score
		totalTokens += (len(sn.Content) + 400) / 4
	}
	tsr := 0.0
	if totalTokens > 0 {
		tsr = totalSignal / (float64(totalTokens) / 100.0)
	}

	span.SetAttributes(
		observability.AttrFloat64("metabolism.tsr", tsr),
		observability.AttrInt("metabolism.iterations", iterations),
	)

	// Adjust for next turn or Neural-Socratic loop based on TSR
	if tsr < 0.2 && iterations < o.Config.DefaultRetrievalDepth+2 {
		span.AddEvent("Low TSR detected: increasing retrieval depth for Neural-Socratic loop")
		iterations++
	}

	// 4. Neural-Socratic Refinement
	if (intentRes.Intent == IntentRelational || uncertainty > 0.6 || tsr < 0.3) && o.inquirer != nil {
		suggestions, err := o.inquirer.AnalyzeGaps(ctx, query, nearby)
		if err == nil && len(suggestions) > 0 {
			extraSeeds, _ := o.inquirer.ExpandSeeds(ctx, suggestions)
			if len(extraSeeds) > 0 {
				extraNearby, _ := o.GetNearbyWisdom(ctx, extraSeeds, 1)
				nearby = append(nearby, extraNearby...)
			}
		}
	}

	// 4. Filter and Budgeting
	var aggregated []string
	var usedNodeIDs []string
	currentSize := 0
	for _, sn := range nearby {
		nodeCost := (len(sn.Content) + 400) / 4
		if currentSize+nodeCost > budget {
			break
		}
		if uncertainty < 0.4 && sn.Stratum == "COLD" {
			continue
		}
		aggregated = append(aggregated, sn.Content)
		usedNodeIDs = append(usedNodeIDs, sn.ID)
		currentSize += nodeCost
	}

	// 5. Synaptic Reinforcement
	if o.reinforce != nil && len(usedNodeIDs) > 0 {
		_ = o.reinforce.ReinforcePath(ctx, usedNodeIDs)
	}

	return &Context{
		Session:    session,
		Wisdom:     aggregated,
		Budget:     budget,
		Strictness: intentRes.Strictness,
	}, nil
}

// CalculateRisk exposes the RiskEngine analysis.
func (o *Orchestrator) CalculateRisk(ctx context.Context, rootID string, depth int) ([]RiskScore, error) {
	if o.risk == nil {
		return nil, nil
	}
	return o.risk.CalculateEntityRisk(ctx, rootID, depth)
}

// TraceCausality exposes the SREAssistant analysis.
func (o *Orchestrator) TraceCausality(ctx context.Context, nodeID string) ([]CausalChain, error) {
	if o.sre == nil {
		return nil, nil
	}
	return o.sre.TraceCausalPath(ctx, nodeID)
}
