package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/helper"
	"github.com/kslamph/tronlib/pkg/types"
)

func createClient() (*client.Client, error) {
	// Initialize client with configuration
	config := client.ClientConfig{
		NodeAddress: "grpc.shasta.trongrid.io:50051",
		Timeout:     10 * time.Second, // 10 second timeout
	}

	// Create client
	client, err := client.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return client, nil
}

func setupGCPEnvironment() error {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading environment variables directly")
	}

	// Check required GCP environment variables
	required := []string{
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GCP_PROJECT_ID",
		"GCP_LOCATION_ID",
		"GCP_KEY_RING_ID",
	}

	missing := make([]string, 0)
	for _, env := range required {
		if os.Getenv(env) == "" {
			missing = append(missing, env)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}

func main() {
	// Setup GCP environment
	if err := setupGCPEnvironment(); err != nil {
		log.Fatalf("Failed to setup GCP environment: %v", err)
	}

	// Create GCP KMS client
	kmsClient, err := NewGCPKMSClient(
		os.Getenv("GCP_PROJECT_ID"),
		os.Getenv("GCP_LOCATION_ID"),
		os.Getenv("GCP_KEY_RING_ID"),
	)
	if err != nil {
		log.Fatalf("Failed to create GCP KMS client: %v", err)
	}
	defer kmsClient.Close()

	// Create a KMSAccount using the key
	kmsAccount, err := types.NewKMSAccount(os.Getenv("GCP_KEY_NAME"), kmsClient)
	if err != nil {
		log.Fatalf("Failed to create KMS account: %v", err)
	}

	// Get and display the Tron address
	address := kmsAccount.Address()
	fmt.Printf("\nKMS-based Tron Address: %s\n", address.String())
	fmt.Println("Please fund this address with some test TRX before proceeding.")
	fmt.Println("Press Enter to continue...")
	fmt.Scanln() // Wait for user input

	// Initialize Tron client
	tronClient, err := createClient()
	if err != nil {
		log.Fatalf("Failed to create Tron client: %v", err)
	}
	defer tronClient.Close()

	// Create receiver address
	receiverAddr, err := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}

	ctx := context.Background()

	// Create and prepare the transaction
	tx, err := tronClient.CreateTransferTransaction(ctx, address.String(), receiverAddr.String(), 1_000_000)
	if err != nil {
		log.Fatalf("Failed to create transaction: %v", err)
	}

	fmt.Println("\nSigning transaction using Cloud KMS...")
	// Sign the transaction using KMS and broadcast
	signed, err := kmsAccount.Sign(tx.GetTransaction())
	txid := helper.GetTxid(signed)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}
	receipt, err := tronClient.BroadcastTransaction(ctx, signed)
	if err != nil {
		log.Fatalf("Failed to broadcast transaction: %v", err)
	}

	fmt.Printf("\nTransaction successful!\n")
	fmt.Printf("Transaction ID: %s\n", txid)
	fmt.Printf("Result: %v\n", receipt.Result)
	if receipt.Message != nil {
		fmt.Printf("Message: %s\n", string(receipt.Message))
	}

	fmt.Println("\nWaiting for transaction confirmation...")
	// Wait for transaction confirmation (timeout after 10 retries)
	confirmation, err := tronClient.WaitForTransactionInfo(ctx, txid)
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}
	fmt.Printf("\nTransaction Information:\n")
	fmt.Printf("====================\n")
	fmt.Println(confirmation)
}
