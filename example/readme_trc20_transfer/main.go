// This snippet is from README.md
// TRC20 Token Transfer
package main

import (
	"context"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

func main() {
	cli, _ := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	defer cli.Close()

	signer, _ := signer.NewPrivateKeySigner("69004ce41c53bcddab3f74d5d358d0b5099e0d536e72c9b551b1420080296f21")

	// USDT contract address on mainnet
	token, _ := types.NewAddress("TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK")
	to, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

	// Transfer 10 USDT
	amount := decimal.NewFromInt(10)

	tx, err := cli.TRC20(token).Transfer(context.Background(), signer.Address(), to, amount)
	if err != nil {
		log.Fatal(err)
	}

	// Sign and broadcast
	result, err := cli.SignAndBroadcast(context.Background(), tx, client.DefaultBroadcastOptions(), signer)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("TRC20 transfer completed: %s", result.TxID)
}
