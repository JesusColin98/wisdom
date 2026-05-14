package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/google/wisdom/pkg/researcher"
	pb "github.com/google/wisdom/pkg/researcher/v1"
)

// Wisdom-Researcher: autonomous content gathering gRPC service.
// Migrated from NATS to GCP Pub/Sub (architecture decision locked 2026-05-14).
// Publishes wisdom.knowledge.ingested events to Pub/Sub on job completion.
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50054"
	}

	// GCS bucket for raw content storage (PDFs, audio, large HTML).
	// Files are auto-deleted after 24h or upon extraction success (TTL-restricted bucket).
	gcsBucket := os.Getenv("GCS_INGESTION_BUCKET")
	if gcsBucket == "" {
		log.Fatal("FATAL: GCS_INGESTION_BUCKET environment variable is required")
	}

	// GCP project — for Pub/Sub publishing.
	gcpProject := os.Getenv("GCP_PROJECT_ID")
	if gcpProject == "" {
		log.Fatal("FATAL: GCP_PROJECT_ID environment variable is required")
	}

	// Initialize Researcher service with Pub/Sub publisher.
	researcherServer, err := researcher.NewServer(gcsBucket, gcpProject)
	if err != nil {
		log.Fatalf("Failed to initialize Researcher server: %v", err)
	}

	// Spin up gRPC server.
	grpcServer := grpc.NewServer()
	pb.RegisterResearcherServer(grpcServer, researcherServer)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("wisdom.researcher.v1.Researcher", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	log.Printf("Wisdom-Researcher gRPC service listening on port %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
