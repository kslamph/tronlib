package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) GetAccount(ctx context.Context, account *types.Address) (*core.Account, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetAccount(ctx, &core.Account{
		Address: account.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account: %v", err)
	}

	return result, nil
}

func (c *Client) GetAccountNet(ctx context.Context, account *types.Address) (*api.AccountNetMessage, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetAccountNet(ctx, &core.Account{
		Address: account.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account net: %v", err)
	}

	return result, nil
}

func (c *Client) GetAccountResource(ctx context.Context, account *types.Address) (*api.AccountResourceMessage, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetAccountResource(ctx, &core.Account{
		Address: account.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account resource: %v", err)
	}

	return result, nil
}
