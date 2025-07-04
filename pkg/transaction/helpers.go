package transaction

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

// transactionCall represents a function that creates a transaction
type transactionCall func(ctx context.Context) (*api.TransactionExtention, error)

// executeTransaction is a helper that handles the common transaction creation pattern
func (tx *Transaction) executeTransaction(ctx context.Context, operation string, call transactionCall) *Transaction {
	if tx.receipt.Err != nil {
		return tx
	}

	txExt, err := call(ctx)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to create %s transaction: %v", operation, err)
		return tx
	}

	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}

// createResourceTransaction is a helper for resource-related transactions
func (tx *Transaction) createResourceTransaction(ctx context.Context, operation string, amount int64, resource core.ResourceCode, call func(context.Context, []byte, int64, core.ResourceCode) (*api.TransactionExtention, error)) *Transaction {
	return tx.executeTransaction(ctx, operation, func(ctx context.Context) (*api.TransactionExtention, error) {
		return call(ctx, tx.owner.Bytes(), amount, resource)
	})
}

// createDelegateTransaction is a helper for delegation transactions
func (tx *Transaction) createDelegateTransaction(ctx context.Context, operation string, to *types.Address, amount int64, resource core.ResourceCode, lock bool, lockPeriod int64) *Transaction {
	return tx.executeTransaction(ctx, operation, func(ctx context.Context) (*api.TransactionExtention, error) {
		return tx.client.CreateDelegateResourceTransaction(
			ctx,
			tx.owner.Bytes(),
			to.Bytes(),
			amount,
			resource,
			lock,
			lockPeriod,
		)
	})
}

// createUndelegateTransaction is a helper for undelegate transactions
func (tx *Transaction) createUndelegateTransaction(ctx context.Context, operation string, to *types.Address, amount int64, resource core.ResourceCode) *Transaction {
	return tx.executeTransaction(ctx, operation, func(ctx context.Context) (*api.TransactionExtention, error) {
		return tx.client.CreateUndelegateResourceTransaction(
			ctx,
			tx.owner.Bytes(),
			to.Bytes(),
			amount,
			resource,
		)
	})
}

// createWithdrawTransaction is a helper for withdraw transactions
func (tx *Transaction) createWithdrawTransaction(ctx context.Context, operation string, call func(context.Context, []byte) (*api.TransactionExtention, error)) *Transaction {
	return tx.executeTransaction(ctx, operation, func(ctx context.Context) (*api.TransactionExtention, error) {
		return call(ctx, tx.owner.Bytes())
	})
}
