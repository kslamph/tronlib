package smartcontract

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pkg/transaction"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

// Transfer transfers tokens to a specified address
func (t *TRC20Contract) Transfer(ctx context.Context, from, to string, amount decimal.Decimal) *transaction.Transaction {
	tx := transaction.NewTransaction(t.client)

	decimals, err := t.Decimals(ctx)
	if err != nil {
		tx.SetError(fmt.Errorf("failed to get decimals: %v", err))
		return tx
	}

	// Convert decimal amount to big.Int with proper decimals
	rawAmount := amount.Shift(int32(decimals)).BigInt()

	data, err := t.EncodeInput("transfer", to, rawAmount)
	if err != nil {
		tx.SetError(fmt.Errorf("failed to create transfer call: %v", err))
		return tx
	}
	ownerAddr, err := types.NewAddress(from)
	if err != nil {
		tx.SetError(fmt.Errorf("failed to parse owner address: %v", err))
		return tx
	}
	tx.SetOwner(ownerAddr)
	tx.TriggerSmartContract(ctx, t.Contract, data, 0)
	tx.SetDefaultOptions()
	return tx
}

// TransferFrom transfers tokens from one address to another
// spender is the address of the from account authorized to transfer
func (t *TRC20Contract) TransferFrom(ctx context.Context, spender, from, to string, amount decimal.Decimal) *transaction.Transaction {
	tx := transaction.NewTransaction(t.client)

	decimals, err := t.Decimals(ctx)
	if err != nil {
		tx.SetError(fmt.Errorf("failed to get decimals: %v", err))
		return tx
	}

	rawAmount := amount.Shift(int32(decimals)).BigInt()

	ownerAddr, err := types.NewAddress(spender)
	if err != nil {
		tx.SetError(fmt.Errorf("failed to parse owner address: %v", err))
		return tx
	}

	data, err := t.EncodeInput("transferFrom", from, to, rawAmount)
	if err != nil {
		tx.SetError(fmt.Errorf("failed to create transfer call: %v", err))
		return tx
	}

	return tx.SetOwner(ownerAddr).TriggerSmartContract(ctx, t.Contract, data, 0).SetDefaultOptions()
}

// Approve allows spender to withdraw from your account multiple times up to the amount
func (t *TRC20Contract) Approve(ctx context.Context, from, spender string, amount decimal.Decimal) *transaction.Transaction {
	tx := transaction.NewTransaction(t.client)
	decimals, err := t.Decimals(ctx)
	if err != nil {
		tx.SetError(fmt.Errorf("failed to get decimals: %v", err))
		return tx
	}

	// Convert decimal amount to big.Int with proper decimals
	rawAmount := amount.Shift(int32(decimals)).BigInt()

	ownerAddr, err := types.NewAddress(from)
	if err != nil {
		tx.SetError(fmt.Errorf("failed to parse owner address: %v", err))
		return tx
	}
	data, err := t.EncodeInput("approve", spender, rawAmount)
	if err != nil {
		tx.SetError(fmt.Errorf("failed to create approve call: %v", err))
		return tx
	}
	return tx.SetOwner(ownerAddr).TriggerSmartContract(ctx, t.Contract, data, 0).SetDefaultOptions()
}
