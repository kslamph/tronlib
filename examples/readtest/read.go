package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"sync"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc/status"
)

func main() {
	start := time.Now()
	fmt.Printf("Starting TRON account query at %v\n\n", start.Format(time.RFC3339))

	// Initialize client with configuration
	config := client.ClientConfig{
		NodeAddress: "localhost:50051",
		Timeout:     15 * time.Second, // 1 second timeout for RPC calls
	}

	fmt.Printf("Connection Settings:\n")
	fmt.Printf("-------------------\n")
	fmt.Printf("Node address: %s\n", config.NodeAddress)
	fmt.Printf("RPC timeout: %v\n", config.Timeout)

	// Create client
	client, err := client.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	addr, err := types.NewAddress("TDUiUScimQNfmD1F76Uq6YaXbofCVuAvxH")
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}
	x := 20 * time.Millisecond

	ticker := time.NewTicker(x)
	defer ticker.Stop()
	reqCount := 0
	failCount := 0
	testDuration := 5 * time.Second // Run for 10 seconds, adjust as needed
	fmt.Printf("\nStarting %d Seconds stress test: %d ms per test....\n", testDuration/time.Second, x/time.Millisecond)
	timeout := time.After(testDuration)
	var wg sync.WaitGroup

	ctx := context.Background()

loop:
	for {
		select {
		case <-timeout:
			break loop
		case <-ticker.C:
			wg.Add(1)
			go func(ctx context.Context) {
				defer wg.Done()
				ac, err := client.GetAccount(ctx, addr)
				if err != nil {
					// Print full gRPC error details if available
					if st, ok := status.FromError(err); ok {
						fmt.Printf("gRPC Status Code: %v\n", st.Code())
						fmt.Printf("gRPC Message: %s\n", st.Message())
						fmt.Printf("gRPC Details: %v\n", st.Details())
					} else {
						fmt.Printf("Non-gRPC error type: %T, error: %v\n", err, err)
					}
					failCount++
					return
				}
				_ = ac // Optionally process result
			}(ctx)
			reqCount++
		}
	}
	wg.Wait()
	fmt.Printf("\nStress test finished. Total requests: %d, Failures: %d\n", reqCount, failCount)

	// Query transaction info with 5 second timeout
	txId := "44519f26abfdc64c4a56fc85122f62279124bb12a41ce26ea65e3ab370d75ca5"
	fmt.Printf("\nQuerying transaction %s...\n", txId)
	txInfo, err := client.GetTransactionInfoById(ctx, txId)
	if err != nil {
		log.Printf("Failed to get transaction info: %v\n", err)
	} else {
		fmt.Printf("\nTransaction Information:\n")
		fmt.Printf("====================\n")
		fmt.Printf("Block Number: %d\n", txInfo.GetBlockNumber())
		fmt.Printf("Result: %v\n", txInfo.GetResult())
	}
}
