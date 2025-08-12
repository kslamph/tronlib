// Package client provides core infrastructure for gRPC client management
package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kslamph/tronlib/pkg/types"
)

// Client errors
var (
	ErrConnectionFailed = types.NewTronError(1001, "connection to node failed", nil)
	ErrClientClosed     = types.NewTronError(1002, "client is closed", nil)
	ErrContextCancelled = types.NewTronError(1003, "context cancelled", nil)
)

// Functional options for Client
type Option func(*clientOptions)

type clientOptions struct {
	timeout         time.Duration
	initConnections int
	maxConnections  int
}

// WithTimeout sets the default timeout for client operations when the context has no deadline
func WithTimeout(d time.Duration) Option {
	return func(co *clientOptions) { co.timeout = d }
}

// WithPool configures the initial and maximum connections for the pool
func WithPool(initConnections, maxConnections int) Option {
	return func(co *clientOptions) {
		co.initConnections = initConnections
		co.maxConnections = maxConnections
	}
}

// Client manages connection to a single Tron node with connection pooling
type Client struct {
	pool        *connPool
	timeout     time.Duration
	nodeAddress string
	closed      int32
}

// NewClient creates a new client to a TRON node using endpoint like grpc://host:port or grpcs://host:port
func NewClient(endpoint string, opts ...Option) (*Client, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("node address must be provided")
	}

	// Enforce scheme-based address: grpc://host:port or grpcs://host:port
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" {
		return nil, fmt.Errorf("invalid node address, expected scheme://host:port (e.g., grpc://127.0.0.1:50051)")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "grpc" && scheme != "grpcs" {
		return nil, fmt.Errorf("unsupported scheme %q for node address; use grpc:// or grpcs://", parsed.Scheme)
	}
	hostPort := parsed.Host
	if hostPort == "" {
		return nil, fmt.Errorf("invalid node address, missing host:port")
	}

	// Apply options with defaults
	co := &clientOptions{
		timeout:         30 * time.Second,
		initConnections: 1,
		maxConnections:  5,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(co)
		}
	}
	if co.maxConnections <= 0 {
		co.maxConnections = 5
	}
	if co.initConnections <= 0 {
		co.initConnections = 1
	}

	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		// Dial using credentials based on scheme
		if scheme == "grpcs" {
			return grpc.NewClient(hostPort, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
		}
		return grpc.NewClient(hostPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Use the same timeout for connection pool
	pool, err := newConnPool(factory, co.initConnections, co.maxConnections)
	if err != nil {
		return nil, err
	}

	return &Client{
		pool:        pool,
		timeout:     co.timeout,
		nodeAddress: endpoint,
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
