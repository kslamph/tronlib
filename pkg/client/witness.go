// package client provides 1:1 wrappers around WalletClient gRPC methods
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
)

// Voting and witness related gRPC calls

// VoteWitnessAccount2 votes for witnesses (v2 - preferred)
func VoteWitnessAccount2(c *client.Client, ctx context.Context, req *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "vote witness account2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.VoteWitnessAccount2(ctx, req)
	})
}

// WithdrawBalance2 withdraws balance (claim rewards) (v2 - preferred)
func WithdrawBalance2(c *client.Client, ctx context.Context, req *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "withdraw balance2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.WithdrawBalance2(ctx, req)
	})
}

// CreateWitness2 creates a witness (v2 - preferred)
func CreateWitness2(c *client.Client, ctx context.Context, req *core.WitnessCreateContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "create witness2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateWitness2(ctx, req)
	})
}

// UpdateWitness2 updates witness information (v2 - preferred)
func UpdateWitness2(c *client.Client, ctx context.Context, req *core.WitnessUpdateContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "update witness2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateWitness2(ctx, req)
	})
}

// ListWitnesses gets list of witnesses
func ListWitnesses(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.WitnessList, error) {
	return grpcGenericCallWrapper(c, ctx, "list witnesses", func(client api.WalletClient, ctx context.Context) (*api.WitnessList, error) {
		return client.ListWitnesses(ctx, req)
	})
}

// GetRewardInfo gets reward information
func GetRewardInfo(c *client.Client, ctx context.Context, req *api.BytesMessage) (*api.NumberMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get reward info", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.GetRewardInfo(ctx, req)
	})
}

// GetBrokerageInfo gets brokerage information
func GetBrokerageInfo(c *client.Client, ctx context.Context, req *api.BytesMessage) (*api.NumberMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get brokerage info", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.GetBrokerageInfo(ctx, req)
	})
}

// UpdateBrokerage updates brokerage
func UpdateBrokerage(c *client.Client, ctx context.Context, req *core.UpdateBrokerageContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "update brokerage", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateBrokerage(ctx, req)
	})
}
