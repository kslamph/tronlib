package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

func (c *Client) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	return grpcGenericCallWrapper(c, ctx, "broadcast transaction", func(client api.WalletClient, ctx context.Context) (*api.Return, error) {
		return client.BroadcastTransaction(ctx, tx)
	})
}
