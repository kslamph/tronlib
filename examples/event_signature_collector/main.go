package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"event_signature_collector/collector"
	"event_signature_collector/database"

	"github.com/kslamph/tronlib/pkg/client"
)

func main() {
	// Create client configuration
	config := client.ClientConfig{
		NodeAddress:     "127.0.0.1:50051", // Mainnet
		Timeout:         30 * time.Second,
		InitConnections: 1,
		MaxConnections:  5,
		IdleTimeout:     60 * time.Second,
	}

	// Create client
	client, err := client.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize database
	db, err := database.NewDatabase("event_signatures.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		cancel()
	}()

	// Create collector
	collector := collector.NewEventSignatureCollector(client, db)

	fmt.Println("Starting event signature collector...")
	fmt.Println("Press Ctrl+C to stop")

	// Start collection loop
	collector.Start(ctx)
}
