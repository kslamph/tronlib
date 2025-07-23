package main

import (
	"context"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/helper"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {

	client, err := client.NewClient(client.ClientConfig{
		NodeAddress: "grpc.nile.trongrid.io:50051",
		Timeout:     15 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	sender, err := types.NewAccountFromPrivateKey("69004ce41c53bcddab3f74d5d358d0b5099e0d536e72c9b551b1420080296f21")
	if err != nil {
		log.Fatalf("Failed to create sender: %v", err)
	}

	tx, err := client.CreateTransferTransaction(context.Background(), sender.Address().String(), "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x", 2066666666666666660)
	if err != nil {
		log.Fatalf("Failed to create transfer transaction: %v", err)
	}

	signed, err := sender.Sign(tx.GetTransaction())
	log.Printf("signed txid: %s\n", helper.GetTxid(signed))

	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	rt, err := client.BroadcastTransaction(context.Background(), signed)
	if err != nil {
		log.Fatalf("Failed to broadcast transaction: %v", err)
	}
	log.Printf("Transaction broadcasted: %v", rt)

	txInfo, err := client.WaitForTransactionInfo(context.Background(), helper.GetTxid(signed))
	if err != nil {
		log.Fatalf("Failed to wait for transaction info: %v", err)
	}
	log.Printf("Transaction info: %v", txInfo)

}
