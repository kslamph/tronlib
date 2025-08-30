// This snippet demonstrates TRC20 total supply checking
// Get the total supply of tokens for a TRC20 contract
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

	// Get token metadata for context
	name, err := trc20Mgr.Name(ctx)
	if err != nil {
		log.Fatal(err)
	}

	symbol, err := trc20Mgr.Symbol(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Get total supply
	totalSupply, err := trc20Mgr.TotalSupply(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Token Contract: %s\n", tokenAddr.String())
	fmt.Printf("Token: %s (%s)\n", name, symbol)
	fmt.Printf("Total Supply: %s tokens\n", totalSupply.String())
}