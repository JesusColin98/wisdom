package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/wisdom/pkg/api"
	"github.com/google/wisdom/pkg/kernel"
	"github.com/google/wisdom/pkg/mcp"
	"github.com/google/wisdom/pkg/observability"
	"github.com/google/wisdom/pkg/thalamus"
)

func main() {
	observability.InitLogger()
	shutdown := observability.InitTracer()
	defer shutdown(context.Background())

	observability.Logger.Info("🌌 Wisdom API Server starting")
	ctx := context.Background()

	k, err := kernel.Bootstrap(ctx)
	if err != nil {
		observability.Logger.Error("Bootstrap failed", "error", err)
		os.Exit(1)
	}
	defer k.Close()

	validator := thalamus.NewValidator()
	if err := validator.LoadSchemasFromDir("pkg/cerebellum/schemas"); err != nil {
		_ = validator.LoadSchemasFromDir("wisdom/pkg/cerebellum/schemas")
	}

	// Check for MCP mode
	mode := os.Getenv("WISDOM_MODE")
	isMCP := mode == "mcp"
	for _, arg := range os.Args {
		if arg == "--mcp" {
			isMCP = true
			break
		}
	}

	if isMCP {
		mcpServer := mcp.NewServer(k.Orchestrator, k.Chat, k.REM)
		if err := mcpServer.Listen(ctx); err != nil {
			observability.Logger.Error("MCP Server failed", "error", err)
			os.Exit(1)
		}
		return
	}

	server := api.NewServer(k.Storage, k.Tracker, validator, k.Registry, k.Chat, k.REM, k.Orchestrator, k.Scheduler, k.Ingestor, k.Mapper, k.Hierarchy, k.Learning)

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	if port[0] != ':' { port = ":" + port }

	fmt.Fprintf(os.Stderr, "🟢 Wisdom API Server listening on %s\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		os.Exit(1)
	}
}
