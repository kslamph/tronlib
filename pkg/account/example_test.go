package account_test

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// ExampleNewManager demonstrates building a TRX transfer.
func ExampleNewManager() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	am := account.NewManager(cli)
	from, _ := types.NewAddress("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1")
	to, _ := types.NewAddress("Tyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy2")
	_, _ = am.TransferTRX(ctx, from, to, 1_000_000)
}
