package trc20

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

// Transfer transfers tokens to a specified address
func (t *TRC20Contract) Transfer(ctx context.Context, from, to string, amount decimal.Decimal) (*api.TransactionExtention, error) {

	decimals, err := t.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert decimal amount to big.Int with proper decimals
	rawAmount := amount.Shift(int32(decimals)).BigInt()

	data, err := t.EncodeInput("transfer", to, rawAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer call: %v", err)
	}
	ownerAddr, err := types.NewAddress(from)
	if err != nil {
		return nil, fmt.Errorf("failed to parse owner address: %v", err)
	}
	return t.client.TriggerSmartContract(ctx, ownerAddr, t.Contract, data, 0)
}

// TransferFrom transfers tokens from `from` to `to`
// `spender` approved by `from` is required
// transaction is expected to be signed by `spender`
func (t *TRC20Contract) TransferFrom(ctx context.Context, spender, from, to string, amount decimal.Decimal) (*api.TransactionExtention, error) {

	decimals, err := t.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	rawAmount := amount.Shift(int32(decimals)).BigInt()

	ownerAddr, err := types.NewAddress(spender)
	if err != nil {
		return nil, fmt.Errorf("failed to parse owner address: %v", err)
	}

	data, err := t.EncodeInput("transferFrom", from, to, rawAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer call: %v", err)
	}

	return t.client.TriggerSmartContract(ctx, ownerAddr, t.Contract, data, 0)
}

// Approve allows spender to withdraw from your account multiple times up to the amount
func (t *TRC20Contract) Approve(ctx context.Context, from, spender string, amount decimal.Decimal) (*api.TransactionExtention, error) {
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	// Convert decimal amount to big.Int with proper decimals
	rawAmount := amount.Shift(int32(decimals)).BigInt()

	ownerAddr, err := types.NewAddress(from)
	if err != nil {
		return nil, fmt.Errorf("failed to parse owner address: %v", err)
	}
	data, err := t.EncodeInput("approve", spender, rawAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to create approve call: %v", err)
	}
	return t.client.TriggerSmartContract(ctx, ownerAddr, t.Contract, data, 0)
}
