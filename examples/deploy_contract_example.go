// Example of deploying a TRC20 contract using the high-level smartcontract package API
// Note: Run this example from the project root directory
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/workflow"
)

func main() {
	// Note: This example requires a private key to be set in the environment
	// For testing purposes, you can use a test key, but never use real keys in examples
	privateKey := "your-private-key-here" // Replace with actual private key for testing

	// Create client
	c, err := client.NewClient(client.DefaultClientConfig("grpc.nile.trongrid.io:50051"))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Create smartcontract manager
	manager := smartcontract.NewManager(c)

	// Create signer
	signer, err := signer.NewPrivateKeySigner(privateKey)
	if err != nil {
		log.Fatalf("Failed to create signer: %v", err)
	}

	// Get owner address from signer
	ownerAddress := signer.Address().String()
	fmt.Printf("Owner address: %s\n", ownerAddress)

	ctx := context.Background()

	// Load TRC20 contract files
	// Note: This example assumes it's run from the project root directory
	contractDir := "cmd/setup_nile_testnet/test_contract/build"
	abiPath := filepath.Join(contractDir, "TRC20.abi")
	binPath := filepath.Join(contractDir, "TRC20.bin")

	abiBytes, err := os.ReadFile(abiPath)
	if err != nil {
		log.Fatalf("Failed to read ABI file: %v", err)
	}

	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		log.Fatalf("Failed to read bytecode file: %v", err)
	}

	// Decode hex bytecode
	bytecode := []byte(strings.TrimSpace(string(binBytes)))

	// Deploy TRC20 contract with constructor parameters
	fmt.Println("Deploying TRC20 contract...")

	// Create deployment transaction using high-level API
	txExt, err := manager.DeployContract(
		ctx,
		ownerAddress,                // ownerAddress
		"My Test Token",             // contractName
		string(abiBytes),            // abi as string
		bytecode,                    // bytecode
		0,                           // callValue
		100,                         // consumeUserResourcePercent
		10000000,                    // originEnergyLimit
		"TestToken",                 // name_
		"TTK",                       // symbol_
		uint8(18),                   // decimals_
		"1000000000000000000000000", // initialSupply_ (1M tokens with 18 decimals)
	)
	if err != nil {
		log.Fatalf("Failed to create deployment transaction: %v", err)
	}

	// Sign and broadcast transaction using workflow
	fmt.Println("Signing and broadcasting transaction...")
	workflowInstance := workflow.NewWorkflow(c, txExt)
	workflowInstance.SetFeeLimit(2000000000)

	// Sign the transaction
	workflowInstance.Sign(signer)
	if err := workflowInstance.GetError(); err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	// Broadcast and wait for confirmation
	txid, success, _, txInfo, err := workflowInstance.Broadcast(ctx, 30) // Wait up to 30 seconds
	if err != nil {
		log.Fatalf("Failed to broadcast transaction: %v", err)
	}

	if !success {
		log.Fatalf("Deployment transaction failed")
	}

	// Extract contract address from transaction info
	if txInfo != nil && len(txInfo.ContractAddress) > 0 {
		fmt.Printf("✅ TRC20 contract deployed successfully!\n")
		fmt.Printf("📍 Contract Address: %s\n", types.MustNewAddressFromBytes(txInfo.ContractAddress))
		fmt.Printf("🔗 Transaction ID: %s\n", txid)
	} else {
		log.Fatalf("Deployment successful but no contract address found")
	}
}
