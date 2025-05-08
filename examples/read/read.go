package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	start := time.Now()
	fmt.Printf("Starting TRON account query at %v\n\n", start.Format(time.RFC3339))

	// Initialize client with configuration
	config := client.ClientConfig{
		Nodes: []client.NodeConfig{
			{
				Address:   "3.225.171.164:50051",
				RateLimit: client.RateLimit{Times: 100, Window: time.Minute},
			},
			{
				Address:   "grpc.trongrid.io:50051",
				RateLimit: client.RateLimit{Times: 100, Window: time.Minute},
			},
			{
				Address:   "grpc.trongrid.io:50055",
				RateLimit: client.RateLimit{Times: 100, Window: time.Minute},
			},
			{
				Address:   "grpc.nile.trongrid.io:50055",
				RateLimit: client.RateLimit{Times: 100, Window: time.Minute},
			},
			{
				Address:   "5.5.6.9:50051",
				RateLimit: client.RateLimit{Times: 100, Window: time.Minute},
			},
			{
				Address:   "grpc.shasta.trongrid.io:50051",
				RateLimit: client.RateLimit{Times: 100, Window: time.Minute},
			},
		},
		CooldownPeriod:     1 * time.Minute,
		MetricsWindowSize:  3,
		BestNodePercentage: 90,
		TimeoutMs:          1000, // 1 second timeout for RPC calls
	}

	fmt.Printf("Connection Settings:\n")
	fmt.Printf("-------------------\n")
	fmt.Printf("Cooldown period: %v\n", config.CooldownPeriod)
	fmt.Printf("Metrics window size: %d\n", config.MetricsWindowSize)
	fmt.Printf("Best node percentage: %d%%\n", config.BestNodePercentage)
	fmt.Printf("RPC timeout: %d ms\n", config.TimeoutMs)

	fmt.Printf("\nConfigured Nodes:\n")
	fmt.Printf("--------------------\n")
	for i, node := range config.Nodes {
		fmt.Printf("%d. %s (Rate limit: %d per %v)\n", i+1, node.Address, node.RateLimit.Times, node.RateLimit.Window)
	}

	// Create client
	client, err := client.NewTronClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	addr, err := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}

	fmt.Printf("\nQuerying account information...\n")
	fmt.Printf("This operation will automatically retry with different nodes if needed\n")
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

	// Query transaction info with 5 second timeout
	txId := "41fee269a29af3604c4082c48ae372d860170745b1b7dbb92425ac01b52dc7dd"
	fmt.Printf("\nQuerying transaction %s...\n", txId)
	txInfo, err := client.GetTransactionInfoById(txId)
	if err != nil {
		log.Printf("Failed to get transaction info: %v\n", err)
	} else {
		fmt.Printf("\nTransaction Information:\n")
		fmt.Printf("====================\n")
		fmt.Printf("Block Number: %d\n", txInfo.GetBlockNumber())
		fmt.Printf("Result: %v\n", txInfo.GetResult())
	}
}
