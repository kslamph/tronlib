// Package lowlevel provides 1:1 wrappers around WalletClient gRPC methods
// This package contains raw gRPC calls with minimal business logic
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
)

// Account-related gRPC calls

// GetAccount gets account information by address
func GetAccount(c *client.Client, ctx context.Context, req *core.Account) (*core.Account, error) {
	return grpcGenericCallWrapper(c, ctx, "get account", func(client api.WalletClient, ctx context.Context) (*core.Account, error) {
		return client.GetAccount(ctx, req)
	})
}

// GetAccountById gets account information by account ID
func GetAccountById(c *client.Client, ctx context.Context, req *core.Account) (*core.Account, error) {
	return grpcGenericCallWrapper(c, ctx, "get account by id", func(client api.WalletClient, ctx context.Context) (*core.Account, error) {
		return client.GetAccountById(ctx, req)
	})
}

// GetAccountNet gets account network information (bandwidth usage)
func GetAccountNet(c *client.Client, ctx context.Context, req *core.Account) (*api.AccountNetMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get account net", func(client api.WalletClient, ctx context.Context) (*api.AccountNetMessage, error) {
		return client.GetAccountNet(ctx, req)
	})
}

// GetAccountResource gets account resource information (energy usage)
func GetAccountResource(c *client.Client, ctx context.Context, req *core.Account) (*api.AccountResourceMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get account resource", func(client api.WalletClient, ctx context.Context) (*api.AccountResourceMessage, error) {
		return client.GetAccountResource(ctx, req)
	})
}

// CreateAccount2 creates a new account (v2 - preferred)
func CreateAccount2(c *client.Client, ctx context.Context, req *core.AccountCreateContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "create account2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateAccount2(ctx, req)
	})
}

// UpdateAccount2 updates account information (v2 - preferred)
func UpdateAccount2(c *client.Client, ctx context.Context, req *core.AccountUpdateContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "update account2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateAccount2(ctx, req)
	})
}

// SetAccountId sets account ID
func SetAccountId(c *client.Client, ctx context.Context, req *core.SetAccountIdContract) (*core.Transaction, error) {
	return grpcGenericCallWrapper(c, ctx, "set account id", func(client api.WalletClient, ctx context.Context) (*core.Transaction, error) {
		return client.SetAccountId(ctx, req)
	})
}

// AccountPermissionUpdate updates account permissions
func AccountPermissionUpdate(c *client.Client, ctx context.Context, req *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "account permission update", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.AccountPermissionUpdate(ctx, req)
	})
}