// This snippet is from docs/trc20.md
// Transfer with Signing and Broadcasting
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

func main() {
	// Connect to TRON network
	cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	// Test TRC20 contract address
	usdtAddr, err := types.NewAddress("TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK")
	if err != nil {
		log.Fatal(err)
	}

	// Create TRC20 manager
	trc20Mgr, err := trc20.NewManager(cli, usdtAddr)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Create signer
	signer, err := signer.NewPrivateKeySigner("69004ce41c53bcddab3f74d5d358d0b5099e0d536e72c9b551b1420080296f21")
	if err != nil {
		log.Fatal(err)
	}

	from := signer.Address()
	to, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

	// Complete transfer workflow
	amount := decimal.NewFromFloat(25.75)

	// Build transaction
	tx, err := trc20Mgr.Transfer(ctx, from, to, amount)
	if err != nil {
		log.Fatalf("Failed to build transfer: %v", err)
	}

	// Configure broadcast options for TRC20 (higher energy needed)
	opts := client.DefaultBroadcastOptions()
	opts.FeeLimit = 50_000_000 // 50 TRX max fee for TRC20 operations
	opts.WaitForReceipt = true
	opts.WaitTimeout = 30 * time.Second

	// Sign and broadcast
	result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
	if err != nil {
		log.Fatalf("Transfer failed: %v", err)
	}

	fmt.Printf("✅ Transfer successful!\n")
	fmt.Printf("Transaction ID: %s\n", result.TxID)
	fmt.Printf("Energy used: %d\n", result.EnergyUsage)
	fmt.Printf("Success: %v\n", result.Success)
}
