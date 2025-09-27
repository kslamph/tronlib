package network_test

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/network"
)

// ExampleNewManager demonstrates fetching node info.
func ExampleNewManager() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, _ := client.NewClient("grpc://grpc.trongrid.io:50051")
	defer cli.Close()

	nm := network.NewManager(cli)
	_, _ = nm.GetNodeInfo(ctx)
}
