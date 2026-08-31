package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/v2/tron"
)

// ConnProvider abstracts how to obtain and release gRPC connections.
// *Client implements it; the free-function wrappers in this package (and the
// tests that exercise them) accept any ConnProvider, which is the seam that
// keeps the ~1:1 gRPC wrappers testable against a bufconn-backed fake.
type ConnProvider interface {
	GetConnection(ctx context.Context) (*grpc.ClientConn, error)
	ReturnConnection(conn *grpc.ClientConn)
	GetTimeout() time.Duration
}

// ValidationFunc allows optional validation of RPC results.
type ValidationFunc[T any] func(result T, operation string) error

// classifyConnError maps a raw transport error onto the v2 code vocabulary.
// An error that is already a *tron.Error (e.g. chain.closed from GetConnection)
// passes through unchanged; context cancellation/deadline expiry becomes
// chain.timeout with the context error preserved as Cause (so
// errors.Is(err, context.DeadlineExceeded) still holds); everything else is a
// connection failure.
func classifyConnError(op string, err error) error {
	var te *tron.Error
	if errors.As(err, &te) {
		return te
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) ||
		status.Code(err) == codes.DeadlineExceeded ||
		status.Code(err) == codes.Canceled {
		return &tron.Error{Code: tron.CodeChainTimeout, Op: op, Cause: err}
	}
	return &tron.Error{Code: tron.CodeChainConnection, Op: op, Cause: err}
}

// Call wraps the lifecycle for a WalletClient RPC: acquire a connection from
// the provider, apply the provider's timeout when the context has no deadline,
// invoke the call, and run the optional validator. See the package comment for
// the error-code mapping.
func Call[T any](cp ConnProvider, ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (T, error), validateFunc ...ValidationFunc[T]) (T, error) {
	var zero T

	conn, err := cp.GetConnection(ctx)
	if err != nil {
		return zero, classifyConnError(operation, err)
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
		// A deadline hit mid-call is chain.timeout; any other node/RPC
		// failure is rpc.method_failed with the operation in Op (the v2
		// boundary every free-function wrapper inherits). *tron.Error raised
		// inside the call passes through untouched.
		var te *tron.Error
		switch {
		case errors.As(err, &te):
			return zero, err
		case errors.Is(err, context.DeadlineExceeded),
			errors.Is(err, context.Canceled),
			status.Code(err) == codes.DeadlineExceeded,
			status.Code(err) == codes.Canceled:
			return zero, &tron.Error{Code: tron.CodeChainTimeout, Op: operation, Cause: err}
		default:
			return zero, &tron.Error{Code: tron.CodeRPCMethodFailed, Op: operation, Cause: err}
		}
	}
	if len(validateFunc) > 0 && validateFunc[0] != nil {
		if err := validateFunc[0](result, operation); err != nil {
			var te *tron.Error
			if errors.As(err, &te) {
				return zero, err
			}
			return zero, &tron.Error{Code: tron.CodeRPCMethodFailed, Op: operation, Cause: err}
		}
	}
	return result, nil
}

// --- Dial options ---

// DialOption configures a Client at Dial time.
type DialOption func(*clientOptions)

type clientOptions struct {
	timeout         time.Duration
	initConnections int
	maxConnections  int
}

// WithTimeout sets the default timeout for client operations when the context
// has no deadline. The default is 30 seconds.
func WithTimeout(d time.Duration) DialOption {
	return func(co *clientOptions) { co.timeout = d }
}

// WithPool configures the initial and maximum connections for the pool:
//   - initConnections: connections created on demand up front (default: 1)
//   - maxConnections: maximum number of connections in the pool (default: 5)
func WithPool(initConnections, maxConnections int) DialOption {
	return func(co *clientOptions) {
		co.initConnections = initConnections
		co.maxConnections = maxConnections
	}
}

// Client manages the connection pool to a single TRON node.
//
// Use Dial to create one, and always call Close when finished. Close is
// idempotent; after Close, GetConnection fails with chain.closed.
type Client struct {
	pool        *connPool
	timeout     time.Duration
	nodeAddress string
	closed      int32
}

