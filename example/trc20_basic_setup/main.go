// This snippet is from docs/trc20.md
// Basic Setup
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	// Connect to TRON network
	cli, err := client.NewClient("grpc://grpc.trongrid.io:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	// USDT contract address on mainnet
	token, err := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err != nil {
		log.Fatal(err)
	}

	// Create TRC20 manager
	trc20Mgr := cli.TRC20(token)
	if trc20Mgr == nil {
		log.Fatal("Failed to create TRC20 manager")
	}

	ctx := context.Background()

	// The manager automatically fetches and caches token metadata
	name, _ := trc20Mgr.Name(ctx)
	symbol, _ := trc20Mgr.Symbol(ctx)
	decimals, _ := trc20Mgr.Decimals(ctx)

	fmt.Printf("Token: %s (%s) with %d decimals\n", name, symbol, decimals)
	// Output: Token: Tether USD (USDT) with 6 decimals
}
