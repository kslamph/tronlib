// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// package lowlevel provides 1:1 wrappers around WalletClient gRPC methods
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Resource management related gRPC calls

// FreezeBalanceV2 freezes balance for resources (v2 - preferred)
func FreezeBalanceV2(cp ConnProvider, ctx context.Context, req *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "freeze balance v2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.FreezeBalanceV2(ctx, req)
	})
}

// UnfreezeBalanceV2 unfreezes balance (v2 - preferred)
func UnfreezeBalanceV2(cp ConnProvider, ctx context.Context, req *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "unfreeze balance v2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnfreezeBalanceV2(ctx, req)
	})
}

// DelegateResource delegates resources to another account
func DelegateResource(cp ConnProvider, ctx context.Context, req *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "delegate resource", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.DelegateResource(ctx, req)
	})
}

// UnDelegateResource undelegates resources from another account
func UnDelegateResource(cp ConnProvider, ctx context.Context, req *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "undelegate resource", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnDelegateResource(ctx, req)
	})
}

// CancelAllUnfreezeV2 cancels all unfreeze operations (v2)
func CancelAllUnfreezeV2(cp ConnProvider, ctx context.Context, req *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "cancel all unfreeze v2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CancelAllUnfreezeV2(ctx, req)
	})
}

// WithdrawExpireUnfreeze withdraws expired unfreeze amount
func WithdrawExpireUnfreeze(cp ConnProvider, ctx context.Context, req *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "withdraw expire unfreeze", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.WithdrawExpireUnfreeze(ctx, req)
	})
}

// GetDelegatedResourceV2 gets delegated resource information (v2 - preferred)
func GetDelegatedResourceV2(cp ConnProvider, ctx context.Context, req *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return Call(cp, ctx, "get delegated resource v2", func(client api.WalletClient, ctx context.Context) (*api.DelegatedResourceList, error) {
		return client.GetDelegatedResourceV2(ctx, req)
	})
}

// GetDelegatedResourceAccountIndexV2 gets delegated resource account index (v2 - preferred)
func GetDelegatedResourceAccountIndexV2(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
	return Call(cp, ctx, "get delegated resource account index v2", func(client api.WalletClient, ctx context.Context) (*core.DelegatedResourceAccountIndex, error) {
		return client.GetDelegatedResourceAccountIndexV2(ctx, req)
	})
}

// GetCanDelegatedMaxSize gets maximum delegatable resource size
func GetCanDelegatedMaxSize(cp ConnProvider, ctx context.Context, req *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	return Call(cp, ctx, "get can delegated max size", func(client api.WalletClient, ctx context.Context) (*api.CanDelegatedMaxSizeResponseMessage, error) {
		return client.GetCanDelegatedMaxSize(ctx, req)
	})
}

// GetAvailableUnfreezeCount gets available unfreeze count
func GetAvailableUnfreezeCount(cp ConnProvider, ctx context.Context, req *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	return Call(cp, ctx, "get available unfreeze count", func(client api.WalletClient, ctx context.Context) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
		return client.GetAvailableUnfreezeCount(ctx, req)
	})
}

// GetCanWithdrawUnfreezeAmount gets withdrawable unfreeze amount
func GetCanWithdrawUnfreezeAmount(cp ConnProvider, ctx context.Context, req *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	return Call(cp, ctx, "get can withdraw unfreeze amount", func(client api.WalletClient, ctx context.Context) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
		return client.GetCanWithdrawUnfreezeAmount(ctx, req)
	})
}
