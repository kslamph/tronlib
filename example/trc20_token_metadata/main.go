// This snippet demonstrates TRC20 token metadata retrieval
// Get token name, symbol, and decimals information
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

	// Get token metadata
	name, err := trc20Mgr.Name(ctx)
	if err != nil {
		log.Fatal(err)
	}

	symbol, err := trc20Mgr.Symbol(ctx)
	if err != nil {
		log.Fatal(err)
	}

	decimals, err := trc20Mgr.Decimals(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Token Contract: %s\n", tokenAddr.String())
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Symbol: %s\n", symbol)
	fmt.Printf("Decimals: %d\n", decimals)
}
