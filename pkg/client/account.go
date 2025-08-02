// This package contains raw gRPC calls with minimal business logic
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Account-related gRPC calls

// GetAccount gets account information by address
func (c *Client) GetAccount(ctx context.Context, req *core.Account) (*core.Account, error) {
	return c.grpcGenericCallWrapper(ctx, "get account", func(client api.WalletClient, ctx context.Context) (*core.Account, error) {
		return client.GetAccount(ctx, req)
	})
}

// GetAccountById gets account information by account ID
func (c *Client) GetAccountById(ctx context.Context, req *core.Account) (*core.Account, error) {
	return c.grpcGenericCallWrapper(ctx, "get account by id", func(client api.WalletClient, ctx context.Context) (*core.Account, error) {
		return client.GetAccountById(ctx, req)
	})
}

// GetAccountNet gets account network information (bandwidth usage)
func (c *Client) GetAccountNet(ctx context.Context, req *core.Account) (*api.AccountNetMessage, error) {
	return c.grpcGenericCallWrapper(ctx, "get account net", func(client api.WalletClient, ctx context.Context) (*api.AccountNetMessage, error) {
		return client.GetAccountNet(ctx, req)
	})
}

// GetAccountResource gets account resource information (energy usage)
func (c *Client) GetAccountResource(ctx context.Context, req *core.Account) (*api.AccountResourceMessage, error) {
	return c.grpcGenericCallWrapper(ctx, "get account resource", func(client api.WalletClient, ctx context.Context) (*api.AccountResourceMessage, error) {
		return client.GetAccountResource(ctx, req)
	})
}

// CreateAccount2 creates a new account (v2 - preferred)
func (c *Client) CreateAccount2(ctx context.Context, req *core.AccountCreateContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "create account2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateAccount2(ctx, req)
	})
}

// UpdateAccount2 updates account information (v2 - preferred)
func (c *Client) UpdateAccount2(ctx context.Context, req *core.AccountUpdateContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "update account2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateAccount2(ctx, req)
	})
}

// SetAccountId sets account ID
func (c *Client) SetAccountId(ctx context.Context, req *core.SetAccountIdContract) (*core.Transaction, error) {
	return c.grpcGenericCallWrapper(ctx, "set account id", func(client api.WalletClient, ctx context.Context) (*core.Transaction, error) {
		return client.SetAccountId(ctx, req)
	})
}

// AccountPermissionUpdate updates account permissions
func (c *Client) AccountPermissionUpdate(ctx context.Context, req *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "account permission update", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.AccountPermissionUpdate(ctx, req)
	})
}
