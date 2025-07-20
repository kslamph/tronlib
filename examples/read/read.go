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

	// Note: The timeout is for connection establishment, not the entire RPC call duration
	// A 1ms timeout means the connection must be established within 1ms, but once connected,
	// the RPC call can take longer depending on server response time
	fmt.Println("Creating client using mainnet endpoint...")
	tronClient, err := client.NewClient(client.ClientConfig{
		NodeAddress: "127.0.0.1:50051",
		Timeout:     200 * time.Millisecond, // Connection timeout (not RPC call timeout)
	})
	if err != nil {
		log.Fatalf("Failed to create mainnet client: %v", err)
	}
	defer tronClient.Close()

	addr, err := types.NewAddress("TDUiUScimQNfmD1F76Uq6YaXbofCVuAvxH")
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}

	fmt.Printf("\nQuerying account information...\n")
	queryStart := time.Now()
	ac, err := tronClient.GetAccount(context.Background(), addr)
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

	// Query transaction info
	txId := "44519f26abfdc64c4a56fc85122f62279124bb12a41ce26ea65e3ab370d75ca5"
	fmt.Printf("\nQuerying transaction %s...\n", txId)
	txInfo, err := tronClient.GetTransactionInfoById(context.Background(), txId)
	if err != nil {
		log.Printf("Failed to get transaction info: %v\n", err)
	} else {
		fmt.Printf("\nTransaction Information:\n")
		fmt.Printf("====================\n")
		fmt.Printf("Block Number: %d\n", txInfo.GetBlockNumber())
		fmt.Printf("Result: %v\n", txInfo.GetResult())
	}

	// Show available node endpoints for reference
	fmt.Printf("\nAvailable Node Endpoints:\n")
	fmt.Printf("========================\n")
	fmt.Printf("Mainnet endpoint: grpc.trongrid.io:50051\n")
	fmt.Printf("Shasta endpoint: grpc.shasta.trongrid.io:50051\n")
	fmt.Printf("Nile endpoint: nile.trongrid.io:50051\n")
}
