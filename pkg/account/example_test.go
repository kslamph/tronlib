package account_test

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// ExampleNewManager demonstrates creating an account manager and performing basic operations.
func ExampleNewManager() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect to TRON node
	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	// Create account manager
	am := account.NewManager(cli)
	from, _ := types.NewAddress("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1")
	to, _ := types.NewAddress("Tyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy2")

	// Get account balance (in SUN units)
	balance, _ := am.GetBalance(ctx, from)
	_ = balance

	// Create TRX transfer transaction (1 TRX = 1,000,000 SUN)
	txExt, _ := am.TransferTRX(ctx, from, to, 1_000_000) // Transfer 1 TRX
	_ = txExt
}
