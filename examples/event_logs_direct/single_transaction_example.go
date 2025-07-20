package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
)

// ExampleSingleTransaction demonstrates processing a single transaction
func ExampleSingleTransaction() {
	// Create client configuration
	config := client.ClientConfig{
		NodeAddress:     "grpc.trongrid.io:50051", // Mainnet
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

	// Create contract cache
	cache := NewContractCache()

	// Create context
	ctx := context.Background()

	// Example transaction ID (you can replace this with any transaction ID)
	txID := "60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c"

	fmt.Printf("Processing transaction: %s\n", txID)

	// Get transaction info
	txInfo, err := client.GetTransactionInfoById(ctx, txID)
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}

	// Get transaction hash
	txHash := hex.EncodeToString(txInfo.GetId())
	fmt.Printf("Transaction hash: %s\n", txHash)
	fmt.Printf("Block number: %d\n", txInfo.GetBlockNumber())
	fmt.Printf("Block timestamp: %d\n", txInfo.GetBlockTimeStamp())

	// Get logs from transaction info
	logs := txInfo.GetLog()
	if len(logs) == 0 {
		fmt.Println("No logs found in this transaction")
		return
	}

	fmt.Printf("Found %d logs in transaction\n", len(logs))

	// Process logs for this transaction
	decodedLogs, err := ProcessTransactionLogs(
		ctx,
		client,
		cache,
		uint64(txInfo.GetBlockNumber()),
		uint64(txInfo.GetBlockTimeStamp()),
		txHash,
		logs,
	)
	if err != nil {
		log.Fatalf("Failed to process logs: %v", err)
	}

	// Display decoded logs
	DisplayDecodedLogs(decodedLogs)

	// Display cache statistics
	fmt.Printf("\n=== Cache Statistics ===\n")
	fmt.Printf("Total contracts cached: %d\n", len(cache.contracts))
	fmt.Printf("Cached contract addresses:\n")
	for addr := range cache.contracts {
		fmt.Printf("  %s\n", addr)
	}
}
