package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/helper"
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

	var txExt *api.TransactionExtention
	var signErr error

	log.Printf("Executing command: %s\n", *command)
	switch *command {
	case "transfer":
		txExt, signErr = client.CreateTransferTransaction(ctx, ownerAccount.Address().String(), receiverAddr.String(), 1_000_000)
	case "freeze0":
		txExt, signErr = client.CreateFreezeTransaction(ctx, ownerAccount.Address().String(), 1_000_000, core.ResourceCode_BANDWIDTH)
	case "freeze1":
		txExt, signErr = client.CreateFreezeTransaction(ctx, ownerAccount.Address().String(), 1_000_000, core.ResourceCode_ENERGY)
	case "unfreeze0":
		txExt, signErr = client.CreateUnfreezeTransaction(ctx, ownerAccount.Address().String(), 1_000_000, core.ResourceCode_BANDWIDTH)
	case "unfreeze1":
		txExt, signErr = client.CreateUnfreezeTransaction(ctx, ownerAccount.Address().String(), 1_000_000, core.ResourceCode_ENERGY)
	case "delegate0":
		txExt, signErr = client.CreateDelegateResourceTransaction(ctx, ownerAccount.Address().String(), receiverAddr.String(), 1_000_000, core.ResourceCode_BANDWIDTH, false)
	case "delegate1":
		txExt, signErr = client.CreateDelegateResourceTransaction(ctx, ownerAccount.Address().String(), receiverAddr.String(), 1_000_000, core.ResourceCode_ENERGY, false)
	case "reclaim0":
		txExt, signErr = client.CreateUndelegateResourceTransaction(ctx, ownerAccount.Address().String(), receiverAddr.String(), 1_000_000, core.ResourceCode_BANDWIDTH)
	case "reclaim1":
		txExt, signErr = client.CreateUndelegateResourceTransaction(ctx, ownerAccount.Address().String(), receiverAddr.String(), 1_000_000, core.ResourceCode_ENERGY)
	default:
		log.Fatalf("Invalid command: %s. Use transfer, freeze, unfreeze, delegate0, delegate1, reclaim0, or reclaim1.", *command)
	}

	if signErr != nil {
		log.Fatalf("Failed to create transaction for command '%s': %v", *command, signErr)
	}

	signed, err := ownerAccount.Sign(txExt.GetTransaction())
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	receipt, err := client.BroadcastTransaction(ctx, signed)
	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}
	if !receipt.GetResult() {
		log.Fatalf("Transaction failed: %s", string(receipt.GetMessage()))
	}
	fmt.Printf("Transaction ID: %s\n", helper.GetTxid(signed))
	fmt.Printf("Result: %v\n", receipt.GetResult())
	if receipt.GetMessage() != nil {
		fmt.Printf("Message: %s\n", string(receipt.GetMessage()))
	}

	// Wait for transaction confirmation
	confirmation, err := client.WaitForTransactionInfo(ctx, helper.GetTxid(signed))
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}
	fmt.Printf("\nTransaction Information:\n")
	fmt.Printf("====================\n")
	fmt.Println(confirmation)
}
