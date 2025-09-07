// package lowlevel provides 1:1 wrappers around WalletClient gRPC methods
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Transaction related gRPC calls

// CreateTransaction2 creates a transfer transaction (v2 - preferred)
func CreateTransaction2(cp ConnProvider, ctx context.Context, req *core.TransferContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "create transaction2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.CreateTransaction2(ctx, req)
	})
}

// BroadcastTransaction broadcasts a signed transaction
func BroadcastTransaction(cp ConnProvider, ctx context.Context, req *core.Transaction) (*api.Return, error) {
	return Call(cp, ctx, "broadcast transaction", func(client api.WalletClient, ctx context.Context) (*api.Return, error) {
		return client.BroadcastTransaction(ctx, req)
	})
}

// GetTransactionById gets transaction by ID
func GetTransactionById(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.Transaction, error) {
	return Call(cp, ctx, "get transaction by id", func(client api.WalletClient, ctx context.Context) (*core.Transaction, error) {
		return client.GetTransactionById(ctx, req)
	})
}

// GetTransactionInfoById gets transaction info by ID
func GetTransactionInfoById(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.TransactionInfo, error) {
	return Call(cp, ctx, "get transaction info by id", func(client api.WalletClient, ctx context.Context) (*core.TransactionInfo, error) {
		return client.GetTransactionInfoById(ctx, req)
	})
}

// GetTransactionCountByBlockNum gets transaction count by block number
func GetTransactionCountByBlockNum(cp ConnProvider, ctx context.Context, req *api.NumberMessage) (*api.NumberMessage, error) {
	return Call(cp, ctx, "get transaction count by block num", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.GetTransactionCountByBlockNum(ctx, req)
	})
}

// GetTransactionSignWeight gets transaction signature weight
func GetTransactionSignWeight(cp ConnProvider, ctx context.Context, req *core.Transaction) (*api.TransactionSignWeight, error) {
	return Call(cp, ctx, "get transaction sign weight", func(client api.WalletClient, ctx context.Context) (*api.TransactionSignWeight, error) {
		return client.GetTransactionSignWeight(ctx, req)
	})
}

// GetTransactionApprovedList gets transaction approved list
func GetTransactionApprovedList(cp ConnProvider, ctx context.Context, req *core.Transaction) (*api.TransactionApprovedList, error) {
	return Call(cp, ctx, "get transaction approved list", func(client api.WalletClient, ctx context.Context) (*api.TransactionApprovedList, error) {
		return client.GetTransactionApprovedList(ctx, req)
	})
}

// CreateCommonTransaction creates a common transaction
func CreateCommonTransaction(cp ConnProvider, ctx context.Context, req *core.Transaction) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "create common transaction", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateCommonTransaction(ctx, req)
	})
}
