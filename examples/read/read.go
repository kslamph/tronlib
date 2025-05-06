package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	start := time.Now()
	fmt.Printf("Starting TRON account query at %v\n\n", start.Format(time.RFC3339))

	// Initialize client with options
	opts := &client.ClientOptions{
		Endpoints: []string{
			"grpc.trongrid.io:50055",        // Invalid port to trigger failover
			"grpc.nile.trongrid.io:50055",   // Invalid port to trigger failover
			"5.5.6.9:50051",                 // Invalid IP to trigger failover
			"grpc.shasta.trongrid.io:50051", // Primary testnet
		},
		Timeout: 1 * time.Second, // Very aggressive timeout for quick failover
		RetryConfig: &client.RetryConfig{
			MaxAttempts:    5,                      // Will try all endpoints
			InitialBackoff: 10 * time.Millisecond,  // Minimal initial wait
			MaxBackoff:     100 * time.Millisecond, // Keep backoff short
			BackoffFactor:  1.2,                    // Gentle increase
		},
	}

	fmt.Printf("Connection Settings:\n")
	fmt.Printf("-------------------\n")
	fmt.Printf("Operation timeout: %v\n", opts.Timeout)
	fmt.Printf("Max retry attempts: %d\n", opts.RetryConfig.MaxAttempts)
	fmt.Printf("Initial backoff: %v\n", opts.RetryConfig.InitialBackoff)
	fmt.Printf("Max backoff: %v\n", opts.RetryConfig.MaxBackoff)

	fmt.Printf("\nConfigured Endpoints:\n")
	fmt.Printf("--------------------\n")
	for i, endpoint := range opts.Endpoints {
		fmt.Printf("%d. %s\n", i+1, endpoint)
	}

	// Create client with tight overall timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := client.NewClient(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	addr, err := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}

	// Get account with timeout
	// ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	// defer cancel()

	fmt.Printf("\nQuerying account information...\n")
	fmt.Printf("This operation will timeout after %v and try next endpoint\n", opts.Timeout)
	queryStart := time.Now()
	ac, err := client.GetAccount(addr)
	if err != nil {
		log.Fatalf("Failed to get account: %v", err)
	}

	fmt.Printf("\nAccount Information:\n")
	fmt.Printf("==================\n")
	fmt.Printf("Address: %s\n", addr.String())
	fmt.Printf("Balance: %d TRX\n", ac.GetBalance()/1_000_000) // Convert from SUN to TRX
	if len(ac.GetFrozenV2()) > 0 {
		fmt.Printf("\nFrozen Balances:\n")
		for _, frozen := range ac.GetFrozenV2() {
			fmt.Printf("- Type: %v, Amount: %d TRX\n", frozen.GetType(), frozen.GetAmount()/1_000_000)
		}
	}
	fmt.Printf("\nAccount Resource Usage:\n")
	if resource := ac.GetAccountResource(); resource != nil {
		fmt.Printf("Energy Window Size: %d\n", resource.GetEnergyWindowSize())
		fmt.Printf("Latest Energy Consumption Time: %v\n",
			time.Unix(resource.GetLatestConsumeTimeForEnergy()/1000, 0))
	}

	fmt.Printf("\nOperation Summary:\n")
	fmt.Printf("=================\n")
	fmt.Printf("Total time: %v\n", time.Since(start))
	fmt.Printf("Query time: %v\n", time.Since(queryStart))

	rcp, err := client.WaitForTransactionInfo("41fee269a29af3604c4082c48ae372d860170745b1b7dbb92425ac01b52dc7dd", 1)
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}
	fmt.Printf("\nTransaction Information:\n")
	fmt.Printf("====================\n")
	fmt.Println(rcp)
	fmt.Println(rcp.GetRet()) // This will trigger a failover if the first endpoint is down

	rcp2, err := client.GetTransactionInfoById("41fee269a29af3604c4082c48ae372d860170745b1b7dbb92425ac01b52dc7dd")
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}
	fmt.Printf("\nTransaction Information:\n")
	fmt.Printf("====================\n")
	fmt.Println(rcp2)
}
