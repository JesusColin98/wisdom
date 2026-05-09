package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/wisdom/pkg/api"
	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/metabolism"
	"github.com/google/wisdom/pkg/observability"
	"github.com/google/wisdom/pkg/thalamus"
)

func main() {
	// 1. Initialize Observability
	observability.InitLogger()
	shutdown := observability.InitTracer()
	defer shutdown(context.Background())

	observability.Logger.Info("🌌 Wisdom: The Cognitive SRE Engine (v2.0) starting")

	ctx := context.Background()

	// 2. Initialize Cortex Storage
	storage, err := cortex.Open("wisdom.db")
	if err != nil {
		observability.Logger.Error("Failed to open Cortex storage", "error", err)
		os.Exit(1)
	}
	defer storage.Close()

	// Initialize Schema
	schemaSQL, err := os.ReadFile("pkg/cortex/schema.sql")
	if err != nil {
		// Fallback for when running from root vs cmd/wisdom
		schemaSQL, err = os.ReadFile("wisdom/pkg/cortex/schema.sql")
		if err != nil {
			observability.Logger.Error("Failed to read Cortex schema", "error", err)
			os.Exit(1)
		}
	}
	if err := storage.InitSchema(ctx, string(schemaSQL)); err != nil {
		observability.Logger.Error("Failed to initialize Cortex schema", "error", err)
		os.Exit(1)
	}

	// 3. Initialize Metabolism Tracker
	tracker := metabolism.NewTracker()

	// 4. Initialize Cerebellum Registry
	registry := cerebellum.NewRegistry()
	if err := registry.LoadDynamicTools(ctx, storage); err != nil {
		observability.Logger.Error("Failed to restore dynamic tools", "error", err)
	}

	// 5. Initialize Thalamus Components
	cache, _ := thalamus.NewCache(1000)
	if err := cache.Warm(ctx, storage); err != nil {
		observability.Logger.Error("Failed to warm Thalamic cache", "error", err)
	}

	hippocampus := thalamus.NewHippocampus(storage)

	// 5.1 Initialize LLM Provider (Real Vertex AI vs Mock)
	var llm cerebellum.LLMProvider
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")

	if projectID != "" && location != "" {
		vProvider, err := cerebellum.NewVertexProvider(ctx, projectID, location)
		if err != nil {
			observability.Logger.Error("Failed to initialize VertexProvider, falling back to mock", "error", err)
			llm = &cerebellum.MockLLM{Echo: true}
		} else {
			observability.Logger.Info("Using Vertex AI Provider", "project", projectID, "location", location)
			llm = vProvider
		}
	} else {
		observability.Logger.Warn("GOOGLE_CLOUD_PROJECT/LOCATION not set, using MockLLM")
		llm = &cerebellum.MockLLM{Echo: true}
	}

	// 5.2 Wrap with Circuit Breaker (ResilientLLM)
	llm = cerebellum.NewResilientLLM(llm, 5, 30*time.Second)

	chat := &thalamus.Chat{
		Storage:     storage,
		LLM:         llm,
		Hippocampus: hippocampus,
	}

	remService := &thalamus.REMService{
		Hippocampus: hippocampus,
		Cortex:      storage,
		Chat:        chat,
	}

	orchestrator := thalamus.NewOrchestrator(storage, nil) // Cache nil for now

	// 6. Initialize API Validator and load schemas dynamically
	validator := thalamus.NewValidator()
	if err := validator.LoadSchemasFromDir("pkg/cerebellum/schemas"); err != nil {
		// Fallback for different execution contexts
		_ = validator.LoadSchemasFromDir("wisdom/pkg/cerebellum/schemas")
	}

	// 7. Create API Server
	server := api.NewServer(storage, tracker, validator, registry, chat, remService, orchestrator)

	// 8. Register Handlers
	mux := http.NewServeMux()
	server.RegisterHandlers(mux)

	// 9. Start the Server
	port := ":8080"
	observability.Logger.Info("Wisdom node operational", "port", port)
	fmt.Printf("🟢 Wisdom Server listening on %s\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		observability.Logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
