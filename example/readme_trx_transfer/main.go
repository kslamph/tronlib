// This snippet is from README.md
// Simple TRX Transfer
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	// Connect to TRON node
	cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	// Create signer from private key
	keyStr := os.Getenv("NILE_TEST_KEY1")
	if keyStr == "" {
		log.Fatal("set NILE_TEST_KEY1 (see integration_test/test.env)")
	}
	signer, err := signer.NewPrivateKeySigner(keyStr)
	if err != nil {
		log.Fatal(err)
	}

	// Define addresses
	from := signer.Address()
	to, err := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		log.Fatal(err)
	}

	// Transfer 1 TRX (1,000,000 SUN)
	tx, err := cli.Account().TransferTRX(context.Background(), from, to, 1_000_000)
	if err != nil {
		log.Fatal(err)
	}

	// Sign and broadcast
	result, err := cli.SignAndBroadcast(context.Background(), tx, client.DefaultBroadcastOptions(), signer)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transaction ID: %s\n", result.TxID)
	fmt.Printf("Success: %v\n", result.Success)
}
