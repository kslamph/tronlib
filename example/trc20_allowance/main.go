// This snippet demonstrates TRC20 allowance checking
// Check how much a spender is allowed to spend on behalf of an owner
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

	// Define owner and spender addresses
	ownerAddr, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
	spenderAddr, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

	// Check allowance
	allowance, err := trc20Mgr.Allowance(ctx, ownerAddr, spenderAddr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Owner: %s\n", ownerAddr.String())
	fmt.Printf("Spender: %s\n", spenderAddr.String())
	fmt.Printf("Allowance: %s tokens\n", allowance.String())
}