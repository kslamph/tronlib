package main

import (
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

	// Create and prepare the transaction
	tx := transaction.NewTransaction(tronClient).
		SetOwner(address).                    // Set the sender
		TransferTRX(receiverAddr, 1_000_000). // Transfer 1 TRX
		SetFeelimit(10_000_000)               // Set fee limit to 10 TRX

	if tx.GetReceipt().Err != nil {
		log.Fatalf("Failed to create transaction: %v", tx.GetReceipt().Err)
	}

	fmt.Println("\nSigning transaction using Cloud KMS...")
	// Sign the transaction using KMS and broadcast
	receipt := tx.Sign(kmsAccount).Broadcast().GetReceipt()
	if receipt.Err != nil {
		log.Fatalf("Transaction failed: %v", receipt.Err)
	}

	fmt.Printf("\nTransaction successful!\n")
	fmt.Printf("Transaction ID: %s\n", receipt.TxID)
	fmt.Printf("Result: %v\n", receipt.Result)
	if receipt.Message != "" {
		fmt.Printf("Message: %s\n", receipt.Message)
	}

	fmt.Println("\nWaiting for transaction confirmation...")
	// Wait for transaction confirmation (timeout after 10 retries)
	confirmation, err := tronClient.WaitForTransactionInfo(receipt.TxID, 10)
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}
	fmt.Printf("\nTransaction Information:\n")
	fmt.Printf("====================\n")
	fmt.Println(confirmation)
}
