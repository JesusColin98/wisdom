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
	"github.com/google/wisdom/pkg/curriculum"
	pb "github.com/google/wisdom/pkg/curriculum/v1"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50055"
	}

	// Connect to Cortex — Curriculum queries the knowledge graph for concept nodes.
	cortexURL := os.Getenv("CORTEX_GRPC_URL")
	if cortexURL == "" {
		cortexURL = "localhost:50051"
		log.Println("WARNING: CORTEX_GRPC_URL not set, defaulting to", cortexURL)
	}

	// Connect to Mastery — Curriculum filters out concepts already mastered by the user.
	masteryURL := os.Getenv("MASTERY_GRPC_URL")
	if masteryURL == "" {
		masteryURL = "localhost:50053"
		log.Println("WARNING: MASTERY_GRPC_URL not set, defaulting to", masteryURL)
	}

	// GCP Pub/Sub — for publishing wisdom.learning.path_created events.
	gcpProject := os.Getenv("GCP_PROJECT_ID")
	if gcpProject == "" {
		log.Fatal("FATAL: GCP_PROJECT_ID environment variable is required")
	}

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

	// Initialize Curriculum server.
	curriculumServer, err := curriculum.NewServer(cortexClient, masteryClient, gcpProject)
	if err != nil {
		log.Fatalf("Failed to initialize Curriculum server: %v", err)
	}

	// Spin up gRPC server.
	grpcServer := grpc.NewServer()
	pb.RegisterCurriculumServer(grpcServer, curriculumServer)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("wisdom.curriculum.v1.Curriculum", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	log.Printf("Wisdom-Curriculum gRPC service listening on port %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
