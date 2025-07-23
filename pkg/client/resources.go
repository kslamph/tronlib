package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

// CreateFreezeTransaction creates a freeze balance transaction
func (c *Client) CreateFreezeTransaction(ctx context.Context, ownerAddress *types.Address, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "freeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		if ownerAddress == nil {
			return nil, fmt.Errorf("owner address is nil")
		}
		return client.FreezeBalanceV2(ctx, &core.FreezeBalanceV2Contract{
			OwnerAddress:  ownerAddress.Bytes(),
			FrozenBalance: amount,
			Resource:      resource,
		})
	})
}

// CreateUnfreezeTransaction creates an unfreeze balance transaction
func (c *Client) CreateUnfreezeTransaction(ctx context.Context, ownerAddress *types.Address, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "unfreeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		if ownerAddress == nil {
			return nil, fmt.Errorf("owner address is nil")
		}
		return client.UnfreezeBalanceV2(ctx, &core.UnfreezeBalanceV2Contract{
			OwnerAddress:    ownerAddress.Bytes(),
			UnfreezeBalance: amount,
			Resource:        resource,
		})
	})
}

// CreateDelegateResourceTransaction creates a delegate resource transaction
func (c *Client) CreateDelegateResourceTransaction(ctx context.Context, ownerAddress, delegateTo *types.Address, amount int64, resource core.ResourceCode, lock bool, blocksToLock ...int64) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "delegate resource", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {

		if lock && len(blocksToLock) > 0 {
			lockPeriod := blocksToLock[0]
			return client.DelegateResource(ctx, &core.DelegateResourceContract{
				OwnerAddress:    ownerAddress.Bytes(),
				ReceiverAddress: delegateTo.Bytes(),
				Balance:         amount,
				Resource:        resource,
				Lock:            lock,
				LockPeriod:      lockPeriod,
			})
		}
		return client.DelegateResource(ctx, &core.DelegateResourceContract{
			OwnerAddress:    ownerAddress.Bytes(),
			ReceiverAddress: delegateTo.Bytes(),
			Balance:         amount,
			Resource:        resource,
		})
	})
}

// CreateUndelegateResourceTransaction creates an undelegate resource transaction
func (c *Client) CreateUndelegateResourceTransaction(ctx context.Context, ownerAddress, reclaimFrom *types.Address, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "undelegate resource", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnDelegateResource(ctx, &core.UnDelegateResourceContract{
			OwnerAddress:    ownerAddress.Bytes(),
			ReceiverAddress: reclaimFrom.Bytes(),
			Balance:         amount,
			Resource:        resource,
		})
	})
}

// CreateWithdrawExpireUnfreezeTransaction creates a withdraw from expired unfreeze transaction
func (c *Client) CreateWithdrawExpireUnfreezeTransaction(ctx context.Context, ownerAddress *types.Address) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "withdraw expire unfreeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.WithdrawExpireUnfreeze(ctx, &core.WithdrawExpireUnfreezeContract{
			OwnerAddress: ownerAddress.Bytes(),
		})
	})
}

// GetDelegatedResourceV2 retrieves delegated resource info
func (c *Client) GetDelegatedResourceV2(ctx context.Context, fromAddr, toAddr *types.Address) (*api.DelegatedResourceList, error) {
	if fromAddr == nil || toAddr == nil {
		return nil, fmt.Errorf("GetDelegatedResourceV2 failed: addresses cannot be nil")
	}
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for GetDelegatedResourceV2: %w", err)
	}
	defer c.pool.Put(conn)
	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	return walletClient.GetDelegatedResourceV2(ctx, &api.DelegatedResourceMessage{
		FromAddress: fromAddr.Bytes(),
		ToAddress:   toAddr.Bytes(),
	})
}

// GetDelegatedResourceAccountIndexV2 retrieves delegated resource account index
func (c *Client) GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	if len(address) == 0 {
		return nil, fmt.Errorf("GetDelegatedResourceAccountIndexV2 failed: address is empty")
	}
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for GetDelegatedResourceAccountIndexV2: %w", err)
	}
	defer c.pool.Put(conn)
	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	return walletClient.GetDelegatedResourceAccountIndexV2(ctx, &api.BytesMessage{Value: address})
}

// GetCanDelegatedMaxSize retrieves the max size that can be delegated
func (c *Client) GetCanDelegatedMaxSize(ctx context.Context, ownerAddr *types.Address, typeCode int32) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	if ownerAddr == nil {
		return nil, fmt.Errorf("GetCanDelegatedMaxSize failed: owner address is nil")
	}
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for GetCanDelegatedMaxSize: %w", err)
	}
	defer c.pool.Put(conn)
	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	return walletClient.GetCanDelegatedMaxSize(ctx, &api.CanDelegatedMaxSizeRequestMessage{
		OwnerAddress: ownerAddr.Bytes(),
		Type:         typeCode,
	})
}

// GetCanWithdrawUnfreezeAmount retrieves the amount that can be withdrawn/unfrozen
func (c *Client) GetCanWithdrawUnfreezeAmount(ctx context.Context, ownerAddr *types.Address) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	if ownerAddr == nil {
		return nil, fmt.Errorf("GetCanWithdrawUnfreezeAmount failed: owner address is nil")
	}
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for GetCanWithdrawUnfreezeAmount: %w", err)
	}
	defer c.pool.Put(conn)
	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	return walletClient.GetCanWithdrawUnfreezeAmount(ctx, &api.CanWithdrawUnfreezeAmountRequestMessage{
		OwnerAddress: ownerAddr.Bytes(),
	})
}
