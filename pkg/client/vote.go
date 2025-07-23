package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

// CreateWithdrawBalanceTransaction creates a withdraw balance transaction (claim rewards)
func (c *Client) CreateWithdrawBalanceTransaction(ctx context.Context, ownerAddress string) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "withdraw balance", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		ownerAddress, err := types.NewAddress(ownerAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid owner address: %w", err)
		}
		return client.WithdrawBalance2(ctx, &core.WithdrawBalanceContract{
			OwnerAddress: ownerAddress.Bytes(),
		})
	})
}

// VoteWitnessAccount2 votes for witnesses using VoteWitnessContract
func (c *Client) VoteWitnessAccount2(ctx context.Context, contract *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	if contract == nil {
		return nil, fmt.Errorf("VoteWitnessAccount2 failed: contract is nil")
	}
	return c.grpcCallWrapper(ctx, "vote witness account2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.VoteWitnessAccount2(ctx, contract)
	})
}

// ListWitnesses retrieves the list of witnesses
func (c *Client) ListWitnesses(ctx context.Context) (*api.WitnessList, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for ListWitnesses: %w", err)
	}
	defer c.pool.Put(conn)
	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	return walletClient.ListWitnesses(ctx, &api.EmptyMessage{})
}

// GetRewardInfo retrieves reward info for an address
func (c *Client) GetRewardInfo(ctx context.Context, address string) (int64, error) {
	if len(address) == 0 {
		return 0, fmt.Errorf("GetRewardInfo failed: address is empty")
	}
	addr, err := types.NewAddress(address)
	if err != nil {
		return 0, fmt.Errorf("invalid address: %w", err)
	}

	conn, err := c.pool.Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get connection for GetRewardInfo: %w", err)
	}
	defer c.pool.Put(conn)
	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	rewardInfo, err := walletClient.GetRewardInfo(ctx, &api.BytesMessage{Value: addr.Bytes()})
	if err != nil {
		return 0, fmt.Errorf("failed to get reward info: %w", err)
	}
	return rewardInfo.GetNum(), nil
}
