package main

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	client, err := client.NewClient(client.DefaultClientConfig("grpc.nile.trongrid.io:50051"))
	if err != nil {
		panic(err)
	}
	defer client.Close()
	address := types.MustNewAddressFromBase58("TD89DBZ6wFVLz38yVaz8RjKARzLXj9ziua")

	req := &api.BytesMessage{
		Value: address.Bytes(),
	}

	contract, err := lowlevel.GetContract(client, context.Background(), req)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Contract: %v\n", contract.GetAbi())

	// This is a placeholder for the main function.
	// The actual implementation will be added later.
}
