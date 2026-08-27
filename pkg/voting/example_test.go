package voting_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/voting"
)

// ExampleNewManager shows constructing the voting manager.
func ExampleNewManager() {
	if testing.Short() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, _ := client.NewClient("grpc://grpc.trongrid.io:50051")
	defer cli.Close()

	_ = voting.NewManager(cli)
	_ = ctx
}
