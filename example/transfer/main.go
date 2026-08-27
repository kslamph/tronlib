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
	cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	key1Str := os.Getenv("NILE_TEST_KEY1")
	if key1Str == "" {
		log.Fatal("set NILE_TEST_KEY1 (see integration_test/test.env)")
	}
	key1, err := signer.NewPrivateKeySigner(key1Str)
	if err != nil {
		log.Fatal(err)
	}
	from := key1.Address()
	to, err := types.NewAddressFromBase58("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		log.Fatal(err)
	}
	tx, err := cli.Account().TransferTRX(context.Background(), from, to, 1000000)
	if err != nil {
		log.Fatal(err)
	}

	ret, err := cli.SignAndBroadcast(context.Background(), tx, client.DefaultBroadcastOptions(), key1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ret)
}
