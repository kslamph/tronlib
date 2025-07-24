// Example: Transaction Workflow Usage with Action Chain Pattern
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/workflow"
	"google.golang.org/protobuf/types/known/anypb"
)

func workflowExample() {
	// Initialize client
	clientConfig := client.DefaultClientConfig("grpc.trongrid.io:50051")
	client, err := client.NewClient(clientConfig)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create a signer from private key
	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	signer, err := signer.NewPrivateKeySigner(privateKey)
	if err != nil {
		log.Fatalf("Failed to create signer: %v", err)
	}

	fmt.Printf("Signer Address: %s\n", signer.Address().String())

	// Example 1: Simple Transfer Workflow
	fmt.Println("\n=== Example 1: Simple Transfer with Action Chain ===")
	demonstrateSimpleTransfer(client, signer)

	// Example 2: Smart Contract with Fee Limit
	fmt.Println("\n=== Example 2: Smart Contract with Fee Limit ===")
	demonstrateSmartContractWithFeeLimit(client, signer)

	// Example 3: Multi-Signature Workflow
	fmt.Println("\n=== Example 3: Multi-Signature Workflow ===")
	demonstrateMultiSignature(client, signer)

	// Example 4: Sign Only (No Broadcast)
	fmt.Println("\n=== Example 4: Sign Only for External Broadcast ===")
	demonstrateSignOnly(client, signer)

	// Example 5: Error Handling
	fmt.Println("\n=== Example 5: Error Handling ===")
	demonstrateErrorHandling(client, signer)
}

func demonstrateSimpleTransfer(client *client.Client, signer *signer.PrivateKeySigner) {
	ctx := context.Background()

	// Create a mock transfer transaction
	mockTx := createMockTransferTransaction()

	// Action chain: Set timeout -> Sign -> Broadcast with waiting
	wf := workflow.NewWorkflow(client, mockTx).
		SetTimeout(time.Now().Add(10*time.Minute).UnixMilli()).
		Sign(signer)

	// Check for errors before broadcasting
	if err := wf.GetError(); err != nil {
		fmt.Printf("Workflow error: %v\n", err)
		return
	}

	// Broadcast and wait for confirmation (smart contract transactions only)
	txID, success, txInfo, err := wf.Broadcast(ctx, 30)
	if err != nil {
		fmt.Printf("Broadcast failed: %v\n", err)
		return
	}

	fmt.Printf("Transfer Result:\n")
	fmt.Printf("  TX ID: %s\n", txID)
	fmt.Printf("  Broadcast Success: %t\n", success)
	if txInfo != nil {
		fmt.Printf("  Block Number: %d\n", txInfo.BlockNumber)
		fmt.Printf("  Result: %v\n", txInfo.Result)
	}
}

func demonstrateSmartContractWithFeeLimit(client *client.Client, signer *signer.PrivateKeySigner) {
	ctx := context.Background()

	// Create a mock smart contract transaction
	mockTx := createMockSmartContractTransaction()

	// Action chain: Set fee limit -> Set timeout -> Sign -> Broadcast
	wf := workflow.NewWorkflow(client, mockTx).
		SetFeeLimit(1000000). // 1 TRX fee limit
		SetTimeout(time.Now().Add(1*time.Hour).UnixMilli()).
		Sign(signer)

	if err := wf.GetError(); err != nil {
		fmt.Printf("Workflow error: %v\n", err)
		return
	}

	// Broadcast with waiting for smart contract result
	txID, success, txInfo, err := wf.Broadcast(ctx, 60)
	if err != nil {
		fmt.Printf("Broadcast failed: %v\n", err)
		return
	}

	fmt.Printf("Smart Contract Result:\n")
	fmt.Printf("  TX ID: %s\n", txID)
	fmt.Printf("  Broadcast Success: %t\n", success)
	if txInfo != nil {
		fmt.Printf("  Block Number: %d\n", txInfo.BlockNumber)
		fmt.Printf("  Contract Result: %v\n", txInfo.ContractResult)
	}
}

