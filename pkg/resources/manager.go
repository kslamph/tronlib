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

// Package resources provides high-level resource management functionality
package resources

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
)

// ResourcesManager provides high-level resource management operations
type ResourcesManager struct {
	conn lowlevel.ConnProvider
}

// NewManager creates a new resource manager
func NewManager(conn lowlevel.ConnProvider) *ResourcesManager {
	return &ResourcesManager{conn: conn}
}

// ResourceType represents the type of resource
type ResourceType int32

const (
	ResourceTypeBandwidth ResourceType = 0
	ResourceTypeEnergy    ResourceType = 1
)

// FreezeBalanceV2 freezes balance for resources (v2)
func (m *ResourcesManager) FreezeBalanceV2(ctx context.Context, ownerAddress *types.Address, frozenBalance int64, resource ResourceType) (*api.TransactionExtention, error) {
	// Validate inputs
	if frozenBalance <= 0 {
		return nil, fmt.Errorf("%w: value must be positive", types.ErrInvalidAmount)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.FreezeBalanceV2Contract{
		OwnerAddress:  ownerAddress.Bytes(),
		FrozenBalance: frozenBalance,
		Resource:      core.ResourceCode(resource),
	}

	return lowlevel.TxCall(m.conn, ctx, "freeze balance v2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.FreezeBalanceV2(ctx, req)
	})
}

// UnfreezeBalanceV2 unfreezes balance (v2)
func (m *ResourcesManager) UnfreezeBalanceV2(ctx context.Context, ownerAddress *types.Address, unfreezeBalance int64, resource ResourceType) (*api.TransactionExtention, error) {
	// Validate inputs
	if unfreezeBalance <= 0 {
		return nil, fmt.Errorf("%w: value must be positive", types.ErrInvalidAmount)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.UnfreezeBalanceV2Contract{
		OwnerAddress:    ownerAddress.Bytes(),
		UnfreezeBalance: unfreezeBalance,
		Resource:        core.ResourceCode(resource),
	}

	return lowlevel.TxCall(m.conn, ctx, "unfreeze balance v2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UnfreezeBalanceV2(ctx, req)
	})
}

// DelegateResource delegates resources to another account
func (m *ResourcesManager) DelegateResource(ctx context.Context, ownerAddress, receiverAddress *types.Address, balance int64, resource ResourceType, lock bool) (*api.TransactionExtention, error) {
	// Validate inputs
	if balance <= 0 {
		return nil, fmt.Errorf("%w: value must be positive", types.ErrInvalidAmount)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}
	if receiverAddress == nil {
		return nil, fmt.Errorf("%w: invalid receiver address: nil", types.ErrInvalidAddress)
	}

	if ownerAddress.String() == receiverAddress.String() {
		return nil, fmt.Errorf("%w: owner and receiver addresses cannot be the same", types.ErrInvalidParameter)
	}

	req := &core.DelegateResourceContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ReceiverAddress: receiverAddress.Bytes(),
		Balance:         balance,
		Resource:        core.ResourceCode(resource),
		Lock:            lock,
	}

	return lowlevel.TxCall(m.conn, ctx, "delegate resource", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.DelegateResource(ctx, req)
	})
}

// UnDelegateResource undelegates resources from another account
func (m *ResourcesManager) UnDelegateResource(ctx context.Context, ownerAddress, receiverAddress *types.Address, balance int64, resource ResourceType) (*api.TransactionExtention, error) {
	// Validate inputs
	if balance <= 0 {
		return nil, fmt.Errorf("%w: value must be positive", types.ErrInvalidAmount)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}
	if receiverAddress == nil {
		return nil, fmt.Errorf("%w: invalid receiver address: nil", types.ErrInvalidAddress)
	}

	req := &core.UnDelegateResourceContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ReceiverAddress: receiverAddress.Bytes(),
		Balance:         balance,
		Resource:        core.ResourceCode(resource),
	}

	return lowlevel.TxCall(m.conn, ctx, "undelegate resource", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UnDelegateResource(ctx, req)
	})
}

