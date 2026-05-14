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
	"github.com/google/wisdom/pkg/mastery"
	pb "github.com/google/wisdom/pkg/mastery/v1"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50053"
	}

	// Cortex connection — Mastery reads/writes MasteryScore to Cortex.
	cortexURL := os.Getenv("CORTEX_GRPC_URL")
	if cortexURL == "" {
		cortexURL = "localhost:50051"
		log.Println("WARNING: CORTEX_GRPC_URL not set, defaulting to", cortexURL)
	}

	// GCP Pub/Sub project — for publishing wisdom.user.struggle_detected events.
	gcpProject := os.Getenv("GCP_PROJECT_ID")
	if gcpProject == "" {
		log.Fatal("FATAL: GCP_PROJECT_ID environment variable is required")
	}

	// Connect to Cortex substrate.
	conn, err := grpc.Dial(cortexURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Cortex: %v", err)
	}
	defer conn.Close()

	cortexClient := cortexv1.NewCortexClient(conn)

	// Initialize the unified Mastery server (Trace + Metabolism).
	masteryServer, err := mastery.NewServer(cortexClient, gcpProject)
	if err != nil {
		log.Fatalf("Failed to initialize Mastery server: %v", err)
	}

	// Spin up gRPC server.
	grpcServer := grpc.NewServer()
	pb.RegisterMasteryServer(grpcServer, masteryServer)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("wisdom.mastery.v1.Mastery", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	log.Printf("Wisdom-Mastery (Trace+Metabolism) gRPC service listening on port %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
