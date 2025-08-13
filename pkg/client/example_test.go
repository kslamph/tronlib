package client_test

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
)

// ExampleClient_SignAndBroadcast demonstrates customizing broadcast options.
func ExampleClient_SignAndBroadcast() {
	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	// Some transaction built elsewhere; nil here just to illustrate API.
	var tx any

	opts := client.DefaultBroadcastOptions()
	opts.FeeLimit = 100_000_000
	opts.WaitForReceipt = true
	opts.WaitTimeout = 20 * time.Second
	opts.PollInterval = 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, _ = cli.SignAndBroadcast(ctx, tx, opts, nil)
}

// ExampleClient_Simulate demonstrates read-only simulation of a transaction.
func ExampleClient_Simulate() {
	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	// Some transaction built elsewhere; nil here just to illustrate API.
	var tx any

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, _ = cli.Simulate(ctx, tx)
}
