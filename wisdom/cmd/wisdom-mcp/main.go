package main

import (
	"context"
	"os"

	"github.com/google/wisdom/pkg/kernel"
	"github.com/google/wisdom/pkg/mcp"
	"github.com/google/wisdom/pkg/observability"
)

func main() {
	observability.InitLogger()
	ctx := context.Background()

	k, err := kernel.Bootstrap(ctx)
	if err != nil {
		os.Exit(1)
	}
	defer k.Close()

	server := mcp.NewServer(k.Orchestrator, k.Chat, k.REM)
	if err := server.Listen(ctx); err != nil {
		os.Exit(1)
	}
}
