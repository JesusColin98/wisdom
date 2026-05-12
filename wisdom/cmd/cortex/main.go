package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/google/wisdom/pkg/cortex"
	pb "github.com/google/wisdom/pkg/cortex/v1"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	connStr := os.Getenv("DB_CONN_STRING")
	if connStr == "" {
		// Use a fallback or log a fatal error based on your environment policies
		log.Println("WARNING: DB_CONN_STRING is not set. Database connection may fail.")
	}

	// Initialize the database engine
	engine, err := cortex.NewPostgresEngine(connStr)
	if err != nil {
		log.Fatalf("Failed to initialize Postgres engine: %v", err)
	}
	defer engine.Close()

	// Initialize the gRPC server implementation
	cortexServer := cortex.NewServer(engine)

	// Create standard gRPC server
	grpcServer := grpc.NewServer()

	// Register Cortex Service (Wait until protoc is fixed to uncomment in real env)
	// Currently using stubs
	pb.RegisterCortexServer(grpcServer, cortexServer)

	// Register Health Check Service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("wisdom.cortex.v1.Cortex", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Start listening
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	log.Printf("Cortex gRPC Substrate listening on port %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