// Dial creates a new Client to a TRON node at endpoint, which must be
// grpc://host:port (plaintext) or grpcs://host:port (TLS). Anything else
// fails with chain.connection and a remediation Hint.
//
// Dial is lazy: like v1's NewClient it builds the connection factory without
// network I/O, so reachability is proven by the first call, not by Dial.
// Options: WithTimeout (default 30s), WithPool (default 1..5; sizes <= 0
// fall back to the defaults).
func Dial(ctx context.Context, endpoint string, opts ...DialOption) (*Client, error) {
	if endpoint == "" {
		return nil, &tron.Error{
			Code: tron.CodeChainConnection,
			Op:   "rpc.Dial",
			Hint: "endpoint must be grpc://host:port or grpcs://host:port",
		}
	}

	// Enforce scheme-based address: grpc://host:port or grpcs://host:port
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" {
		return nil, &tron.Error{
			Code: tron.CodeChainConnection,
			Op:   "rpc.Dial",
			Hint: "endpoint must be grpc://host:port or grpcs://host:port",
		}
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "grpc" && scheme != "grpcs" {
		return nil, &tron.Error{
			Code: tron.CodeChainConnection,
			Op:   "rpc.Dial",
			Hint: "endpoint must be grpc://host:port or grpcs://host:port",
		}
	}
	hostPort := parsed.Host
	if hostPort == "" {
		return nil, &tron.Error{
			Code: tron.CodeChainConnection,
			Op:   "rpc.Dial",
			Hint: "endpoint must be grpc://host:port or grpcs://host:port",
		}
	}

	// Apply options with defaults (v1 semantics: sizes <= 0 fall back).
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

	pool, err := newConnPool(factory, co.initConnections, co.maxConnections)
	if err != nil {
		return nil, classifyConnError("rpc.Dial", err)
	}

	// A caller that hands Dial an already-expired context almost certainly
	// made a mistake upstream; fail fast instead of returning a doomed client.
	select {
	case <-ctx.Done():
		return nil, &tron.Error{Code: tron.CodeChainTimeout, Op: "rpc.Dial", Cause: ctx.Err()}
	default:
	}

	return &Client{
		pool:        pool,
		timeout:     co.timeout,
		nodeAddress: endpoint,
	}, nil
}

// GetConnection safely gets a connection from the pool. Use ReturnConnection
// to give it back — or Call, which does both. After Close this fails with
// chain.closed.
func (c *Client) GetConnection(ctx context.Context) (*grpc.ClientConn, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, &tron.Error{
			Code: tron.CodeChainClosed,
			Op:   "rpc.GetConnection",
			Hint: "client is closed; create a new one with rpc.Dial",
		}
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, &tron.Error{Code: tron.CodeChainTimeout, Op: "rpc.GetConnection", Cause: ctx.Err()}
	default:
	}

	// Apply client timeout if context doesn't have a deadline.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	if c.pool == nil {
		return nil, &tron.Error{Code: tron.CodeChainConnection, Op: "rpc.GetConnection", Cause: errors.New("connection pool not initialized")}
	}

	conn, err := c.pool.get(ctx)
	if err != nil {
		if errors.Is(err, errPoolClosed) {
			return nil, &tron.Error{
				Code: tron.CodeChainClosed,
				Op:   "rpc.GetConnection",
				Hint: "client is closed; create a new one with rpc.Dial",
			}
		}
		return nil, classifyConnError("rpc.GetConnection", err)
	}
	return conn, nil
}

// ReturnConnection safely returns a connection to the pool. It is a no-op on
// a closed client and safe to call with nil.
func (c *Client) ReturnConnection(conn *grpc.ClientConn) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return
	}
	if c.pool != nil {
		c.pool.put(conn)
	}
}

// Close closes the client and all pooled connections. Idempotent.
func (c *Client) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil // Already closed
	}
	if c.pool != nil {
		c.pool.close()
	}
	return nil
}

// GetTimeout returns the client's configured timeout, applied to operations
// whose context has no deadline.
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}

// GetNodeAddress returns the configured endpoint (scheme://host:port).
func (c *Client) GetNodeAddress() string {
	return c.nodeAddress
}

// IsConnected reports whether the client is still open.
func (c *Client) IsConnected() bool {
	return atomic.LoadInt32(&c.closed) == 0
}

// errPoolClosed is returned by get when the pool's channel has been closed by
// close(); GetConnection maps it to chain.closed.
var errPoolClosed = errors.New("connection pool closed")

// --- Connection pool (ported from v1 pkg/client/connection.go) ---

