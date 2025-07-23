package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/helper"

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
	tx, err := tronClient.DeployContract(ctx, owner.Address(),
		bytecode,
		string(abi),
		"MyContract",
		1000000,
		100,
		"MyToken",       // constructor parameter 1: token name
		"MTK",           // constructor parameter 2: token symbol
		uint64(1000000), // constructor parameter 3: total supply
	)
	if err != nil {
		fmt.Printf("Failed to deploy contract: %v\n", err)
		return
	}

	signed, err := owner.Sign(tx.GetTransaction())
	if err != nil {
		fmt.Printf("Failed to sign transaction: %v\n", err)
		return
	}
	txid := helper.GetTxid(signed)
	log.Printf("Signed transaction: %s", txid)

	ret, err := tronClient.BroadcastTransaction(ctx, signed)
	if err != nil {
		fmt.Printf("Failed to broadcast transaction: %v\n", err)
		return
	}
	fmt.Printf("Transaction broadcasted successfully! Transaction ID: %s\nRet: %v\n", txid, ret)
	info, err := tronClient.WaitForTransactionInfo(ctx, txid)
	if err != nil {
		fmt.Printf("Failed to wait for transaction info: %v\n", err)
		return
	}
	fmt.Printf("Contract deployed successfully! Transaction ID: %s\nContract address: %s", txid, types.MustNewAddressFromBytes(info.ContractAddress).String())

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
