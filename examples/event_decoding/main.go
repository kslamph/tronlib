package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/parser"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	// Create a new client
	tclient, err := client.NewClient(client.ClientConfig{
		NodeAddress: "127.0.0.1:50051",
		Timeout:     30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer tclient.Close()

	// Get a contract (example: USDT contract)
	contract := getContract(tclient, "TXJgMdjVX5dKiQaUi9QobwNxtSQaFqccvd")
	if contract == nil {
		log.Fatalf("Failed to get contract")
	}

	// Example transaction ID with events
	txID := "60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c"

	// Get transaction info
	transactionInfo, err := tclient.GetTransactionInfoById(context.Background(), txID)
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}

	// Create contracts map
	contractsMap := parser.ContractsSliceToMap([]*types.Contract{contract})

	// Parse transaction info logs - now returns []*TransactionEvent
	transactionEvents := parser.ParseTransactionInfoLog(transactionInfo, contractsMap)

	fmt.Printf("Transaction: %s\n", txID)
	fmt.Printf("Block Number: %d\n", transactionInfo.GetBlockNumber())
	fmt.Printf("Block Timestamp: %d\n", transactionInfo.GetBlockTimeStamp())
	fmt.Printf("Number of Events: %d\n\n", len(transactionEvents))

	// Process each transaction event
	for i, event := range transactionEvents {
		fmt.Printf("Event %d:\n", i+1)
		fmt.Printf("  Block Number: %d\n", event.BlockNumber)
		fmt.Printf("  Block Timestamp: %d\n", event.BlockTimestamp)
		fmt.Printf("  Transaction Hash: %s\n", event.TransactionHash)
		fmt.Printf("  Contract Address: %s\n", event.ContractAddress)
		fmt.Printf("  Event Name: %s\n", event.Event.EventName)

		fmt.Printf("  Parameters:\n")
		for _, param := range event.Event.Parameters {
			fmt.Printf("    %s (%s): %v (indexed: %t)\n",
				param.Name, param.Type, param.Value, param.Indexed)
		}
		fmt.Println()
	}
}

func getContract(tclient *client.Client, address string) *types.Contract {
	contract, err := tclient.NewContractFromAddress(context.Background(), types.MustNewAddress(address))
	if err != nil {
		return nil
	}
	return contract
}
