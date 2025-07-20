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

// grpcCallWrapper wraps common gRPC call patterns with proper connection management
func (c *Client) grpcCallWrapper(operation string, ctx context.Context, call func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error)) (*api.TransactionExtention, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("%s failed: %w", operation, ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s failed: %w", operation, ErrContextCancelled)
	default:
	}

	// Apply client timeout to context if not already set
	// This ensures the entire operation (connection + RPC) respects the timeout
	var callCtx context.Context
	var cancel context.CancelFunc

	if _, ok := ctx.Deadline(); ok {
		// Context already has a deadline, use it as is
		callCtx = ctx
	} else {
		// Apply client timeout
		callCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	// Get connection from pool (this will also apply timeout if needed)
	conn, err := c.GetConnection(callCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for %s: %w", operation, err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	// Create wallet client
	client := api.NewWalletClient(conn)

	// Execute the call with proper context
	result, err := call(client, callCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s transaction: %w", operation, err)
	}

	if err := validateTransactionResult(result, operation); err != nil {
		return nil, err
	}

	return result, nil
}
