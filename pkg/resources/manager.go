// Package resources provides high-level resource management functionality
package resources

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Manager provides high-level resource management operations
type Manager struct {
	client *client.Client
}

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
func (m *Manager) FreezeBalanceV2(ctx context.Context, ownerAddress string, frozenBalance int64, resource ResourceType) (*api.TransactionExtention, error) {
	// Validate inputs
	if frozenBalance <= 0 {
		return nil, fmt.Errorf("frozen balance must be positive")
	}

	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.FreezeBalanceV2Contract{
		OwnerAddress:  addr.Bytes(),
		FrozenBalance: frozenBalance,
		Resource:      core.ResourceCode(resource),
	}

	return m.client.FreezeBalanceV2(ctx, req)
}

// UnfreezeBalanceV2 unfreezes balance (v2)
func (m *Manager) UnfreezeBalanceV2(ctx context.Context, ownerAddress string, unfreezeBalance int64, resource ResourceType) (*api.TransactionExtention, error) {
	// Validate inputs
	if unfreezeBalance <= 0 {
		return nil, fmt.Errorf("unfreeze balance must be positive")
	}

	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.UnfreezeBalanceV2Contract{
		OwnerAddress:    addr.Bytes(),
		UnfreezeBalance: unfreezeBalance,
		Resource:        core.ResourceCode(resource),
	}

	return m.client.UnfreezeBalanceV2(ctx, req)
}

// DelegateResource delegates resources to another account
func (m *Manager) DelegateResource(ctx context.Context, ownerAddress string, receiverAddress string, balance int64, resource ResourceType, lock bool) (*api.TransactionExtention, error) {
	// Validate inputs
	if balance <= 0 {
		return nil, fmt.Errorf("balance must be positive")
	}

	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	receiverAddr, err := utils.ValidateAddress(receiverAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver address: %w", err)
	}

	if ownerAddr.String() == receiverAddr.String() {
		return nil, fmt.Errorf("owner and receiver addresses cannot be the same")
	}

	req := &core.DelegateResourceContract{
		OwnerAddress:    ownerAddr.Bytes(),
		ReceiverAddress: receiverAddr.Bytes(),
		Balance:         balance,
		Resource:        core.ResourceCode(resource),
		Lock:            lock,
	}

	return m.client.DelegateResource(ctx, req)
}

// UnDelegateResource undelegates resources from another account
func (m *Manager) UnDelegateResource(ctx context.Context, ownerAddress string, receiverAddress string, balance int64, resource ResourceType) (*api.TransactionExtention, error) {
	// Validate inputs
	if balance <= 0 {
		return nil, fmt.Errorf("balance must be positive")
	}

	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	receiverAddr, err := utils.ValidateAddress(receiverAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver address: %w", err)
	}

	req := &core.UnDelegateResourceContract{
		OwnerAddress:    ownerAddr.Bytes(),
		ReceiverAddress: receiverAddr.Bytes(),
		Balance:         balance,
		Resource:        core.ResourceCode(resource),
	}

	return m.client.UnDelegateResource(ctx, req)
}

// CancelAllUnfreezeV2 cancels all unfreeze operations (v2)
func (m *Manager) CancelAllUnfreezeV2(ctx context.Context, ownerAddress string) (*api.TransactionExtention, error) {
	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.CancelAllUnfreezeV2Contract{
		OwnerAddress: addr.Bytes(),
	}

	return m.client.CancelAllUnfreezeV2(ctx, req)
}

// WithdrawExpireUnfreeze withdraws expired unfreeze amount
func (m *Manager) WithdrawExpireUnfreeze(ctx context.Context, ownerAddress string) (*api.TransactionExtention, error) {
	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.WithdrawExpireUnfreezeContract{
		OwnerAddress: addr.Bytes(),
	}

	return m.client.WithdrawExpireUnfreeze(ctx, req)
}

// GetDelegatedResourceV2 gets delegated resource information (v2)
func (m *Manager) GetDelegatedResourceV2(ctx context.Context, fromAddress string, toAddress string) (*api.DelegatedResourceList, error) {
	fromAddr, err := utils.ValidateAddress(fromAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid from address: %w", err)
	}

	toAddr, err := utils.ValidateAddress(toAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %w", err)
	}

	req := &api.DelegatedResourceMessage{
		FromAddress: fromAddr.Bytes(),
		ToAddress:   toAddr.Bytes(),
	}

	return m.client.GetDelegatedResourceV2(ctx, req)
}

// GetDelegatedResourceAccountIndexV2 gets delegated resource account index (v2)
func (m *Manager) GetDelegatedResourceAccountIndexV2(ctx context.Context, address string) (*core.DelegatedResourceAccountIndex, error) {
	addr, err := utils.ValidateAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	req := &api.BytesMessage{
		Value: addr.Bytes(),
	}

	return m.client.GetDelegatedResourceAccountIndexV2(ctx, req)
}

// GetCanDelegatedMaxSize gets maximum delegatable resource size
func (m *Manager) GetCanDelegatedMaxSize(ctx context.Context, ownerAddress string, delegateType int32) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &api.CanDelegatedMaxSizeRequestMessage{
		OwnerAddress: addr.Bytes(),
		Type:         delegateType,
	}

	return m.client.GetCanDelegatedMaxSize(ctx, req)
}

// GetAvailableUnfreezeCount gets available unfreeze count
func (m *Manager) GetAvailableUnfreezeCount(ctx context.Context, ownerAddress string) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &api.GetAvailableUnfreezeCountRequestMessage{
		OwnerAddress: addr.Bytes(),
	}

	return m.client.GetAvailableUnfreezeCount(ctx, req)
}

// GetCanWithdrawUnfreezeAmount gets withdrawable unfreeze amount
func (m *Manager) GetCanWithdrawUnfreezeAmount(ctx context.Context, ownerAddress string, timestamp int64) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &api.CanWithdrawUnfreezeAmountRequestMessage{
		OwnerAddress: addr.Bytes(),
		Timestamp:    timestamp,
	}

	return m.client.GetCanWithdrawUnfreezeAmount(ctx, req)
}
