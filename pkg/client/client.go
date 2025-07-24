// Package client provides core infrastructure for gRPC client management
package client

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/types"
)

// Client errors
var (
	ErrConnectionFailed = types.NewTronError(1001, "connection to node failed", nil)
	ErrClientClosed     = types.NewTronError(1002, "client is closed", nil)
	ErrContextCancelled = types.NewTronError(1003, "context cancelled", nil)
)

// ClientConfig represents the configuration for the TronClient
type ClientConfig struct {
	NodeAddress     string        // Single node address
	Timeout         time.Duration // Universal timeout for all operations (connection + RPC calls)
	InitConnections int           // Initial number of connections in pool
	MaxConnections  int           // Maximum number of connections in pool
	IdleTimeout     time.Duration // How long connections can be idle
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig(nodeAddress string) ClientConfig {
	return ClientConfig{
		NodeAddress:     nodeAddress,
		Timeout:         30 * time.Second,
		InitConnections: 1,
		MaxConnections:  5,
		IdleTimeout:     5 * time.Minute,
	}
}

// Client manages connection to a single Tron node with connection pooling
type Client struct {
	pool        *ConnPool
	timeout     time.Duration
	nodeAddress string
	closed      int32
}

// NewClient creates a new TronClient with the provided configuration (lazy connection)
func NewClient(config ClientConfig) (*Client, error) {
	if config.NodeAddress == "" {
		return nil, fmt.Errorf("node address must be provided")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout for all operations
	}

	maxConnections := config.MaxConnections
	if maxConnections <= 0 {
		maxConnections = 5 // Default pool size
	}

	initConnections := config.InitConnections
	if initConnections <= 0 {
		initConnections = 1 // Default initial pool size
	}

	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		return grpc.NewClient(config.NodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Use the same timeout for connection pool
	pool, err := NewConnPool(factory, initConnections, maxConnections)
	if err != nil {
		return nil, err
	}

	return &Client{
		pool:        pool,
		timeout:     timeout,
		nodeAddress: config.NodeAddress,
	}, nil
}

// GetConnection safely gets a connection from the pool
func (c *Client) GetConnection(ctx context.Context) (*grpc.ClientConn, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, ErrClientClosed
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ErrContextCancelled
	default:
	}

	// Apply client timeout if context doesn't have a deadline
	// This ensures the entire operation (connection + RPC) respects the timeout
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	return c.pool.Get(ctx)
}

// ReturnConnection safely returns a connection to the pool
func (c *Client) ReturnConnection(conn *grpc.ClientConn) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return
	}
	c.pool.Put(conn)
}

// Close closes the client and all connections in the pool
func (c *Client) Close() {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return // Already closed
	}
	c.pool.Close()
}

// GetTimeout returns the client's configured timeout
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}

// GetNodeAddress returns the configured node address
func (c *Client) GetNodeAddress() string {
	return c.nodeAddress
}

// IsConnected checks if the client is connected (not closed)
func (c *Client) IsConnected() bool {
	return atomic.LoadInt32(&c.closed) == 0
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
func (c *Client) grpcTransactionCallWrapper(ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error)) (*api.TransactionExtention, error) {
	return grpcGenericCallWrapper(c, ctx, operation, call, validateTransactionResult)
}