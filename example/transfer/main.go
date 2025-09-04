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
	cli, _ := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	defer cli.Close()
	key1, _ := signer.NewPrivateKeySigner("69004ce41c53bcddab3f74d5d358d0b5099e0d536e72c9b551b1420080296f21")
	from := key1.Address()
	to := types.MustNewAddressFromBase58("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
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
