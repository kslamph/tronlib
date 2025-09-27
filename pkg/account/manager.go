package account

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
)

// ConnProvider is the minimal dependency required to perform low-level RPCs.
// It is satisfied by *client.Client and any other connection pool provider.
type ConnProvider = lowlevel.ConnProvider

// AccountManager provides high-level account operations.
//
// The AccountManager allows you to query account information, retrieve balances,
// and create TRX transfer transactions. It works with a connection provider
// (typically a *client.Client) to communicate with the TRON network.
type AccountManager struct {
	conn lowlevel.ConnProvider
}

// NewManager creates a new account manager.
//
// The account manager requires a connection provider (typically a *client.Client)
// to communicate with the TRON network.
//
// Example:
//
//	cli, err := client.NewClient("grpc://grpc.trongrid.io:50051")
//	if err != nil {
//	    // handle error
//	}
//	defer cli.Close()
//
//	accountMgr := account.NewManager(cli)
func NewManager(conn lowlevel.ConnProvider) *AccountManager {
	return &AccountManager{conn: conn}
}

// GetAccount retrieves account information by address.
//
// This method fetches detailed account information from the TRON network,
// including balance, resources, and other account properties.
//
// Returns an error if the address is invalid or if the account doesn't exist.
func (m *AccountManager) GetAccount(ctx context.Context, address *types.Address) (*core.Account, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: address cannot be nil", types.ErrInvalidAddress)
	}
	if address.String() == "" {
		return nil, fmt.Errorf("%w: address cannot be empty", types.ErrInvalidAddress)
	}
	// Prepare gRPC parameters
	req := &core.Account{Address: address.Bytes()}

	// Call lowlevel
	return lowlevel.Call(m.conn, ctx, "get account", func(cl api.WalletClient, ctx context.Context) (*core.Account, error) {
		return cl.GetAccount(ctx, req)
	})
}

// GetAccountNet retrieves account bandwidth information
func (m *AccountManager) GetAccountNet(ctx context.Context, address *types.Address) (*api.AccountNetMessage, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: address cannot be nil", types.ErrInvalidAddress)
	}

	// Prepare gRPC parameters
	req := &core.Account{Address: address.Bytes()}

	// Call lowlevel
	return lowlevel.Call(m.conn, ctx, "get account net", func(cl api.WalletClient, ctx context.Context) (*api.AccountNetMessage, error) {
		return cl.GetAccountNet(ctx, req)
	})
}

// GetAccountResource retrieves account energy information
func (m *AccountManager) GetAccountResource(ctx context.Context, address *types.Address) (*api.AccountResourceMessage, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: address cannot be nil", types.ErrInvalidAddress)
	}

	// Prepare gRPC parameters
	req := &core.Account{Address: address.Bytes()}

	// Call lowlevel
	return lowlevel.Call(m.conn, ctx, "get account resource", func(cl api.WalletClient, ctx context.Context) (*api.AccountResourceMessage, error) {
		return cl.GetAccountResource(ctx, req)
	})
}

// GetBalance retrieves the TRX balance for an address (convenience method).
//
// This method returns the TRX balance in SUN (1 TRX = 1,000,000 SUN).
// It's a convenience method that fetches the full account information
// and returns just the balance.
//
// Example:
//
//		balance, err := accountMgr.GetBalance(ctx, address)
//		if err != nil {
//		    // handle error
//		}
//	    // Convert SUN to TRX using utils package for proper formatting
//		trxBalance, err := utils.HumanReadableBalance(balance, 6) // 6 decimal places for TRX
//		if err != nil {
//		    // handle error
//		    fmt.Printf("Balance: %d SUN\n", balance)
//		} else {
//		    fmt.Printf("Balance: %s TRX\n", trxBalance)
//		}
func (m *AccountManager) GetBalance(ctx context.Context, address *types.Address) (int64, error) {
	// Get account info
	account, err := m.GetAccount(ctx, address)
	if err != nil {
		return 0, fmt.Errorf("failed to get account balance: %w", err)
	}

	return account.GetBalance(), nil
}

// TransferTRX creates an unsigned TRX transfer transaction.
//
// This method creates a TRX transfer transaction from one address to another.
// The transaction is not signed or broadcast - use client.SignAndBroadcast
// to complete the transfer.
//
// The amount should be specified in SUN (1 TRX = 1,000,000 SUN).
//
// Example:
//
//	txExt, err := accountMgr.TransferTRX(ctx, from, to, 1_000_000) // 1 TRX
//	if err != nil {
//	    // handle error
//	}
//
//	// Sign and broadcast the transaction
//	opts := client.DefaultBroadcastOptions()
//	result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
func (m *AccountManager) TransferTRX(ctx context.Context, from *types.Address, to *types.Address, amount int64) (*api.TransactionExtention, error) {
	// Validate inputs
	if err := m.validateTransferInputs(from, to, amount); err != nil {
		return nil, err
	}

	// Prepare gRPC parameters
	req := &core.TransferContract{
		OwnerAddress: from.Bytes(),
		ToAddress:    to.Bytes(),
		Amount:       amount,
	}

	// Call lowlevel
	return lowlevel.TxCall(m.conn, ctx, "create transaction2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.CreateTransaction2(ctx, req)
	})
}

// validateTransferInputs validates common transfer parameters
func (m *AccountManager) validateTransferInputs(from *types.Address, to *types.Address, amount int64) error {
	// Validate addresses
	if from == nil {
		return fmt.Errorf("%w: from address cannot be nil", types.ErrInvalidAddress)
	}
	if to == nil {
		return fmt.Errorf("%w: to address cannot be nil", types.ErrInvalidAddress)
	}

	// Validate amount (must be positive, in SUN units)
	if amount <= 0 {
		return fmt.Errorf("%w: amount must be positive, got %d SUN", types.ErrInvalidAmount, amount)
	}

	// Check addresses are different
	if from.String() == to.String() {
		return fmt.Errorf("%w: from and to addresses cannot be the same: %s", types.ErrInvalidParameter, from.String())
	}

	return nil
}
