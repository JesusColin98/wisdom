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
	"github.com/google/wisdom/pkg/entity"
	pb "github.com/google/wisdom/pkg/entity/v1"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50057"
	}

	// Connect to Cortex — Entity Dictionary stores entity profiles as Cortex nodes.
	cortexURL := os.Getenv("CORTEX_GRPC_URL")
	if cortexURL == "" {
		cortexURL = "localhost:50051"
		log.Println("WARNING: CORTEX_GRPC_URL not set, defaulting to", cortexURL)
	}

	// Connect to Cortex.
	cortexConn, err := grpc.Dial(cortexURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Cortex: %v", err)
	}
	defer cortexConn.Close()

	cortexClient := cortexv1.NewCortexClient(cortexConn)

	// Initialize Entity Dictionary server.
	entityServer := entity.NewServer(cortexClient)

	// Spin up gRPC server.
	grpcServer := grpc.NewServer()
	pb.RegisterEntityDictionaryServer(grpcServer, entityServer)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("wisdom.entity.v1.EntityDictionary", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	log.Printf("Wisdom-Entity-Dictionary gRPC service listening on port %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
