package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) GetAccount(ctx context.Context, account *types.Address) (*core.Account, error) {
	// Validate input
	if account == nil {
		return nil, fmt.Errorf("get account failed: account address is nil")
	}

	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get account: %w", err)
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.GetAccount(ctx, &core.Account{
		Address: account.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return result, nil
}

func (c *Client) GetAccountNet(ctx context.Context, account *types.Address) (*api.AccountNetMessage, error) {
	// Validate input
	if account == nil {
		return nil, fmt.Errorf("get account net failed: account address is nil")
	}

	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get account net: %w", err)
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.GetAccountNet(ctx, &core.Account{
		Address: account.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account net: %w", err)
	}

	return result, nil
}

func (c *Client) GetAccountResource(ctx context.Context, account *types.Address) (*api.AccountResourceMessage, error) {
	// Validate input
	if account == nil {
		return nil, fmt.Errorf("get account resource failed: account address is nil")
	}

	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get account resource: %w", err)
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.GetAccountResource(ctx, &core.Account{
		Address: account.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account resource: %w", err)
	}

	return result, nil
}