// CancelAllUnfreezeV2 cancels all unfreeze operations (v2)
func (m *ResourcesManager) CancelAllUnfreezeV2(ctx context.Context, ownerAddress *types.Address) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.CancelAllUnfreezeV2Contract{OwnerAddress: ownerAddress.Bytes()}
	return lowlevel.TxCall(m.conn, ctx, "cancel all unfreeze v2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.CancelAllUnfreezeV2(ctx, req)
	})
}

// WithdrawExpireUnfreeze withdraws expired unfreeze amount
func (m *ResourcesManager) WithdrawExpireUnfreeze(ctx context.Context, ownerAddress *types.Address) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.WithdrawExpireUnfreezeContract{OwnerAddress: ownerAddress.Bytes()}
	return lowlevel.TxCall(m.conn, ctx, "withdraw expire unfreeze", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.WithdrawExpireUnfreeze(ctx, req)
	})
}

// GetDelegatedResourceV2 gets delegated resource information (v2)
func (m *ResourcesManager) GetDelegatedResourceV2(ctx context.Context, fromAddress, toAddress *types.Address) (*api.DelegatedResourceList, error) {
	if fromAddress == nil {
		return nil, fmt.Errorf("%w: invalid from address: nil", types.ErrInvalidAddress)
	}
	if toAddress == nil {
		return nil, fmt.Errorf("%w: invalid to address: nil", types.ErrInvalidAddress)
	}

	req := &api.DelegatedResourceMessage{FromAddress: fromAddress.Bytes(), ToAddress: toAddress.Bytes()}
	return lowlevel.Call(m.conn, ctx, "get delegated resource v2", func(cl api.WalletClient, ctx context.Context) (*api.DelegatedResourceList, error) {
		return cl.GetDelegatedResourceV2(ctx, req)
	})
}

// GetDelegatedResourceAccountIndexV2 gets delegated resource account index (v2)
func (m *ResourcesManager) GetDelegatedResourceAccountIndexV2(ctx context.Context, address *types.Address) (*core.DelegatedResourceAccountIndex, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: invalid address: nil", types.ErrInvalidAddress)
	}

	req := &api.BytesMessage{Value: address.Bytes()}
	return lowlevel.Call(m.conn, ctx, "get delegated resource account index v2", func(cl api.WalletClient, ctx context.Context) (*core.DelegatedResourceAccountIndex, error) {
		return cl.GetDelegatedResourceAccountIndexV2(ctx, req)
	})
}

// GetCanDelegatedMaxSize gets maximum delegatable resource size
func (m *ResourcesManager) GetCanDelegatedMaxSize(ctx context.Context, ownerAddress *types.Address, delegateType int32) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &api.CanDelegatedMaxSizeRequestMessage{OwnerAddress: ownerAddress.Bytes(), Type: delegateType}
	return lowlevel.Call(m.conn, ctx, "get can delegated max size", func(cl api.WalletClient, ctx context.Context) (*api.CanDelegatedMaxSizeResponseMessage, error) {
		return cl.GetCanDelegatedMaxSize(ctx, req)
	})
}

// GetAvailableUnfreezeCount gets available unfreeze count
func (m *ResourcesManager) GetAvailableUnfreezeCount(ctx context.Context, ownerAddress *types.Address) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &api.GetAvailableUnfreezeCountRequestMessage{OwnerAddress: ownerAddress.Bytes()}
	return lowlevel.Call(m.conn, ctx, "get available unfreeze count", func(cl api.WalletClient, ctx context.Context) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
		return cl.GetAvailableUnfreezeCount(ctx, req)
	})
}

// GetCanWithdrawUnfreezeAmount gets withdrawable unfreeze amount
func (m *ResourcesManager) GetCanWithdrawUnfreezeAmount(ctx context.Context, ownerAddress *types.Address, timestamp int64) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &api.CanWithdrawUnfreezeAmountRequestMessage{OwnerAddress: ownerAddress.Bytes(), Timestamp: timestamp}
	return lowlevel.Call(m.conn, ctx,
		"get can withdraw unfreeze amount",
		func(cl api.WalletClient, ctx context.Context) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
			return cl.GetCanWithdrawUnfreezeAmount(ctx, req)
		},
	)
}
