// Package client provides core infrastructure for gRPC client management
package client

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
	// IdleTimeout     time.Duration // How long connections can be idle
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig(nodeAddress string) ClientConfig {
	return ClientConfig{
		NodeAddress:     nodeAddress,
		Timeout:         30 * time.Second,
		InitConnections: 1,
		MaxConnections:  5,
		// IdleTimeout:     5 * time.Minute,
	}
}

// Client manages connection to a single Tron node with connection pooling
type Client struct {
	pool        *connPool
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
	pool, err := newConnPool(factory, initConnections, maxConnections)
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

	if c.pool == nil {
		return nil, ErrConnectionFailed
	}

	return c.pool.get(ctx)
}

// ReturnConnection safely returns a connection to the pool
func (c *Client) ReturnConnection(conn *grpc.ClientConn) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return
	}
	if c.pool != nil {
		if c.pool != nil {
			c.pool.put(conn)
		}
	}
}

// Close closes the client and all connections in the pool
func (c *Client) Close() {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return // Already closed
	}
	if c.pool != nil {
		if c.pool != nil {
			if c.pool != nil {
				c.pool.close()
			}
		}
	}
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
// type ValidationFunc[T any] func(result T, operation string) error

// Account-related gRPC calls
