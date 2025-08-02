// package client provides 1:1 wrappers around WalletClient gRPC methods
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Transaction related gRPC calls

// CreateTransaction2 creates a transfer transaction (v2 - preferred)
func (c *Client) CreateTransaction2(ctx context.Context, req *core.TransferContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "create transaction2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.CreateTransaction2(ctx, req)
	})
}

// BroadcastTransaction broadcasts a signed transaction
func (c *Client) BroadcastTransaction(ctx context.Context, req *core.Transaction) (*api.Return, error) {
	return grpcGenericCallWrapper(c, ctx, "broadcast transaction", func(client api.WalletClient, ctx context.Context) (*api.Return, error) {
		return client.BroadcastTransaction(ctx, req)
	})
}

// GetTransactionById gets transaction by ID
func (c *Client) GetTransactionById(ctx context.Context, req *api.BytesMessage) (*core.Transaction, error) {
	return grpcGenericCallWrapper(c, ctx, "get transaction by id", func(client api.WalletClient, ctx context.Context) (*core.Transaction, error) {
		return client.GetTransactionById(ctx, req)
	})
}

// GetTransactionInfoById gets transaction info by ID
func (c *Client) GetTransactionInfoById(ctx context.Context, req *api.BytesMessage) (*core.TransactionInfo, error) {
	return grpcGenericCallWrapper(c, ctx, "get transaction info by id", func(client api.WalletClient, ctx context.Context) (*core.TransactionInfo, error) {
		return client.GetTransactionInfoById(ctx, req)
	})
}

// GetTransactionCountByBlockNum gets transaction count by block number
func (c *Client) GetTransactionCountByBlockNum(ctx context.Context, req *api.NumberMessage) (*api.NumberMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get transaction count by block num", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.GetTransactionCountByBlockNum(ctx, req)
	})
}

// GetTransactionSignWeight gets transaction signature weight
func (c *Client) GetTransactionSignWeight(ctx context.Context, req *core.Transaction) (*api.TransactionSignWeight, error) {
	return grpcGenericCallWrapper(c, ctx, "get transaction sign weight", func(client api.WalletClient, ctx context.Context) (*api.TransactionSignWeight, error) {
		return client.GetTransactionSignWeight(ctx, req)
	})
}

// GetTransactionApprovedList gets transaction approved list
func (c *Client) GetTransactionApprovedList(ctx context.Context, req *core.Transaction) (*api.TransactionApprovedList, error) {
	return grpcGenericCallWrapper(c, ctx, "get transaction approved list", func(client api.WalletClient, ctx context.Context) (*api.TransactionApprovedList, error) {
		return client.GetTransactionApprovedList(ctx, req)
	})
}

// CreateCommonTransaction creates a common transaction
func (c *Client) CreateCommonTransaction(ctx context.Context, req *core.Transaction) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "create common transaction", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateCommonTransaction(ctx, req)
	})
}
