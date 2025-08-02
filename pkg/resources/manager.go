// Package resources provides high-level resource management functionality
package resources

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// Manager provides high-level resource management operations
type Manager struct {
	client *client.Client
}

// ResourceManager is an explicit alias of Manager for discoverability and future clarity.
type ResourceManager = Manager

// NewManager creates a new resource manager
func NewManager(client *client.Client) *Manager {
	return &Manager{
		client: client,
	}
}

// ResourceType represents the type of resource
type ResourceType int32

const (
	ResourceTypeBandwidth ResourceType = 0
	ResourceTypeEnergy    ResourceType = 1
)

// FreezeBalanceV2 freezes balance for resources (v2)
func (m *Manager) FreezeBalanceV2(ctx context.Context, ownerAddress *types.Address, frozenBalance int64, resource ResourceType) (*api.TransactionExtention, error) {
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

	return m.client.FreezeBalanceV2(ctx, req)
}

// UnfreezeBalanceV2 unfreezes balance (v2)
func (m *Manager) UnfreezeBalanceV2(ctx context.Context, ownerAddress *types.Address, unfreezeBalance int64, resource ResourceType) (*api.TransactionExtention, error) {
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

	return m.client.UnfreezeBalanceV2(ctx, req)
}

// DelegateResource delegates resources to another account
func (m *Manager) DelegateResource(ctx context.Context, ownerAddress, receiverAddress *types.Address, balance int64, resource ResourceType, lock bool) (*api.TransactionExtention, error) {
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

	return m.client.DelegateResource(ctx, req)
}

// UnDelegateResource undelegates resources from another account
func (m *Manager) UnDelegateResource(ctx context.Context, ownerAddress, receiverAddress *types.Address, balance int64, resource ResourceType) (*api.TransactionExtention, error) {
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

	return m.client.UnDelegateResource(ctx, req)
}

// CancelAllUnfreezeV2 cancels all unfreeze operations (v2)
func (m *Manager) CancelAllUnfreezeV2(ctx context.Context, ownerAddress *types.Address) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.CancelAllUnfreezeV2Contract{
		OwnerAddress: ownerAddress.Bytes(),
	}

	return m.client.CancelAllUnfreezeV2(ctx, req)
}

// WithdrawExpireUnfreeze withdraws expired unfreeze amount
func (m *Manager) WithdrawExpireUnfreeze(ctx context.Context, ownerAddress *types.Address) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.WithdrawExpireUnfreezeContract{
		OwnerAddress: ownerAddress.Bytes(),
	}

	return m.client.WithdrawExpireUnfreeze(ctx, req)
}

// GetDelegatedResourceV2 gets delegated resource information (v2)
func (m *Manager) GetDelegatedResourceV2(ctx context.Context, fromAddress, toAddress *types.Address) (*api.DelegatedResourceList, error) {
	if fromAddress == nil {
		return nil, fmt.Errorf("%w: invalid from address: nil", types.ErrInvalidAddress)
	}
	if toAddress == nil {
		return nil, fmt.Errorf("%w: invalid to address: nil", types.ErrInvalidAddress)
	}

	req := &api.DelegatedResourceMessage{
		FromAddress: fromAddress.Bytes(),
		ToAddress:   toAddress.Bytes(),
	}

	return m.client.GetDelegatedResourceV2(ctx, req)
}

// GetDelegatedResourceAccountIndexV2 gets delegated resource account index (v2)
func (m *Manager) GetDelegatedResourceAccountIndexV2(ctx context.Context, address *types.Address) (*core.DelegatedResourceAccountIndex, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: invalid address: nil", types.ErrInvalidAddress)
	}

	req := &api.BytesMessage{
		Value: address.Bytes(),
	}

	return m.client.GetDelegatedResourceAccountIndexV2(ctx, req)
}

// GetCanDelegatedMaxSize gets maximum delegatable resource size
func (m *Manager) GetCanDelegatedMaxSize(ctx context.Context, ownerAddress *types.Address, delegateType int32) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &api.CanDelegatedMaxSizeRequestMessage{
		OwnerAddress: ownerAddress.Bytes(),
		Type:         delegateType,
	}

	return m.client.GetCanDelegatedMaxSize(ctx, req)
}

// GetAvailableUnfreezeCount gets available unfreeze count
func (m *Manager) GetAvailableUnfreezeCount(ctx context.Context, ownerAddress *types.Address) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &api.GetAvailableUnfreezeCountRequestMessage{
		OwnerAddress: ownerAddress.Bytes(),
	}

	return m.client.GetAvailableUnfreezeCount(ctx, req)
}

// GetCanWithdrawUnfreezeAmount gets withdrawable unfreeze amount
func (m *Manager) GetCanWithdrawUnfreezeAmount(ctx context.Context, ownerAddress *types.Address, timestamp int64) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &api.CanWithdrawUnfreezeAmountRequestMessage{
		OwnerAddress: ownerAddress.Bytes(),
		Timestamp:    timestamp,
	}

	return m.client.GetCanWithdrawUnfreezeAmount(ctx, req)
}
