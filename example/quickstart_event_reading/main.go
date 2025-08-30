package main

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/eventdecoder"
)

func main() {
	nodeaddr := "grpc://grpc.trongrid.io:50051"

	txid := "f3a3b5240f1857915400f760fc5888a8ae1a0cf34ef4c9705dff1f1f2df3db11"

	cli, _ := client.NewClient(nodeaddr)
	defer cli.Close()

	tx, err := cli.Network().GetTransactionInfoById(context.Background(), txid)
	if err != nil {
		panic(err)
	}

	decodedEvents, err := eventdecoder.DecodeLogs(tx.GetLog())
	if err != nil {
		panic(err)
	}

	for i, decodedEvent := range decodedEvents {
		fmt.Printf("event %d: %s : %s\n", i, decodedEvent.Contract, decodedEvent.EventName)

		for _, param := range decodedEvent.Parameters {
			fmt.Printf("  %s: %v\n", param.Name, param.Value)
		}

	}

}
