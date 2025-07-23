package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) CreateTransferTransaction(ctx context.Context, from, to *types.Address, amount int64) (*api.TransactionExtention, error) {
	if from == nil {
		return nil, fmt.Errorf("CreateTransferTransaction failed: from address is nil")
	}
	if to == nil {
		return nil, fmt.Errorf("CreateTransferTransaction failed: to address is nil")
	}
	return c.grpcCallWrapper(ctx, "transfer", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {

		return client.CreateTransaction2(ctx, &core.TransferContract{
			OwnerAddress: from.Bytes(),
			ToAddress:    to.Bytes(),
			Amount:       amount,
		})
	})
}
