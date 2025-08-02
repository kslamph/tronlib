// Package account provides high-level account management functionality
package account

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Manager provides high-level account operations
type Manager struct {
	client *client.Client
}

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
func (m *Manager) GetAccount(ctx context.Context, address string) (*core.Account, error) {
	// Validate address
	addr, err := utils.ValidateAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	// Prepare gRPC parameters
	req := &core.Account{
		Address: addr.Bytes(),
	}

	// Call client package function
	return m.client.GetAccount(ctx, req)
}

// GetAccountNet retrieves account bandwidth information
func (m *Manager) GetAccountNet(ctx context.Context, address string) (*api.AccountNetMessage, error) {
	// Validate address
	addr, err := utils.ValidateAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	// Prepare gRPC parameters
	req := &core.Account{
		Address: addr.Bytes(),
	}

	// Call client package function
	return m.client.GetAccountNet(ctx, req)
}

// GetAccountResource retrieves account energy information
func (m *Manager) GetAccountResource(ctx context.Context, address string) (*api.AccountResourceMessage, error) {
	// Validate address
	addr, err := utils.ValidateAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	// Prepare gRPC parameters
	req := &core.Account{
		Address: addr.Bytes(),
	}

	// Call client package function
	return m.client.GetAccountResource(ctx, req)
}

// GetBalance retrieves the TRX balance for an address (convenience method)
func (m *Manager) GetBalance(ctx context.Context, address string) (int64, error) {
	// Get account info
	account, err := m.GetAccount(ctx, address)
	if err != nil {
		return 0, fmt.Errorf("failed to get account: %w", err)
	}

	return account.GetBalance(), nil
}

// TransferTRX creates an unsigned TRX transfer transaction
func (m *Manager) TransferTRX(ctx context.Context, from string, to string, amount int64, opts *TransferOptions) (*api.TransactionExtention, error) {
	// Validate inputs
	if err := m.validateTransferInputs(from, to, amount); err != nil {
		return nil, err
	}

	// Set default options
	if opts == nil {
		opts = &TransferOptions{}
	}

	// Validate addresses
	fromAddr, err := utils.ValidateAddress(from)
	if err != nil {
		return nil, fmt.Errorf("invalid from address: %w", err)
	}

	toAddr, err := utils.ValidateAddress(to)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %w", err)
	}

	// Prepare gRPC parameters
	req := &core.TransferContract{
		OwnerAddress: fromAddr.Bytes(),
		ToAddress:    toAddr.Bytes(),
		Amount:       amount,
	}

	// Call client package function
	return m.client.CreateTransaction2(ctx, req)
}

// validateTransferInputs validates common transfer parameters
func (m *Manager) validateTransferInputs(from string, to string, amount int64) error {
	// Validate from address
	if from == "" {
		return fmt.Errorf("from address cannot be empty")
	}

	// Validate to address
	if to == "" {
		return fmt.Errorf("to address cannot be empty")
	}

	// Validate amount (must be positive, in SUN units)
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Check reasonable upper bound (less than total TRX supply in SUN)
	maxSupply := int64(100_000_000_000 * types.SunPerTRX) // 100B TRX in SUN
	if amount > maxSupply {
		return fmt.Errorf("amount %d exceeds maximum supply", amount)
	}

	// Check addresses are different
	fromAddr, err := utils.ValidateAddress(from)
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	toAddr, err := utils.ValidateAddress(to)
	if err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}

	if fromAddr.String() == toAddr.String() {
		return fmt.Errorf("from and to addresses cannot be the same")
	}

	return nil
}
