// This snippet is from docs/quickstart.md
// Your First TRX Transfer
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	// Connect to Nile testnet
	cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer cli.Close()

	// Create signer
	privateKey := "69004ce41c53bcddab3f74d5d358d0b5099e0d536e72c9b551b1420080296f21"
	signer, err := signer.NewPrivateKeySigner(privateKey)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	from := signer.Address()
	ctx := context.Background()

	// Check balance
	balance, err := cli.Accounts().GetBalance(ctx, from)
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}

	fmt.Printf("Balance: %.2f TRX\n", float64(balance)/1_000_000)

	// Transfer setup
	to, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	transferAmount := int64(1_000_000) // 1 TRX

	// Build and send transaction
	tx, err := cli.Accounts().TransferTRX(ctx, from, to, transferAmount)
	if err != nil {
		log.Fatalf("Failed to build transaction: %v", err)
	}

	opts := client.DefaultBroadcastOptions()
	opts.WaitForReceipt = true
	opts.WaitTimeout = 30 * time.Second

	result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}

	fmt.Printf("✅ Success! TxID: %s\n", result.TxID)
}
