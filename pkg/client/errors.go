package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
)

// Common error handling helpers to reduce code duplication

// ensureConnectionWithError is a helper that wraps connection errors with context
func (c *Client) ensureConnectionWithError(operation string) error {
	if err := c.ensureConnection(); err != nil {
		return fmt.Errorf("connection error for %s: %v", operation, err)
	}
	return nil
}

// validateTransactionResult checks the common result pattern
func validateTransactionResult(result *api.TransactionExtention, operation string) error {
	if result == nil {
		return fmt.Errorf("nil result for %s", operation)
	}
	if !result.Result.Result {
		return fmt.Errorf("failed to create %s transaction: %v", operation, result.Result)
	}
	return nil
}

// grpcCallWrapper wraps common gRPC call patterns
func (c *Client) grpcCallWrapper(operation string, ctx context.Context, call func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error)) (*api.TransactionExtention, error) {
	if err := c.ensureConnectionWithError(operation); err != nil {
		return nil, err
	}

	client := api.NewWalletClient(c.conn)
	result, err := call(client, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s transaction: %v", operation, err)
	}

	if err := validateTransactionResult(result, operation); err != nil {
		return nil, err
	}

	return result, nil
}
