// Package account provides high-level account management functionality
package account

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// Manager provides high-level account operations
type Manager struct {
	client *client.Client
}

// AccountManager is an explicit alias of Manager for discoverability and future clarity.
type AccountManager = Manager

// NewManager creates a new account manager
func NewManager(client *client.Client) *Manager {
	return &Manager{
		client: client,
	}
}

// TransferOptions contains options for TRX transfers
type TransferOptions struct {
	// Memo is an optional memo for the transfer (currently unused by protocol)
	Memo string
}

// GetAccount retrieves account information by address
func (m *Manager) GetAccount(ctx context.Context, address *types.Address) (*core.Account, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: invalid address: nil", types.ErrInvalidAddress)
	}
	// Prepare gRPC parameters
	req := &core.Account{
		Address: address.Bytes(),
	}

	// Call client package function
	return m.client.GetAccount(ctx, req)
}

// GetAccountNet retrieves account bandwidth information
func (m *Manager) GetAccountNet(ctx context.Context, address *types.Address) (*api.AccountNetMessage, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: invalid address: nil", types.ErrInvalidAddress)
	}

	// Prepare gRPC parameters
	req := &core.Account{
		Address: address.Bytes(),
	}

	// Call client package function
	return m.client.GetAccountNet(ctx, req)
}

// GetAccountResource retrieves account energy information
func (m *Manager) GetAccountResource(ctx context.Context, address *types.Address) (*api.AccountResourceMessage, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: invalid address: nil", types.ErrInvalidAddress)
	}

	// Prepare gRPC parameters
	req := &core.Account{
		Address: address.Bytes(),
	}

	// Call client package function
	return m.client.GetAccountResource(ctx, req)
}

// GetBalance retrieves the TRX balance for an address (convenience method)
func (m *Manager) GetBalance(ctx context.Context, address *types.Address) (int64, error) {
	// Get account info
	account, err := m.GetAccount(ctx, address)
	if err != nil {
		return 0, fmt.Errorf("failed to get account: %w", err)
	}

	return account.GetBalance(), nil
}

// TransferTRX creates an unsigned TRX transfer transaction
func (m *Manager) TransferTRX(ctx context.Context, from *types.Address, to *types.Address, amount int64, opts *TransferOptions) (*api.TransactionExtention, error) {
	// Validate inputs
	if err := m.validateTransferInputs(from, to, amount); err != nil {
		return nil, err
	}

	// Set default options
	if opts == nil {
		opts = &TransferOptions{}
	}

	// Prepare gRPC parameters
	req := &core.TransferContract{
		OwnerAddress: from.Bytes(),
		ToAddress:    to.Bytes(),
		Amount:       amount,
	}

	// Call client package function
	return m.client.CreateTransaction2(ctx, req)
}

// validateTransferInputs validates common transfer parameters
func (m *Manager) validateTransferInputs(from *types.Address, to *types.Address, amount int64) error {
	// Validate addresses
	if from == nil {
		return fmt.Errorf("%w: from address cannot be nil", types.ErrInvalidAddress)
	}
	if to == nil {
		return fmt.Errorf("%w: to address cannot be nil", types.ErrInvalidAddress)
	}

	// Validate amount (must be positive, in SUN units)
	if amount <= 0 {
		return fmt.Errorf("%w: amount must be positive", types.ErrInvalidAmount)
	}

	// Check reasonable upper bound (less than total TRX supply in SUN)
	maxSupply := int64(100_000_000_000 * types.SunPerTRX) // 100B TRX in SUN
	if amount > maxSupply {
		return fmt.Errorf("%w: amount %d exceeds maximum supply", types.ErrInvalidAmount, amount)
	}

	// Check addresses are different
	if from.String() == to.String() {
		return fmt.Errorf("%w: from and to addresses cannot be the same", types.ErrInvalidParameter)
	}

	return nil
}
