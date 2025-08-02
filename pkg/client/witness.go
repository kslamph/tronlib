// package client provides 1:1 wrappers around WalletClient gRPC methods
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Voting and witness related gRPC calls

// VoteWitnessAccount2 votes for witnesses (v2 - preferred)
func (c *Client) VoteWitnessAccount2(ctx context.Context, req *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "vote witness account2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.VoteWitnessAccount2(ctx, req)
	})
}

// WithdrawBalance2 withdraws balance (claim rewards) (v2 - preferred)
func (c *Client) WithdrawBalance2(ctx context.Context, req *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "withdraw balance2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.WithdrawBalance2(ctx, req)
	})
}

// CreateWitness2 creates a witness (v2 - preferred)
func (c *Client) CreateWitness2(ctx context.Context, req *core.WitnessCreateContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "create witness2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.CreateWitness2(ctx, req)
	})
}

// UpdateWitness2 updates witness information (v2 - preferred)
func (c *Client) UpdateWitness2(ctx context.Context, req *core.WitnessUpdateContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "update witness2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UpdateWitness2(ctx, req)
	})
}

// ListWitnesses gets list of witnesses
func (c *Client) ListWitnesses(ctx context.Context, req *api.EmptyMessage) (*api.WitnessList, error) {
	return grpcGenericCallWrapper(c, ctx, "list witnesses", func(cl api.WalletClient, ctx context.Context) (*api.WitnessList, error) {
		return cl.ListWitnesses(ctx, req)
	})
}

// GetRewardInfo gets reward information
func (c *Client) GetRewardInfo(ctx context.Context, req *api.BytesMessage) (*api.NumberMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get reward info", func(cl api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return cl.GetRewardInfo(ctx, req)
	})
}

// GetBrokerageInfo gets brokerage information
func (c *Client) GetBrokerageInfo(ctx context.Context, req *api.BytesMessage) (*api.NumberMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get brokerage info", func(cl api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return cl.GetBrokerageInfo(ctx, req)
	})
}

// UpdateBrokerage updates brokerage
func (c *Client) UpdateBrokerage(ctx context.Context, req *core.UpdateBrokerageContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "update brokerage", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UpdateBrokerage(ctx, req)
	})
}
