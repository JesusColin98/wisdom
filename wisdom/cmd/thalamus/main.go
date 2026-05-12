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
	"github.com/google/wisdom/pkg/thalamus"
	pb "github.com/google/wisdom/pkg/thalamus/v1"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50052"
	}

	cortexURL := os.Getenv("CORTEX_GRPC_URL")
	if cortexURL == "" {
		cortexURL = "localhost:50051" // Default local
		log.Println("WARNING: CORTEX_GRPC_URL is not set. Defaulting to", cortexURL)
	}

	// 1. Establish connection to Cortex Substrate
	// Note: Using insecure credentials for local dev. In production, use TLS.
	conn, err := grpc.Dial(cortexURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Cortex: %v", err)
	}
	defer conn.Close()

	cortexClient := cortexv1.NewCortexClient(conn)

	// 2. Initialize Thalamus Server
	thalamusServer := thalamus.NewServer(cortexClient)

	// 3. Create standard gRPC server
	grpcServer := grpc.NewServer()

	// Register Thalamus Service
	pb.RegisterThalamusServer(grpcServer, thalamusServer)

	// Register Health Check Service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("wisdom.thalamus.v1.Thalamus", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// 4. Start listening
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	log.Printf("Thalamus gRPC Gateway listening on port %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
