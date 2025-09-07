// Package client provides high-level client functionality. Low-level 1:1 gRPC
// helpers are implemented in package lowlevel.
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
)

// ValidationFunc is a function type for validating gRPC call results
// T represents the return type of the gRPC call
// Re-exported from lowlevel for backward compatibility.
type ValidationFunc[T any] = lowlevel.ValidationFunc[T]

// grpcGenericCallWrapper wraps common gRPC call patterns using the lowlevel.Call
// while keeping the high-level client signature unchanged.
func grpcGenericCallWrapper[T any](c *Client, ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (T, error), validateFunc ...ValidationFunc[T]) (T, error) {
	return lowlevel.Call(c, ctx, operation, call, validateFunc...)
}
