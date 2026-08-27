// This snippet is from README.md
// TRC20 Token Transfer
package main

import (
	"context"
	"log"
	"os"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

func main() {
	cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	keyStr := os.Getenv("NILE_TEST_KEY1")
	if keyStr == "" {
		log.Fatal("set NILE_TEST_KEY1 (see integration_test/test.env)")
	}
	signer, err := signer.NewPrivateKeySigner(keyStr)
	if err != nil {
		log.Fatal(err)
	}

	// USDT contract address on mainnet
	token, err := types.NewAddress("TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK")
	if err != nil {
		log.Fatal(err)
	}
	to, err := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		log.Fatal(err)
	}

	// Transfer 10 USDT
	amount := decimal.NewFromInt(10)

	trc20Mgr, err := cli.TRC20Manager(token)
	if err != nil {
		log.Fatal(err)
	}
	tx, err := trc20Mgr.Transfer(context.Background(), signer.Address(), to, amount)
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
