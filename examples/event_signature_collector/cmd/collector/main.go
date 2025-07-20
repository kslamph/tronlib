package main

import (
	"context"
	"flag"
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
	// Define command line flags
	var (
		nodeAddr = flag.String("node", "127.0.0.1:50051", "TRON node address")
		dbPath   = flag.String("db", "event_signatures.db", "Path to the SQLite database")
		timeout  = flag.Duration("timeout", 30*time.Second, "Client timeout")
		interval = flag.Duration("interval", 4*time.Second, "Block processing interval")
	)
	flag.Parse()

	// Create client configuration
	config := client.ClientConfig{
		NodeAddress:     *nodeAddr,
		Timeout:         *timeout,
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
	db, err := database.NewDatabase(*dbPath)
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

	fmt.Printf("Starting event signature collector...\n")
	fmt.Printf("Node: %s\n", *nodeAddr)
	fmt.Printf("Database: %s\n", *dbPath)
	fmt.Printf("Interval: %v\n", *interval)
	fmt.Println("Press Ctrl+C to stop")

	// Start collection loop with custom interval
	collector.StartWithInterval(ctx, *interval)
}

func init() {
	// Set up usage information
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Event Signature Collector\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -node 127.0.0.1:50051 -db signatures.db\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -interval 2s -timeout 60s\n", os.Args[0])
	}
}
