// This snippet is from docs/quickstart.md
// TRC20 Token Transfer
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

func main() {
	// Connect and setup
	cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer cli.Close()

	signer, err := signer.NewPrivateKeySigner("69004ce41c53bcddab3f74d5d358d0b5099e0d536e72c9b551b1420080296f21")
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	from := signer.Address()
	ctx := context.Background()

	// Create TRC20 manager for test TRC20 token
	usdtAddr, _ := types.NewAddress("TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK")
	trc20Mgr := cli.TRC20(usdtAddr)
	if trc20Mgr == nil {
		log.Fatal("Failed to create TRC20 manager")
	}

	// Check balance
	balance, err := trc20Mgr.BalanceOf(ctx, from)
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}

	fmt.Printf("USDT Balance: %s\n", balance.String())

	// Transfer
	recipient, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	amount := decimal.NewFromFloat(10.5)

	tx, err := trc20Mgr.Transfer(ctx, from, recipient, amount)
	if err != nil {
		log.Fatalf("Failed to build transfer: %v", err)
	}

	opts := client.DefaultBroadcastOptions()
	opts.FeeLimit = 50_000_000
	opts.WaitForReceipt = true

	result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}

	fmt.Printf("✅ Success! TxID: %s\n", result.TxID)
}
