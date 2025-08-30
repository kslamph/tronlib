// This snippet demonstrates TRC20 balance checking
// Check the token balance of a specific address
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	// Connect to Nile testnet
	cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	// Test TRC20 contract address
	tokenAddr, _ := types.NewAddress("TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK")

	// Create TRC20 manager
	trc20Mgr, err := trc20.NewManager(cli, tokenAddr)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Address to check balance for
	checkAddr, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

	// Get balance
	balance, err := trc20Mgr.BalanceOf(ctx, checkAddr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Token Contract: %s\n", tokenAddr.String())
	fmt.Printf("Account: %s\n", checkAddr.String())
	fmt.Printf("Balance: %s tokens\n", balance.String())
}