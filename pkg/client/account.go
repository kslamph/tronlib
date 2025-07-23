package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) GetAccount(ctx context.Context, address *types.Address) (*core.Account, error) {
	// Validate input
	if address == nil {
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
		Address: address.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return result, nil
}

func (c *Client) GetAccountNet(ctx context.Context, address *types.Address) (*api.AccountNetMessage, error) {
	// Validate input
	if address == nil {
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
		Address: address.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account net: %w", err)
	}

	return result, nil
}

func (c *Client) GetAccountResource(ctx context.Context, address *types.Address) (*api.AccountResourceMessage, error) {
	// Validate input
	if address == nil {
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
		Address: address.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account resource: %w", err)
	}

	return result, nil
}

// UpdateAccount2 updates account information using AccountUpdateContract
func (c *Client) UpdateAccount2(ctx context.Context, address types.Address, accountName string) (*api.TransactionExtention, error) {

	contract := &core.AccountUpdateContract{
		OwnerAddress: address.Bytes(),
		AccountName:  []byte(accountName),
	}
	return c.grpcCallWrapper(ctx, "update account2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateAccount2(ctx, contract)
	})
}

// AccountPermissionUpdate updates account permissions using AccountPermissionUpdateContract
func (c *Client) AccountPermissionUpdate(ctx context.Context, contract *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	if contract == nil {
		return nil, fmt.Errorf("AccountPermissionUpdate failed: contract is nil")
	}
	return c.grpcCallWrapper(ctx, "account permission update", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.AccountPermissionUpdate(ctx, contract)
	})
}
