package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) CreateTransferTransaction(ctx context.Context, from, to string, amount int64) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "transfer", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		fromAddress, err := types.NewAddressFromHex(from)
		if err != nil {
			return nil, fmt.Errorf("invalid from address: %w", err)
		}
		toAddress, err := types.NewAddressFromHex(to)
		if err != nil {
			return nil, fmt.Errorf("invalid to address: %w", err)
		}

		return client.CreateTransaction2(ctx, &core.TransferContract{
			OwnerAddress: fromAddress.Bytes(),
			ToAddress:    toAddress.Bytes(),
			Amount:       amount,
		})
	})
}
