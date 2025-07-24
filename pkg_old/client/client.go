package client

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	// ErrConnectionFailed is returned when connection to the node fails
	ErrConnectionFailed = errors.New("connection to node failed")
	// ErrClientClosed is returned when trying to use a closed client
	ErrClientClosed = errors.New("client is closed")
	// ErrContextCancelled is returned when context is cancelled
	ErrContextCancelled = errors.New("context cancelled")
)

// ClientConfig represents the configuration for the TronClient
type ClientConfig struct {
	NodeAddress     string        // Single node address
	Timeout         time.Duration // Universal timeout for all operations (connection + RPC calls)
	InitConnections int           // Initial number of connections in pool
	MaxConnections  int           // Maximum number of connections in pool
	IdleTimeout     time.Duration // How long connections can be idle
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
		return nil, errors.New("node address must be provided")
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

// Transaction creation methods
