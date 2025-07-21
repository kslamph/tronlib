package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"event_signature_collector/collector"
	"event_signature_collector/database"

	"github.com/kslamph/tronlib/pkg/client"
)

func main() {
	// Define command line flags
	var (
		nodeAddr    = flag.String("node", "127.0.0.1:50051", "TRON node address")
		dbPath      = flag.String("db", "event_signatures.db", "Path to the SQLite database")
		timeout     = flag.Duration("timeout", 30*time.Second, "Client timeout")
		startBlock  = flag.Int64("start", 0, "Start block number (inclusive)")
		endBlock    = flag.Int64("end", 0, "End block number (inclusive)")
		concurrency = flag.Int("concurrency", 4, "Number of concurrent workers")
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
	fmt.Println("Strategy: Scan current block once, then scan backwards from current-1200 blocks continuously")
	fmt.Println("Press Ctrl+C to stop")

	// If start/end blocks are specified, use concurrent workers
	if *startBlock > 0 && *endBlock > 0 && *endBlock >= *startBlock {
		fmt.Printf("Concurrent block processing: start=%d end=%d concurrency=%d\n", *startBlock, *endBlock, *concurrency)
		processBlocksConcurrently(ctx, collector, *startBlock, *endBlock, *concurrency)
		collector.ShutdownWriter()
	} else {
		// Start collection loop (legacy behavior)
		collector.Start(ctx)
		collector.ShutdownWriter()
	}
}

// processBlocksConcurrently processes blocks in the given range using concurrent workers
func processBlocksConcurrently(ctx context.Context, collector *collector.EventSignatureCollector, start, end int64, concurrency int) {
	blockCh := make(chan int64, concurrency*2)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for blockNum := range blockCh {
				select {
				case <-ctx.Done():
					return
				default:
					if err := collector.ProcessBlockByNumber(ctx, blockNum); err != nil {
						log.Printf("Error processing block %d: %v", blockNum, err)
					}
				}
			}
		}()
	}

	// Feed block numbers
	for b := start; b <= end; b++ {
		select {
		case <-ctx.Done():
			break
		case blockCh <- b:
		}
	}
	close(blockCh)
	wg.Wait()
}

func init() {
	// Set up usage information
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Event Signature Collector\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Strategy: Scans current block once at startup, then scans backwards\n")
		fmt.Fprintf(os.Stderr, "         from current block -1200 continuously without delay.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -node 127.0.0.1:50051 -db signatures.db\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -timeout 60s\n", os.Args[0])
	}
}
