package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/transaction"
	"github.com/kslamph/tronlib/pkg/types"
)

func createClient() (*client.Client, error) {
	// Initialize client with configuration
	config := client.ClientConfig{
		NodeAddress: "127.0.0.1:50051",
		Timeout:     10 * time.Second, // 10 second timeout
	}

	// Create client
	client, err := client.NewClient(config)
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

	// Initialize client
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
	ownerAccount, err := types.NewAccountFromPrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Failed to create sender account: %v", err)
	}

	// Create receiver address
	receiverAddr, err := types.NewAddress("TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu")
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}

	ctx := context.Background()

	// Create new transaction
	tx := transaction.NewTransaction(client).SetOwner(ownerAccount.Address())

	// Execute the transaction based on the command flag
	log.Printf("Executing command: %s\n", *command)
	switch *command {
	case "transfer":
		// Transfer 1 TRX (1_000_000 SUN = 1 TRX)
		tx.TransferTRX(ctx, receiverAddr, 1_000_000)
	case "freeze0":
		// Freeze 1 TRX for Bandwidth (ResourceCode 0)
		tx.Freeze(ctx, 1_000_000, core.ResourceCode_BANDWIDTH)
	case "freeze1":
		// Freeze 1 TRX for Energy (ResourceCode 1)
		tx.Freeze(ctx, 1_000_000, core.ResourceCode_ENERGY)
	case "unfreeze0":
		// Unfreeze Bandwidth (ResourceCode 0)
		tx.Unfreeze(ctx, 1_000_000, core.ResourceCode_BANDWIDTH)
	case "unfreeze1":
		// Unfreeze Energy (ResourceCode 1)
		tx.Unfreeze(ctx, 1_000_000, core.ResourceCode_ENERGY)
	case "delegate0":
		// Delegate 1 TRX Bandwidth (ResourceCode 0)
		tx.Delegate(ctx, receiverAddr, 1_000_000, core.ResourceCode_BANDWIDTH)
	case "delegate1":
		// Delegate 1 TRX Energy (ResourceCode 1)
		tx.Delegate(ctx, receiverAddr, 1_000_000, core.ResourceCode_ENERGY)
	case "reclaim0":
		// Reclaim delegated Bandwidth (ResourceCode 0)
		tx.Reclaim(ctx, receiverAddr, 1_000_000, core.ResourceCode_BANDWIDTH)
	case "reclaim1":
		// Reclaim delegated Energy (ResourceCode 1)
		tx.Reclaim(ctx, receiverAddr, 1_000_000, core.ResourceCode_ENERGY)
	default:
		log.Fatalf("Invalid command: %s. Use transfer, freeze, unfreeze, delegate0, delegate1, reclaim0, or reclaim1.", *command)
	}

	if tx.GetReceipt().Err != nil {
		log.Fatalf("Failed to create transaction for command '%s': %v", *command, tx.GetReceipt().Err)
	}

	// Set transaction parameters
	// 30 seconds
	// tx.SetExpiration(60)

	// Get receipt
	receipt := tx.Sign(ownerAccount).Broadcast().GetReceipt()
	if receipt.Err != nil {
		log.Fatalf("Transaction failed: %v", receipt.Err)
	}
	//Err is nil, meaning broadcast was successful
	fmt.Printf("Transaction ID: %s\n", receipt.TxID)
	fmt.Printf("Result: %v\n", receipt.Result)
	if receipt.Message != "" {
		fmt.Printf("Message: %s\n", receipt.Message)
	}

	// Wait for transaction confirmation
	confirmation, err := client.WaitForTransactionInfo(ctx, receipt.TxID, 10)
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}
	fmt.Printf("\nTransaction Information:\n")
	fmt.Printf("====================\n")
	fmt.Println(confirmation)

}
