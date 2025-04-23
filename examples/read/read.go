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
	// Initialize client with options
	opts := &client.ClientOptions{
		Endpoints: []string{"grpc.shasta.trongrid.io:50051"},
		Timeout:   10 * time.Second,
		RetryConfig: &client.RetryConfig{
			MaxAttempts:    2, // Will try 3 times total (initial + 2 retries)
			InitialBackoff: time.Second,
			MaxBackoff:     10 * time.Second,
			BackoffFactor:  2.0,
		},
	}

	// Create client
	ctx := context.Background()
	client, err := client.NewClient(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	addr, err := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}

	// Create new transaction
	ac, err := client.GetAccount(addr)
	if err != nil {
		log.Fatalf("Failed to get account: %v", err)
	}
	fmt.Println("Account:", ac)
}