// connPool manages a pool of gRPC client connections.
// For testing purposes, GetFunc can be overridden to mock connection behavior.
type connPool struct {
	mu sync.Mutex
	// closed is guarded by mu. put() checks it and sends to conns while still
	// holding mu, so no send to p.conns can happen after close(p.conns).
	closed bool
	// inPool tracks conns currently sitting in the channel so a double return
	// of the same conn cannot add a second entry (double-return guard).
	inPool      map[*grpc.ClientConn]struct{}
	conns       chan *grpc.ClientConn
	factory     func(ctx context.Context) (*grpc.ClientConn, error)
	initialSize int

	// For testing only: A function to override the Get method's behavior.
	getFunc func(ctx context.Context) (*grpc.ClientConn, error)
}

// newConnPool creates a new connection pool.
func newConnPool(factory func(ctx context.Context) (*grpc.ClientConn, error), initialSize int, capacity int) (*connPool, error) {
	if initialSize < 0 || capacity <= 0 || initialSize > capacity {
		return nil, errors.New("invalid pool configuration")
	}

	p := &connPool{
		conns:       make(chan *grpc.ClientConn, capacity),
		factory:     factory,
		initialSize: initialSize,
		inPool:      make(map[*grpc.ClientConn]struct{}),
	}

	// Don't create initial connections - let them be created on demand
	// This allows the pool to be created even if the target is not available

	return p, nil
}

// capacity returns the pool's maximum number of connections.
func (p *connPool) capacity() int {
	if p == nil || p.conns == nil {
		return 0
	}
	return cap(p.conns)
}

// take removes a received conn from the inPool set under mu.
func (p *connPool) take(conn *grpc.ClientConn) *grpc.ClientConn {
	p.mu.Lock()
	delete(p.inPool, conn)
	p.mu.Unlock()
	return conn
}

// get retrieves a connection from the pool. If no connection is available,
// it will try to create a new one if the pool has not reached its capacity.
func (p *connPool) get(ctx context.Context) (*grpc.ClientConn, error) {
	// If a mock GetFunc is provided, use it
	if p.getFunc != nil {
		return p.getFunc(ctx)
	}

	select {
	case conn := <-p.conns:
		return healthyOrNew(p, ctx, p.take(conn))
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		p.mu.Lock()
		if len(p.conns) < cap(p.conns) {
			conn, err := p.factory(ctx)
			p.mu.Unlock()
			if err != nil {
				return nil, fmt.Errorf("connection failed: %w", err)
			}
			return conn, nil
		}
		p.mu.Unlock()
		// Wait for a connection to be returned to the pool. mu is NOT held
		// here: put() needs it to return a conn, so holding it through this
		// blocking receive would deadlock (v1 got away with it because put
		// was lock-free — and racy).
		select {
		case conn := <-p.conns:
			return healthyOrNew(p, ctx, p.take(conn))
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// healthyOrNew keeps v1's health check: a pooled connection that is not Ready
// is closed and replaced by a fresh one from the factory.
func healthyOrNew(p *connPool, ctx context.Context, conn *grpc.ClientConn) (*grpc.ClientConn, error) {
	if conn == nil {
		// Only close(p.conns) yields a nil receive; the pool is closed.
		return nil, errPoolClosed
	}
	if conn.GetState() != connectivity.Ready {
		// Connection is not ready, close it and create a new one
		_ = conn.Close()
		conn, err := p.factory(ctx)
		if err != nil {
			return nil, fmt.Errorf("connection failed: %w", err)
		}
		return conn, nil
	}
	return conn, nil
}

// put returns a connection to the pool.
func (p *connPool) put(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}
	if p.conns == nil {
		// No backing channel (tests override get); close to avoid leaks.
		_ = conn.Close()
		return
	}

	// The closed check and the channel send happen under the same mutex as
	// close(), so a put that passes the check cannot race with close(p.conns).
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		// Pool closed concurrently: close the conn instead of sending on a
		// closed channel (which would panic).
		_ = conn.Close()
		return
	}
	if _, dup := p.inPool[conn]; dup {
		// Double return of the same conn: ignore the duplicate entry so one
		// conn can never be handed to two callers at once.
		return
	}
	select {
	case p.conns <- conn:
		p.inPool[conn] = struct{}{}
	default:
		// Pool is full, close the connection and ignore close error
		_ = conn.Close()
	}
}

// close closes all connections in the pool.
func (p *connPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	if p.conns == nil {
		// Nothing to close in test/fake pools.
		return
	}

	close(p.conns)
	for conn := range p.conns {
		_ = conn.Close()
	}
}
