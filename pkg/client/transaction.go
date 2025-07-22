package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
)

// Common error handling helpers to reduce code duplication

// ensureConnectionWithError is a helper that wraps connection errors with context

// validateTransactionResult checks the common result pattern
func validateTransactionResult(result *api.TransactionExtention, operation string) error {
	if result == nil {
		return fmt.Errorf("nil result for %s", operation)
	}
	if result.Result == nil {
		return fmt.Errorf("nil result field for %s", operation)
	}
	if !result.Result.Result {
		return fmt.Errorf("failed to create %s transaction: %v", operation, result.Result)
	}
	return nil
}

// grpcCallWrapper wraps common transaction building gRPC call patterns with proper connection management
// the function call should return a transaction extension
func (c *Client) grpcCallWrapper(ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error)) (*api.TransactionExtention, error) {
	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for %s: %w", operation, err)
	}
	defer c.pool.Put(conn)

	// Create wallet client
	walletClient := api.NewWalletClient(conn)

	// Apply client timeout to context
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()

	// Execute the call with proper context
	result, err := call(walletClient, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s transaction: %w", operation, err)
	}

	if err := validateTransactionResult(result, operation); err != nil {
		return nil, err
	}

	return result, nil
}
