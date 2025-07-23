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

	// fallback to context with timeout if no deadline is set
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, c.GetTimeout())
		defer cancel()
	}

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

// ValidationFunc is a function type for validating gRPC call results
// T represents the return type of the gRPC call
type ValidationFunc[T any] func(result T, operation string) error

// grpcGenericCallWrapper wraps common gRPC call patterns with proper connection management
// T represents the return type of the gRPC call
// This generic wrapper can handle any gRPC operation return type while maintaining type safety
func grpcGenericCallWrapper[T any](c *Client, ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (T, error), validateFunc ...ValidationFunc[T]) (T, error) {
	var zero T // zero value for type T
	
	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to get connection for %s: %w", operation, err)
	}
	defer c.pool.Put(conn)

	// Create wallet client
	walletClient := api.NewWalletClient(conn)

	// fallback to context with timeout if no deadline is set
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, c.GetTimeout())
		defer cancel()
	}

	// Execute the call with proper context
	result, err := call(walletClient, ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to execute %s: %w", operation, err)
	}

	// Apply validation if provided
	if len(validateFunc) > 0 && validateFunc[0] != nil {
		if err := validateFunc[0](result, operation); err != nil {
			return zero, err
		}
	}

	return result, nil
}

// Placeholder validation functions for different return types
// TODO: Implement specific validation logic for different operation types

// validateGenericResult is a placeholder validation function for generic results
func validateGenericResult[T any](result T, operation string) error {
	// Placeholder implementation - add specific validation logic as needed
	// This can be customized per operation type in the future
	return nil
}