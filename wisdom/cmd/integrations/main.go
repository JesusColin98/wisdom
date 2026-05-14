package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
	masterv1 "github.com/google/wisdom/pkg/mastery/v1"
	"github.com/google/wisdom/pkg/integrations"
	pb "github.com/google/wisdom/pkg/integrations/v1"
)

// Wisdom-Integrations: the single authoritative bridge between Expert Agents
// and external UI tools (Obsidian, Anki, Logseq via local MCP servers).
// Rule: Expert Agents NEVER call MCP servers directly — all requests route here.
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50056"
	}

	// Connect to Cortex — for PENDING_SYNC queue storage and deduplication.
	cortexURL := os.Getenv("CORTEX_GRPC_URL")
	if cortexURL == "" {
		cortexURL = "localhost:50051"
		log.Println("WARNING: CORTEX_GRPC_URL not set, defaulting to", cortexURL)
	}

	// Connect to Mastery — for syncing Anki review grades to MasteryScore.
	masteryURL := os.Getenv("MASTERY_GRPC_URL")
	if masteryURL == "" {
		masteryURL = "localhost:50053"
		log.Println("WARNING: MASTERY_GRPC_URL not set, defaulting to", masteryURL)
	}

	// Local MCP server endpoints (run on user's machine, not Cloud Run).
	obsidianMCPURL := os.Getenv("OBSIDIAN_MCP_URL")
	if obsidianMCPURL == "" {
		obsidianMCPURL = "http://localhost:3333"
		log.Println("WARNING: OBSIDIAN_MCP_URL not set, defaulting to", obsidianMCPURL)
	}

	ankiMCPURL := os.Getenv("ANKI_MCP_URL")
	if ankiMCPURL == "" {
		ankiMCPURL = "http://localhost:3334"
		log.Println("WARNING: ANKI_MCP_URL not set, defaulting to", ankiMCPURL)
	}

	// GCP Pub/Sub — for publishing wisdom.integrations.sync_ready events to Portal.
	gcpProject := os.Getenv("GCP_PROJECT_ID")
	if gcpProject == "" {
		log.Fatal("FATAL: GCP_PROJECT_ID environment variable is required")
	}

	// Default user for the Anki polling and retry loops.
	defaultUserID := os.Getenv("DEFAULT_USER_ID")
	if defaultUserID == "" {
		defaultUserID = "default"
		log.Println("WARNING: DEFAULT_USER_ID not set, defaulting to 'default'")
	}

	// Anki polling cadence — 15 minutes per spec (CONTRACTS.md).
	const ankiPollIntervalSec = 15 * 60

	// Connect to Cortex.
	cortexConn, err := grpc.Dial(cortexURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Cortex: %v", err)
	}
	defer cortexConn.Close()

	// Connect to Mastery.
	masteryConn, err := grpc.Dial(masteryURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Mastery: %v", err)
	}
	defer masteryConn.Close()

	cortexClient := cortexv1.NewCortexClient(cortexConn)
	masteryClient := masterv1.NewMasteryClient(masteryConn)

	// Initialize Integrations server with all dependencies.
	intServer, err := integrations.NewServer(integrations.Config{
		CortexClient:        cortexClient,
		MasteryClient:       masteryClient,
		ObsidianMCPURL:      obsidianMCPURL,
		AnkiMCPURL:          ankiMCPURL,
		GCPProject:          gcpProject,
		AnkiPollIntervalSec: ankiPollIntervalSec,
		DefaultUserID:       defaultUserID,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Integrations server: %v", err)
	}

	// Start background loops.
	go intServer.StartAnkiPoller()
	go intServer.StartRetryLoop(defaultUserID)

	// Spin up gRPC server.
	grpcServer := grpc.NewServer()
	pb.RegisterIntegrationsServer(grpcServer, intServer)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("wisdom.integrations.v1.Integrations", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	log.Printf("Wisdom-Integrations gRPC service listening on port %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
