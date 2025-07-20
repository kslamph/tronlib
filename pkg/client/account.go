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
	if c.isClosed() {
		return nil, fmt.Errorf("get account net failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get account net failed: %w", ErrContextCancelled)
	default:
	}

	// Validate input
	if account == nil {
		return nil, fmt.Errorf("get account net failed: account address is nil")
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get account net: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	client := api.NewWalletClient(conn)
	result, err := client.GetAccountNet(ctx, &core.Account{
		Address: account.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account net: %w", err)
	}

	return result, nil
}

func (c *Client) GetAccountResource(ctx context.Context, account *types.Address) (*api.AccountResourceMessage, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("get account resource failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get account resource failed: %w", ErrContextCancelled)
	default:
	}

	// Validate input
	if account == nil {
		return nil, fmt.Errorf("get account resource failed: account address is nil")
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get account resource: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	client := api.NewWalletClient(conn)
	result, err := client.GetAccountResource(ctx, &core.Account{
		Address: account.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account resource: %w", err)
	}

	return result, nil
}
