// This snippet demonstrates how to retrieve and decode events from a transaction ID
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/eventdecoder"
)

func main() {
	// Ask user for transaction ID
	fmt.Print("Enter a transaction ID: ")
	reader := bufio.NewReader(os.Stdin)
	txid, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return
	}

	// Trim whitespace from the input
	txid = strings.TrimSpace(txid)

	// Validate transaction ID format (should be 64 hex characters)
	if len(txid) != 64 {
		fmt.Println("Invalid transaction ID format. Expected 64 hex characters.")
		return
	}

	// Create client connection to TronGrid
	nodeaddr := "grpc://grpc.trongrid.io:50051"
	cli, err := client.NewClient(nodeaddr)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer cli.Close()

	// Retrieve transaction info by ID
	tx, err := cli.Network().GetTransactionInfoById(context.Background(), txid)
	if err != nil {
		fmt.Printf("Failed to retrieve transaction: %v\n", err)
		return
	}

	// Check if transaction has logs
	logs := tx.GetLog()
	if len(logs) == 0 {
		fmt.Println("No events found in this transaction.")
		return
	}

	// Decode all logs in the transaction
	decodedEvents, err := eventdecoder.DecodeLogs(logs)
	if err != nil {
		fmt.Printf("Failed to decode events: %v\n", err)
		return
	}

	// Print decoded events in a clear format
	fmt.Printf("Transaction ID: %s\n", txid)
	fmt.Printf("Number of events found: %d\n\n", len(decodedEvents))

	for i, decodedEvent := range decodedEvents {
		fmt.Printf("Event %d:\n", i+1)
		fmt.Printf("  Contract: %s\n", decodedEvent.Contract)
		fmt.Printf("  Event Name: %s\n", decodedEvent.EventName)
		fmt.Printf("  Parameters:\n")

		for _, param := range decodedEvent.Parameters {
			fmt.Printf("    %s (%s): %v\n", param.Name, param.Type, param.Value)
		}
		fmt.Println()
	}
}
