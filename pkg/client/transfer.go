package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) CreateTransferTransaction(ctx context.Context, from, to types.Address, amount int64) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "transfer", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {

		return client.CreateTransaction2(ctx, &core.TransferContract{
			OwnerAddress: from.Bytes(),
			ToAddress:    to.Bytes(),
			Amount:       amount,
		})
	})
}
