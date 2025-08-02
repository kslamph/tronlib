// package client provides 1:1 wrappers around WalletClient gRPC methods
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
)

// Resource management related gRPC calls

// FreezeBalanceV2 freezes balance for resources (v2 - preferred)
func FreezeBalanceV2(c *client.Client, ctx context.Context, req *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "freeze balance v2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.FreezeBalanceV2(ctx, req)
	})
}

// UnfreezeBalanceV2 unfreezes balance (v2 - preferred)
func UnfreezeBalanceV2(c *client.Client, ctx context.Context, req *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "unfreeze balance v2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnfreezeBalanceV2(ctx, req)
	})
}

// DelegateResource delegates resources to another account
func DelegateResource(c *client.Client, ctx context.Context, req *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "delegate resource", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.DelegateResource(ctx, req)
	})
}

// UnDelegateResource undelegates resources from another account
func UnDelegateResource(c *client.Client, ctx context.Context, req *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "undelegate resource", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnDelegateResource(ctx, req)
	})
}

// CancelAllUnfreezeV2 cancels all unfreeze operations (v2)
func CancelAllUnfreezeV2(c *client.Client, ctx context.Context, req *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "cancel all unfreeze v2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CancelAllUnfreezeV2(ctx, req)
	})
}

// WithdrawExpireUnfreeze withdraws expired unfreeze amount
func WithdrawExpireUnfreeze(c *client.Client, ctx context.Context, req *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "withdraw expire unfreeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.WithdrawExpireUnfreeze(ctx, req)
	})
}

// GetDelegatedResourceV2 gets delegated resource information (v2 - preferred)
func GetDelegatedResourceV2(c *client.Client, ctx context.Context, req *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return grpcGenericCallWrapper(c, ctx, "get delegated resource v2", func(client api.WalletClient, ctx context.Context) (*api.DelegatedResourceList, error) {
		return client.GetDelegatedResourceV2(ctx, req)
	})
}

// GetDelegatedResourceAccountIndexV2 gets delegated resource account index (v2 - preferred)
func GetDelegatedResourceAccountIndexV2(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
	return grpcGenericCallWrapper(c, ctx, "get delegated resource account index v2", func(client api.WalletClient, ctx context.Context) (*core.DelegatedResourceAccountIndex, error) {
		return client.GetDelegatedResourceAccountIndexV2(ctx, req)
	})
}

// GetCanDelegatedMaxSize gets maximum delegatable resource size
func GetCanDelegatedMaxSize(c *client.Client, ctx context.Context, req *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get can delegated max size", func(client api.WalletClient, ctx context.Context) (*api.CanDelegatedMaxSizeResponseMessage, error) {
		return client.GetCanDelegatedMaxSize(ctx, req)
	})
}

// GetAvailableUnfreezeCount gets available unfreeze count
func GetAvailableUnfreezeCount(c *client.Client, ctx context.Context, req *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get available unfreeze count", func(client api.WalletClient, ctx context.Context) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
		return client.GetAvailableUnfreezeCount(ctx, req)
	})
}

// GetCanWithdrawUnfreezeAmount gets withdrawable unfreeze amount
func GetCanWithdrawUnfreezeAmount(c *client.Client, ctx context.Context, req *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get can withdraw unfreeze amount", func(client api.WalletClient, ctx context.Context) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
		return client.GetCanWithdrawUnfreezeAmount(ctx, req)
	})
}
