package kernel

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/metabolism"
	"github.com/google/wisdom/pkg/observability"
	"github.com/google/wisdom/pkg/sensory"
	"github.com/google/wisdom/pkg/thalamus"
)

// WisdomKernel aggregates all core engine components.
type WisdomKernel struct {
	Storage      *cortex.Cortex
	Tracker      *metabolism.Tracker
	Registry     *cerebellum.Registry
	Chat         *thalamus.Chat
	REM          *thalamus.REMService
	Orchestrator *thalamus.Orchestrator
	Scheduler    *thalamus.Scheduler
	Ingestor     *sensory.DocumentIngestor
	Mapper       *thalamus.MapperService
	Hierarchy    *thalamus.HierarchyManager
}

// Bootstrap initializes the entire Wisdom engine.
func Bootstrap(ctx context.Context) (*WisdomKernel, error) {
	// 1. Initialize Cortex Storage
	dbPath := os.Getenv("WISDOM_DB_PATH")
	if dbPath == "" {
		dbPath = "wisdom.db"
	}
	storage, err := cortex.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cortex: %w", err)
	}

	// Initialize Schema (heuristic paths)
	schemaPaths := []string{"pkg/cortex/schema.sql", "wisdom/pkg/cortex/schema.sql", "../pkg/cortex/schema.sql"}
	var schemaSQL []byte
	for _, p := range schemaPaths {
		if data, err := os.ReadFile(p); err == nil {
			schemaSQL = data
			break
		}
	}
	if schemaSQL != nil {
		_ = storage.InitSchema(ctx, string(schemaSQL))
	}

	// 2. Initialize Subsystems
	tracker := metabolism.NewTracker()
	registry := cerebellum.NewRegistry()
	_ = registry.LoadDynamicTools(ctx, storage)

	cache, _ := thalamus.NewCache(1000)
	_ = cache.Warm(ctx, storage)

	hippocampus := thalamus.NewHippocampus(storage)

	// 3. LLM Setup
	var llm cerebellum.LLMProvider
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")

	if projectID != "" && location != "" {
		vProvider, err := cerebellum.NewVertexProvider(ctx, projectID, location)
		if err != nil {
			observability.Logger.Warn("Vertex fallback to mock", "error", err)
			llm = &cerebellum.MockLLM{Echo: true}
		} else {
			llm = vProvider
		}
	} else {
		llm = &cerebellum.MockLLM{Echo: true}
	}
	llm = cerebellum.NewResilientLLM(llm, 5, 30*time.Second)

	// 4. Thalamic Web
	chat := &thalamus.Chat{
		Storage:     storage,
		LLM:         llm,
		Hippocampus: hippocampus,
	}

	scheduler := thalamus.NewScheduler(storage)
	clustering := thalamus.NewClusteringService(storage, chat)
	ingestor := sensory.NewDocumentIngestor(llm, storage)
	mapper := thalamus.NewMapperService(storage, chat)

	remService := &thalamus.REMService{
		Hippocampus: hippocampus,
		Cortex:      storage,
		Chat:        chat,
		Clustering:  clustering,
	}

	inquirer := thalamus.NewInquirerService(llm, storage)
	reinforce := thalamus.NewReinforcementService(storage)
	classifier := thalamus.NewIntentClassifierV2(llm)
	grepRAG := cerebellum.NewGrepRAGAgent(".")
	identity := thalamus.NewIdentityService(storage)
	hierarchy := thalamus.NewHierarchyManager(storage)
	risk := thalamus.NewRiskEngine(storage)
	sre := thalamus.NewSREAssistant(storage)

	orchestrator := thalamus.NewOrchestrator(storage, cache, inquirer, reinforce, classifier, grepRAG, identity, hierarchy, risk, sre)
	chat.Orchestrator = orchestrator

	return &WisdomKernel{
		Storage:      storage,
		Tracker:      tracker,
		Registry:     registry,
		Chat:         chat,
		REM:          remService,
		Orchestrator: orchestrator,
		Scheduler:    scheduler,
		Ingestor:     ingestor,
		Mapper:       mapper,
		Hierarchy:    hierarchy,
	}, nil
}

func (k *WisdomKernel) Close() {
	if k.Storage != nil {
		k.Storage.Close()
	}
}
