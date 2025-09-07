package lowlevel

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"github.com/kslamph/tronlib/pb/api"
)

// ConnProvider abstracts how to obtain and release gRPC connections.
type ConnProvider interface {
	GetConnection(ctx context.Context) (*grpc.ClientConn, error)
	ReturnConnection(conn *grpc.ClientConn)
	GetTimeout() time.Duration
}

// ValidationFunc allows optional validation of RPC results.
type ValidationFunc[T any] func(result T, operation string) error

// Call wraps the lifecycle for a WalletClient RPC.
func Call[T any](cp ConnProvider, ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (T, error), validateFunc ...ValidationFunc[T]) (T, error) {
	var zero T

	conn, err := cp.GetConnection(ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to get connection for %s: %w", operation, err)
	}
	defer func() {
		if conn != nil {
			cp.ReturnConnection(conn)
		}
	}()

	cl := api.NewWalletClient(conn)

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cp.GetTimeout())
		defer cancel()
	}

	result, err := call(cl, ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to execute %s: %w", operation, err)
	}
	if len(validateFunc) > 0 && validateFunc[0] != nil {
		if err := validateFunc[0](result, operation); err != nil {
			return zero, err
		}
	}
	return result, nil
}
