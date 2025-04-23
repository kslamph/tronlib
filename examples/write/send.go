package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/transaction"
	"github.com/kslamph/tronlib/pkg/types"
)

func createClient() (*client.Client, error) {
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
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return client, nil
}

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading environment variables directly")
	}

	// Define command-line flag for the transaction command (no default)
	command := flag.String("command", "", "Transaction command (transfer, freeze0, freeze1, unfreeze0, unfreeze1, delegate0, delegate1, reclaim0, reclaim1)")
	flag.Parse()

	// Check if command was provided
	if *command == "" {
		fmt.Println("Error: -command flag is required.\nUsage: go run main.go -command <transfer|freeze0|freeze1|unfreeze0|unfreeze1|delegate0|delegate1|reclaim0|reclaim1>")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize client with options
	client, err := createClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Get private key from environment variable
	privateKey := os.Getenv("TRON_PRIVATE_KEY")
	if privateKey == "" {
		log.Fatalf("TRON_PRIVATE_KEY environment variable not set")
	}

	// Create sender account from private key
	senderAccount, err := types.NewAccountFromPrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Failed to create sender account: %v", err)
	}

	// Create receiver address
	receiverAddr, err := types.NewAddress("TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu")
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}

	// Create new transaction
	tx := transaction.NewTransaction(client, senderAccount)

	// Transfer 1 TRX (1_000_000 SUN = 1 TRX)

	// Execute the transaction based on the command flag
	log.Printf("Executing command: %s\n", *command)
	switch *command {
	case "transfer":
		// Transfer 1 TRX (1_000_000 SUN = 1 TRX)
		err = tx.TransferTRX(senderAccount.Address(), receiverAddr, 1_000_000)
	case "freeze0":
		// Freeze 1 TRX for Bandwidth (ResourceCode 0)
		err = tx.Freeze(senderAccount.Address(), 1_000_000, 0)
	case "freeze1":
		// Freeze 1 TRX for Energy (ResourceCode 1)
		err = tx.Freeze(senderAccount.Address(), 1_000_000, 1)

	case "unfreeze0":
		// Unfreeze Energy (ResourceCode 1) - Ensure you have frozen first
		err = tx.Unfreeze(senderAccount.Address(), 1_000_000, 0) // ResourceCode 1 for Energy
	case "unfreeze1":
		err = tx.Unfreeze(senderAccount.Address(), 1_000_000, 1) // ResourceCode 1 for Energy
	case "delegate0":
		// Delegate 1 TRX Bandwidth (ResourceCode 0)
		err = tx.Delegate(senderAccount.Address(), receiverAddr, 1_000_000, 0)
	case "delegate1":
		// Delegate 1 TRX Energy (ResourceCode 1)
		err = tx.Delegate(senderAccount.Address(), receiverAddr, 1_000_000, 1)
	case "reclaim0":
		// Reclaim delegated Bandwidth (ResourceCode 0) - Ensure you have delegated first
		err = tx.Reclaim(senderAccount.Address(), receiverAddr, 1_000_000, 0)
	case "reclaim1":
		// Reclaim delegated Energy (ResourceCode 1) - Ensure you have delegated first
		err = tx.Reclaim(senderAccount.Address(), receiverAddr, 1_000_000, 1) // ResourceCode 1 for Energy
	default:
		log.Fatalf("Invalid command: %s. Use transfer, freeze, unfreeze, delegate0, delegate1, reclaim0, or reclaim1.", *command)
	}

	if err != nil {
		log.Fatalf("Failed to create transaction for command '%s': %v", *command, err)
	}

	// Set transaction parameters
	tx.SetFeelimit(10_000_000) // 10 TRX
	tx.SetExpiration(30)       // default is 60 seconds
	// time.Sleep(2 * time.Second) //expired transaction should failed when broadcast
	// Sign transaction
	err = tx.Sign(senderAccount)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	// Broadcast transaction
	err = tx.Broadcast()
	if err != nil {
		log.Fatalf("Failed to broadcast transaction: %v", err)
	}

	// Get receipt
	receipt := tx.GetReceipt()
	fmt.Printf("Transaction successful!\n")
	fmt.Printf("Transaction ID: %s\n", receipt.TxID)
	fmt.Printf("Result: %v\n", receipt.Result)
	if receipt.Message != "" {
		fmt.Printf("Message: %s\n", receipt.Message)
	}
}
