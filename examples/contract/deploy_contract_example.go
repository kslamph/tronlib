package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/transaction"
	"github.com/kslamph/tronlib/pkg/types"
)

func deployContractExample() {
	// Example of using the new high-level DeployContract function

	// 1. Create client
	config := client.ClientConfig{
		NodeAddress:     "grpc.trongrid.io:50051", // Replace with your node
		Timeout:         30 * time.Second,
		InitConnections: 1,
		MaxConnections:  5,
	}

	tronClient, err := client.NewClient(config)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer tronClient.Close()

	// 2. Create account from private key
	owner, err := types.NewAccountFromPrivateKey("your_private_key_here")
	if err != nil {
		fmt.Printf("Failed to create account: %v\n", err)
		return
	}

	// 3. Load contract bytecode and ABI
	bytecode, err := loadContractBytecode("path/to/contract.bin")
	if err != nil {
		fmt.Printf("Failed to load bytecode: %v\n", err)
		return
	}

	abi, err := loadContractABI("path/to/contract.abi")
	if err != nil {
		fmt.Printf("Failed to load ABI: %v\n", err)
		return
	}

	// 4. Deploy contract using the new high-level function
	ctx := context.Background()

	// Example 1: Deploy contract without constructor parameters
	tx := transaction.NewTransaction(tronClient).
		SetOwner(owner.Address()).
		DeployContract(
			ctx,
			bytecode,     // contract bytecode
			abi,          // contract ABI
			"MyContract", // contract name
			1000000,      // origin energy limit
			100,          // consume user resource percent (100%)
		).
		SetFeelimit(150000000).
		Sign(owner).
		Broadcast()

	receipt := tx.GetReceipt()
	if receipt.Err != nil {
		fmt.Printf("Contract deployment failed: %v\n", receipt.Err)
		return
	}

	fmt.Printf("Contract deployed successfully! Transaction ID: %s\n", receipt.TxID)

	// Example 2: Deploy contract with constructor parameters
	tx2 := transaction.NewTransaction(tronClient).
		SetOwner(owner.Address()).
		DeployContract(
			ctx,
			bytecode,          // contract bytecode
			abi,               // contract ABI
			"MyTokenContract", // contract name
			1000000,           // origin energy limit
			100,               // consume user resource percent (100%)
			"MyToken",         // constructor parameter 1: token name
			"MTK",             // constructor parameter 2: token symbol
			uint64(1000000),   // constructor parameter 3: total supply
		).
		SetFeelimit(150000000).
		Sign(owner).
		Broadcast()

	receipt2 := tx2.GetReceipt()
	if receipt2.Err != nil {
		fmt.Printf("Token contract deployment failed: %v\n", receipt2.Err)
		return
	}

	fmt.Printf("Token contract deployed successfully! Transaction ID: %s\n", receipt2.TxID)
}

// Helper functions to load contract files
func loadContractBytecode(binPath string) ([]byte, error) {
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read contract bin: %w", err)
	}

	// Remove any whitespace and decode hex
	binString := strings.TrimSpace(string(binData))
	bytecode, err := hex.DecodeString(binString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode contract bin: %w", err)
	}

	return bytecode, nil
}

func loadContractABI(abiPath string) ([]byte, error) {
	abi, err := os.ReadFile(abiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read contract abi: %w", err)
	}

	return abi, nil
}
