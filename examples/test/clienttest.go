package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	// Initialize client with configuration

	tronClient, err := client.NewClient(client.DefaultClientConfig())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer tronClient.Close()

	// Wait a bit for initial connections to establish
	// time.Sleep(2 * time.Second)

	// Create target address
	addr, err := types.NewAddress("TYUkwxLiWrt16YLWr5tK7KcQeNya3vhyLM")
	if err != nil {
		log.Fatalf("Failed to create address: %v", err)
	}

	fmt.Printf("\nStarting 500 account queries...\n")
	fmt.Printf("==========================\n\n")

	totalStart := time.Now()
	successCount := 0
	failCount := 0

	// Perform 500 queries
	for i := 0; i < 500; i++ {
		start := time.Now()
		acc, err := tronClient.GetAccount(addr)
		duration := time.Since(start)

		if err != nil {
			failCount++
			fmt.Printf("[%d] FAIL (%v): %v\n", i+1, duration, err)
		} else {
			successCount++
			if i%10 == 0 { // Only print every 10th success to avoid too much output
				fmt.Printf("[%d] SUCCESS (%v): Balance=%d TRX\n",
					i+1, duration, acc.GetBalance()/1_000_000)
			}
		}

		// Small delay between requests to not overwhelm nodes
		time.Sleep(100 * time.Millisecond)
	}

	totalDuration := time.Since(totalStart)
	fmt.Printf("\nTest Summary:\n")
	fmt.Printf("=============\n")
	fmt.Printf("Total Duration: %v\n", totalDuration)
	fmt.Printf("Average Time Per Request: %v\n", totalDuration/500)
	fmt.Printf("Success Rate: %d/%d (%.2f%%)\n",
		successCount, successCount+failCount,
		float64(successCount)*100/float64(successCount+failCount))
}
