package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Multi-signature TRX transfer example with 2 keys and permission ID 3
// This demonstrates how to create and sign a transaction that requires multiple signatures
// using a custom permission ID (3) for advanced multi-signature scenarios.
func main() {
	// Connect to TRON network (using Nile testnet)
	cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	// Create two private key signers for multi-signature
	key1, err := signer.NewPrivateKeySigner("dbd76c6e1f8488b6298b4a89699c63109e609e815b5c05575e4a09470a07f374") // Example private key 1
	if err != nil {
		log.Fatal(err)
	}

	key2, err := signer.NewPrivateKeySigner("a392604efc3a29f1d2e5f5a6c4c1c2b3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9") // Example private key 2
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Key 1 Address: %s\n", key1.Address().String())
	fmt.Printf("Key 2 Address: %s\n", key2.Address().String())

	// Define transaction parameters
	from := key1.Address()                                            // The account that owns the permission (or one of the signers)
	to, err := types.NewAddress("TQGzC6c2rJ8G8qj5p7J5q9p2w4r8g7y6u3") // Example recipient address
	if err != nil {
		log.Fatal(err)
	}

	// Create TRX transfer transaction
	amount := int64(1000000) // 1 TRX (in SUN)
	tx, err := cli.Account().TransferTRX(context.Background(), from, to, amount)
	if err != nil {
		log.Fatal(err)
	}

	// Set permission ID to 3 for custom multi-signature permission using the helper function
	// IMPORTANT: This must be done BEFORE signing, as the signature is calculated based on
	// the transaction's raw data including the permission ID
	// Permission ID 3 typically represents a custom permission level that can be
	// configured for multi-signature accounts, allowing for complex authorization schemes
	err = utils.SetPermissionID(tx, 3)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Transaction created with permission ID %d (multi-signature required)\n", tx.Transaction.RawData.Contract[0].PermissionId)

	// Sign with first key using the high-level SignTx function
	fmt.Println("Signing with key 1...")
	err = signer.SignTx(key1, tx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Signed with key 1")

	// Sign with second key using the high-level SignTx function
	fmt.Println("Signing with key 2...")
	err = signer.SignTx(key2, tx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Signed with key 2")

	// Prepare broadcast options
	// Note: Permission ID is already set in the transaction raw data above
	// The transaction has been signed with the correct permission ID
	// We don't pass any signers to SignAndBroadcast since we've already signed manually
	opts := client.DefaultBroadcastOptions()

	// Broadcast the fully signed transaction using high-level API
	fmt.Println("Broadcasting multi-signed transaction with permission ID 3...")
	result, err := cli.SignAndBroadcast(context.Background(), tx, opts)
	if err != nil {
		log.Fatal(err)
	}

	if result.Success {
		fmt.Printf("✅ Multi-signature transaction successful: %s\n", result.TxID)
	} else {
		fmt.Printf("❌ Multi-signature transaction failed: %s - %s\n", result.Code, result.Message)
	}
}
