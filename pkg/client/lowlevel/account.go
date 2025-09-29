// This package contains raw gRPC calls with minimal business logic
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Account-related gRPC calls

// GetAccount gets account information by address
func GetAccount(cp ConnProvider, ctx context.Context, req *core.Account) (*core.Account, error) {
	return Call(cp, ctx, "get account", func(client api.WalletClient, ctx context.Context) (*core.Account, error) {
		return client.GetAccount(ctx, req)
	})
}

// GetAccountById gets account information by account ID
func GetAccountById(cp ConnProvider, ctx context.Context, req *core.Account) (*core.Account, error) {
	return Call(cp, ctx, "get account by id", func(client api.WalletClient, ctx context.Context) (*core.Account, error) {
		return client.GetAccountById(ctx, req)
	})
}

// GetAccountNet gets account network information (bandwidth usage)
func GetAccountNet(cp ConnProvider, ctx context.Context, req *core.Account) (*api.AccountNetMessage, error) {
	return Call(cp, ctx, "get account net", func(client api.WalletClient, ctx context.Context) (*api.AccountNetMessage, error) {
		return client.GetAccountNet(ctx, req)
	})
}

// GetAccountResource gets account resource information (energy usage)
func GetAccountResource(cp ConnProvider, ctx context.Context, req *core.Account) (*api.AccountResourceMessage, error) {
	return Call(cp, ctx, "get account resource", func(client api.WalletClient, ctx context.Context) (*api.AccountResourceMessage, error) {
		return client.GetAccountResource(ctx, req)
	})
}

// CreateAccount2 creates a new account (v2 - preferred)
func CreateAccount2(cp ConnProvider, ctx context.Context, req *core.AccountCreateContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "create account2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateAccount2(ctx, req)
	})
}

// UpdateAccount2 updates account information (v2 - preferred)
func UpdateAccount2(cp ConnProvider, ctx context.Context, req *core.AccountUpdateContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "update account2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateAccount2(ctx, req)
	})
}

// SetAccountId sets account ID
func SetAccountId(cp ConnProvider, ctx context.Context, req *core.SetAccountIdContract) (*core.Transaction, error) {
	return Call(cp, ctx, "set account id", func(client api.WalletClient, ctx context.Context) (*core.Transaction, error) {
		return client.SetAccountId(ctx, req)
	})
}

// AccountPermissionUpdate updates account permissions
func AccountPermissionUpdate(cp ConnProvider, ctx context.Context, req *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "account permission update", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.AccountPermissionUpdate(ctx, req)
	})
}

// GetAccountBalance gets account balance
func GetAccountBalance(cp ConnProvider, ctx context.Context, req *core.AccountBalanceRequest) (*core.AccountBalanceResponse, error) {
	return Call(cp, ctx, "get account balance", func(client api.WalletClient, ctx context.Context) (*core.AccountBalanceResponse, error) {
		return client.GetAccountBalance(ctx, req)
	})
}

// GetBlockBalanceTrace gets block balance trace
func GetBlockBalanceTrace(cp ConnProvider, ctx context.Context, req *core.BlockBalanceTrace_BlockIdentifier) (*core.BlockBalanceTrace, error) {
	return Call(cp, ctx, "get block balance trace", func(client api.WalletClient, ctx context.Context) (*core.BlockBalanceTrace, error) {
		return client.GetBlockBalanceTrace(ctx, req)
	})
}
