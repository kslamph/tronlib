package workflow

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
)

// Broadcaster provides transaction broadcasting utilities
type Broadcaster struct {
	client *client.Client
}

// NewBroadcaster creates a new broadcaster instance
func NewBroadcaster(client *client.Client) *Broadcaster {
	return &Broadcaster{
		client: client,
	}
}

// Broadcast broadcasts a signed transaction
func (b *Broadcaster) Broadcast(ctx context.Context, signedTx *core.Transaction) (*api.Return, error) {
	if signedTx == nil {
		return nil, fmt.Errorf("signed transaction cannot be nil")
	}

	// Validate transaction has signatures
	if len(signedTx.Signature) == 0 {
		return nil, fmt.Errorf("transaction must be signed before broadcasting")
	}

	return lowlevel.BroadcastTransaction(b.client, ctx, signedTx)
}

// BroadcastWithValidation broadcasts a transaction with additional validation
func (b *Broadcaster) BroadcastWithValidation(ctx context.Context, signedTx *core.Transaction) (*api.Return, error) {
	// Perform basic validation
	if err := b.validateTransaction(signedTx); err != nil {
		return nil, fmt.Errorf("transaction validation failed: %w", err)
	}

	// Broadcast the transaction
	result, err := b.Broadcast(ctx, signedTx)
	if err != nil {
		return nil, err
	}

	// Check result
	if !result.Result {
		return result, fmt.Errorf("broadcast failed: %s", string(result.Message))
	}

	return result, nil
}

// validateTransaction performs basic transaction validation
func (b *Broadcaster) validateTransaction(tx *core.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	if tx.RawData == nil {
		return fmt.Errorf("transaction raw data cannot be nil")
	}

	if len(tx.Signature) == 0 {
		return fmt.Errorf("transaction must have at least one signature")
	}

	if len(tx.RawData.Contract) == 0 {
		return fmt.Errorf("transaction must have at least one contract")
	}

	// Check expiration
	if tx.RawData.Expiration <= 0 {
		return fmt.Errorf("transaction expiration must be set")
	}

	return nil
}

// GetTransactionSignWeight gets the signature weight for a transaction
func (b *Broadcaster) GetTransactionSignWeight(ctx context.Context, tx *core.Transaction) (*api.TransactionSignWeight, error) {
	return lowlevel.GetTransactionSignWeight(b.client, ctx, tx)
}

// GetTransactionApprovedList gets the approved list for a transaction
func (b *Broadcaster) GetTransactionApprovedList(ctx context.Context, tx *core.Transaction) (*api.TransactionApprovedList, error) {
	return lowlevel.GetTransactionApprovedList(b.client, ctx, tx)
}