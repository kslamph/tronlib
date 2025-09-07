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

// WithTimeout sets the default timeout for client operations when the context has no deadline.
//
// This option configures the default timeout that will be applied to operations
// when the context doesn't have a deadline. The default is 30 seconds.
func WithTimeout(d time.Duration) Option {
	return func(co *clientOptions) { co.timeout = d }
}

// WithPool configures the initial and maximum connections for the pool.
//
// This option configures the connection pool size:
//   - initConnections: Number of connections to create initially (default: 1)
//   - maxConnections: Maximum number of connections in the pool (default: 5)
func WithPool(initConnections, maxConnections int) Option {
	return func(co *clientOptions) {
		co.initConnections = initConnections
		co.maxConnections = maxConnections
	}
}

// Client manages connection to a single Tron node with connection pooling.
//
// The Client maintains a pool of gRPC connections to improve performance for
// concurrent operations. It automatically handles connection lifecycle,
// including reconnection and timeout management.
//
// Use NewClient to create a new client instance, and always call Close when
// finished to free up resources.
type Client struct {
	pool        *connPool
	timeout     time.Duration
	nodeAddress string
	closed      int32
}

// NewClient creates a new client to a TRON node using endpoint like grpc://host:port or grpcs://host:port
//
// The endpoint must include a scheme (grpc:// or grpcs://) followed by host and port.
// The client maintains a connection pool for improved performance.
//
// Options can be used to configure:
//   - Connection timeout with WithTimeout()
//   - Connection pool size with WithPool()
//
// Example:
//
//	cli, err := client.NewClient("grpc://127.0.0.1:50051",
//	    client.WithTimeout(30*time.Second),
//	    client.WithPool(5, 10))
//	if err != nil {
//	    // handle error
//	}
//	defer cli.Close()
//
// Returns an error if the endpoint is invalid or connection fails.
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

// GetConnection safely gets a connection from the pool.
//
// This method should be used in conjunction with ReturnConnection to properly
// manage connection lifecycle. It applies the client's default timeout if
// the context doesn't have a deadline.
//
// Returns ErrClientClosed if the client has been closed, or ErrConnectionFailed
// if no connection is available.
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

// ReturnConnection safely returns a connection to the pool.
//
// This method should always be called after GetConnection to return the
// connection to the pool for reuse. It is safe to call on a closed client.
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

// Close closes the client and all connections in the pool.
//
// This method should be called when the client is no longer needed to free
// up resources. It is safe to call multiple times.
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

// GetTimeout returns the client's configured timeout.
//
// This timeout is applied to operations when the context doesn't have a deadline.
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}

// GetNodeAddress returns the configured node address.
//
// The address is in the format scheme://host:port (e.g., grpc://127.0.0.1:50051).
func (c *Client) GetNodeAddress() string {
	return c.nodeAddress
}

// IsConnected checks if the client is connected (not closed).
//
// Returns true if the client is still open and can be used for operations,
// false if it has been closed.
func (c *Client) IsConnected() bool {
	return atomic.LoadInt32(&c.closed) == 0
}
