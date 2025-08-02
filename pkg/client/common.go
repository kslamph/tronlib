// package client provides 1:1 wrappers around WalletClient gRPC methods
// This package contains raw gRPC calls with minimal business logic
package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"

	"github.com/kslamph/tronlib/pkg/types"
)

// ValidationFunc is a function type for validating gRPC call results
// T represents the return type of the gRPC call
type ValidationFunc[T any] func(result T, operation string) error

// grpcGenericCallWrapper wraps common gRPC call patterns with proper connection management
// T represents the return type of the gRPC call
// This generic wrapper can handle any gRPC operation return type while maintaining type safety
func grpcGenericCallWrapper[T any](c *Client, ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (T, error), validateFunc ...ValidationFunc[T]) (T, error) {
	var zero T // zero value for type T

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to get connection for %s: %w", operation, err)
	}
	// Ensure we always return or close the connection safely.
	defer func() {
		if conn != nil && c != nil && c.pool != nil {
			c.ReturnConnection(conn)
		}
	}()

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

// validateTransactionResult checks the common result pattern for transaction operations
func validateTransactionResult(result *api.TransactionExtention, operation string) error {
	if result == nil {
		return fmt.Errorf("nil result for %s", operation)
	}
	if result.Result == nil {
		return fmt.Errorf("nil result field for %s", operation)
	}
	if !result.Result.Result {
		return types.WrapTransactionResult(result.Result, operation)
	}
	return nil
}

// grpcTransactionCallWrapper wraps gRPC calls that return TransactionExtention
// NOTE: Go does not support type parameters on methods. This helper remains a normal method without generics,
//
//	delegating to the generic function above via a type-specific call.
func (c *Client) grpcTransactionCallWrapper(ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error)) (*api.TransactionExtention, error) {
	return grpcGenericCallWrapper(c, ctx, operation, call, validateTransactionResult)
}
