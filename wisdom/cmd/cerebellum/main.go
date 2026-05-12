package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/google/wisdom/pkg/cerebellum"
	cortexv1 "github.com/google/wisdom/pkg/cortex/v1"
)

func main() {
	// For production, consider using Google Cloud Pub/Sub if deploying on GCP.
	// For this architecture, we stick to NATS as specified in the Master Plan.
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	dbConnStr := os.Getenv("DB_CONN_STRING")
	if dbConnStr == "" {
		log.Println("WARNING: DB_CONN_STRING is not set.")
	}

	cortexURL := os.Getenv("CORTEX_GRPC_URL")
	if cortexURL == "" {
		cortexURL = "localhost:50051"
	}

	// 1. Establish connection to Cortex Substrate
	conn, err := grpc.Dial(cortexURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Cortex: %v", err)
	}
	defer conn.Close()

	cortexClient := cortexv1.NewCortexClient(conn)

	// 2. Initialize Worker
	worker, err := cerebellum.NewWorker(dbConnStr, natsURL, cortexClient)
	if err != nil {
		log.Fatalf("Failed to initialize Cerebellum worker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutdown signal received...")
		cancel()
	}()

	// Start processing
	if err := worker.Start(ctx); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}
