// This snippet is from README.md
// Simple TRX Transfer
package main

import (
	"context"
	"fmt"
	"log"

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
	signer, err := signer.NewPrivateKeySigner("69004ce41c53bcddab3f74d5d358d0b5099e0d536e72c9b551b1420080296f21")
	if err != nil {
		log.Fatal(err)
	}

	// Define addresses
	from := signer.Address()
	to, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

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