func demonstrateMultiSignature(client *client.Client, signer1 *signer.PrivateKeySigner) {
	// Create a second signer for multi-signature
	privateKey2 := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	signer2, err := signer.NewPrivateKeySigner(privateKey2)
	if err != nil {
		log.Printf("Failed to create second signer: %v", err)
		return
	}

	// Create a mock transaction
	mockTx := createMockTransferTransaction()

	// Multi-signature workflow: Sign with multiple signers
	wf := workflow.NewWorkflow(client, mockTx).
		SetTimeout(time.Now().Add(30*time.Minute).UnixMilli()).
		Sign(signer1).                    // First signature
		MultiSign(signer2, 1)             // Second signature with permission ID

	if err := wf.GetError(); err != nil {
		fmt.Printf("Multi-signature workflow error: %v\n", err)
		return
	}

	// Get the signed transaction for external handling
	txID, signedTx, err := wf.GetSignedTransaction()
	if err != nil {
		fmt.Printf("Failed to get signed transaction: %v\n", err)
		return
	}

	fmt.Printf("Multi-Signature Result:\n")
	fmt.Printf("  TX ID: %s\n", txID)
	fmt.Printf("  Signatures Count: %d\n", len(signedTx.Transaction.Signature))
	fmt.Printf("  Ready for broadcast: %t\n", len(signedTx.Transaction.Signature) >= 2)
}

func demonstrateSignOnly(client *client.Client, signer *signer.PrivateKeySigner) {
	// Create a mock transaction
	mockTx := createMockTransferTransaction()

	// Sign only workflow - useful for multi-party scenarios
	wf := workflow.NewWorkflow(client, mockTx).
		SetTimeout(time.Now().Add(2*time.Hour).UnixMilli()).
		SetFeeLimit(500000).
		Sign(signer)

	if err := wf.GetError(); err != nil {
		fmt.Printf("Sign-only workflow error: %v\n", err)
		return
	}

	// Get transaction ID and signed transaction for external use
	txID := wf.GetTxid()
	_, signedTx, err := wf.GetSignedTransaction()
	if err != nil {
		fmt.Printf("Failed to get signed transaction: %v\n", err)
		return
	}

	fmt.Printf("Sign-Only Result:\n")
	fmt.Printf("  TX ID: %s\n", txID)
	fmt.Printf("  Transaction signed: %t\n", len(signedTx.Transaction.Signature) > 0)
	fmt.Printf("  Ready for external broadcast or additional signatures\n")

	// The signed transaction can now be:
	// 1. Passed to another application for additional signatures
	// 2. Broadcasted by another service
	// 3. Stored for later processing
}

func demonstrateErrorHandling(client *client.Client, signer *signer.PrivateKeySigner) {
	// Create an invalid transaction (nil raw data)
	invalidTx := &core.Transaction{
		RawData: nil, // This will cause an error
	}

	// Try to use the invalid transaction
	wf := workflow.NewWorkflow(client, invalidTx).
		SetTimeout(time.Now().Add(1*time.Hour).UnixMilli()).
		Sign(signer)

	// Check for errors
	if err := wf.GetError(); err != nil {
		fmt.Printf("Expected error caught: %v\n", err)
		fmt.Printf("Workflow state: %s\n", wf.GetState().String())
	}

	// Try to operate on signed transaction when it's in error state
	txID := wf.GetTxid()
	fmt.Printf("TX ID from error state: '%s' (should be empty)\n", txID)

	// Demonstrate state validation
	validTx := createMockTransferTransaction()
	validWorkflow := workflow.NewWorkflow(client, validTx).
		Sign(signer)

	// Try to set fee limit on signed transaction (should fail)
	validWorkflow.SetFeeLimit(1000000)
	
	if err := validWorkflow.GetError(); err != nil {
		fmt.Printf("State validation error: %v\n", err)
	}
}

// Helper functions to create mock transactions
func createMockTransferTransaction() *core.Transaction {
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type: core.Transaction_Contract_TransferContract,
					Parameter: &anypb.Any{
						Value: []byte("mock_transfer_contract_data"),
					},
				},
			},
			Timestamp:     time.Now().UnixMilli(),
			Expiration:    time.Now().Add(time.Hour).UnixMilli(),
			RefBlockBytes: []byte{0x12, 0x34},
			RefBlockHash:  []byte{0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34},
		},
	}
}

func createMockSmartContractTransaction() *core.Transaction {
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type: core.Transaction_Contract_TriggerSmartContract,
					Parameter: &anypb.Any{
						Value: []byte("mock_smart_contract_data"),
					},
				},
			},
			Timestamp:     time.Now().UnixMilli(),
			Expiration:    time.Now().Add(time.Hour).UnixMilli(),
			RefBlockBytes: []byte{0x12, 0x34},
			RefBlockHash:  []byte{0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34},
			FeeLimit:      1000000, // 1 TRX fee limit for smart contracts
		},
	}
}