package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Client represents a Tron network client
// RateLimiter tracks the last call time for each endpoint
type RateLimiter struct {
	lastCall map[string]time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		lastCall: make(map[string]time.Time),
	}
}

// CheckAndUpdate checks if enough time has passed since the last call and updates the timestamp
func (r *RateLimiter) CheckAndUpdate(endpoint string, minGap time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	lastTime, exists := r.lastCall[endpoint]

	if !exists || now.Sub(lastTime) >= minGap {
		r.lastCall[endpoint] = now
		return true
	}
	return false
}

type Client struct {
	conn   *grpc.ClientConn
	opts   *ClientOptions
	wallet api.WalletClient

	endpoints []string     // List of available endpoints
	mu        sync.RWMutex // Protects current connection state
	current   int          // Current endpoint index

	rateLimiter *RateLimiter                     // Rate limiter for API calls
	backlogChan chan func(context.Context) error // Channel for backlog tasks
	workerWg    sync.WaitGroup                   // WaitGroup for tracking workers
}

// NewClient creates a new Tron client with the given options
// NewClient creates a new Tron client with the given options
func NewClient(ctx context.Context, opts *ClientOptions) (*Client, error) {
	if err := opts.validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	// Shuffle endpoints for random initial order
	shuffled := make([]string, len(opts.Endpoints))
	copy(shuffled, opts.Endpoints)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	client := &Client{
		opts:        opts,
		endpoints:   shuffled,
		current:     0,
		rateLimiter: NewRateLimiter(),
		backlogChan: make(chan func(context.Context) error, 1000), // Buffer size for backlog
	}

	// Start priority workers
	for i := 0; i < opts.PriorityWorkers; i++ {
		client.workerWg.Add(1)
		go client.priorityWorker()
	}

	// Start backlog worker (uses remaining CPU capacity)
	client.workerWg.Add(1)
	go client.backlogWorker()

	// Try each endpoint until one works
	var lastErr error
	for i := 0; i < len(opts.Endpoints); i++ {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return nil, fmt.Errorf("connection attempts aborted: %w (last error: %v)", ctx.Err(), lastErr)
			}
			return nil, fmt.Errorf("connection attempts aborted: %w", ctx.Err())
		default:
			connCtx, cancel := context.WithTimeout(ctx, opts.Timeout)

			err := client.connect(connCtx)
			if err != nil {
				lastErr = err

				cancel()
				client.current = (client.current + 1) % len(opts.Endpoints)
				time.Sleep(50 * time.Millisecond) // Brief pause before next attempt
				continue
			}

			cancel()

			return client, nil
		}
	}

	return nil, fmt.Errorf("failed to establish initial connection to any endpoint (tried %d endpoints), last error: %w",
		len(opts.Endpoints), lastErr)
}

// connect establishes a gRPC connection to the specified endpoint
// This is an internal method used by both NewClient and reconnect
func (c *Client) connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	endpoint := c.endpoints[c.current]

	// Ensure any existing connection is properly cleaned up
	if c.conn != nil {
		if state := c.conn.GetState(); state != connectivity.Shutdown {
			c.conn.Close()
		}
		c.conn = nil
		c.wallet = nil
	}

	// Attempt to establish new connection
	conn, err := c.connectWithNewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", endpoint, err)
	}

	// Verify connection state
	state := conn.GetState()
	if state != connectivity.Ready && state != connectivity.Idle {
		conn.Close()
		return fmt.Errorf("connection established but in unusable state: %v", state)
	}

	// Update client state with new connection
	c.conn = conn
	c.wallet = api.NewWalletClient(conn)

	return nil
}

// reconnect attempts to establish a new connection to the specified endpoint
// This is used by executeWithFailover to handle connection failures and endpoint switching
// It ensures proper cleanup of existing connections and maintains endpoint indexing
// func (c *Client) reconnect(ctx context.Context, endpoint string) error {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	// Clean up existing connection
// 	if c.conn != nil {
// 		if state := c.conn.GetState(); state != connectivity.Shutdown {
// 			c.conn.Close()
// 		}
// 		c.conn = nil
// 		c.wallet = nil
// 	}

// 	// Update current endpoint index
// 	found := false
// 	for i, ep := range c.endpoints {
// 		if ep == endpoint {
// 			c.current = i
// 			found = true
// 			break
// 		}
// 	}
// 	if !found {
// 		return fmt.Errorf("invalid endpoint: %s", endpoint)
// 	}

// 	// Attempt connection with new endpoint
// 	err := c.connect(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to reconnect to %s: %w", endpoint, err)
// 	}

// 	return nil
// }

// Close closes the client connection
// Close closes the client connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		// Signal all workers to stop
		for i := 0; i < c.opts.PriorityWorkers+1; i++ { // +1 for backlog worker
			c.backlogChan <- nil
		}
		c.workerWg.Wait()    // Wait for workers to finish
		close(c.backlogChan) // Close channel after all workers are done
		return c.conn.Close()
	}
	return nil
}

// executeWithFailover executes an operation with automatic failover and retry logic
func (c *Client) executeWithFailover(ctx context.Context, op func(context.Context) error) error {
	var lastErr error
	attempts := 0
	maxAttempts := c.opts.RetryConfig.MaxAttempts

	for attempts <= maxAttempts {
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		default:
			currentEndpoint := c.endpoints[c.current]

			// Check rate limit
			if !c.rateLimiter.CheckAndUpdate(currentEndpoint, c.opts.RateLimit) {
				// If rate limited, try next endpoint
				c.current = (c.current + 1) % len(c.endpoints)
				continue
			}

			if attempts > 0 {
				c.applyBackoff(attempts - 1)

			}

			// Check connection state
			if c.conn != nil {
				state := c.conn.GetState()
				if state != connectivity.Ready && state != connectivity.Idle {

					c.conn.Close()
					c.conn = nil
				}
			}

			// Establish connection if needed
			if c.conn == nil {
				connCtx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
				err := c.connect(connCtx)
				cancel()

				if err != nil {
					lastErr = err
					attempts++

					continue
				}
			}

			// Execute operation with timeout
			opCtx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
			err := op(opCtx)
			cancel()

			if err == nil {
				// Operation succeeded
				return nil
			}

			lastErr = err
			attempts++

			if attempts > maxAttempts {
				return fmt.Errorf("all attempts exhausted (%d/%d), last error: %w",
					attempts, maxAttempts+1, lastErr)
			}

			// Check if error is retryable
			if !c.isRetryableError(err) {
				return fmt.Errorf("non-retryable error on %s: %w",
					currentEndpoint, err)
			}

			// Get next endpoint (random selection)

		}
	}

	return fmt.Errorf("all endpoints tried (%d attempts), last error: %w",
		attempts, lastErr)
}

// getNextEndpoint returns a randomly selected endpoint, excluding the current one
func (c *Client) getNextEndpoint() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If we only have one endpoint, return it
	if len(c.endpoints) == 1 {
		return c.endpoints[0]
	}

	// Create a list of available indices excluding current
	available := make([]int, 0, len(c.endpoints)-1)
	for i := range c.endpoints {
		if i != c.current {
			available = append(available, i)
		}
	}

	// Randomly select from available endpoints
	if len(available) > 0 {
		nextIndex := available[rand.Intn(len(available))]
		c.current = nextIndex
		return c.endpoints[nextIndex]
	}

	// Fallback: if somehow we have no available endpoints, wrap around
	c.current = (c.current + 1) % len(c.endpoints)
	return c.endpoints[c.current]
}

// applyBackoff waits for an exponentially increasing duration
func (c *Client) applyBackoff(attempt int) {
	base := 1 << uint(attempt)
	multiplier := float64(base) * c.opts.RetryConfig.BackoffFactor
	backoff := time.Duration(float64(c.opts.RetryConfig.InitialBackoff) * multiplier)
	if backoff > c.opts.RetryConfig.MaxBackoff {
		backoff = c.opts.RetryConfig.MaxBackoff
	}
	time.Sleep(backoff)
}

// isRetryableError checks if an error should trigger a retry
func (c *Client) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// First check for context and network errors as they're most common
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {

		return true
	}

	// Handle gRPC status codes
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable,
			codes.DeadlineExceeded,
			codes.Unknown,
			codes.Aborted,
			codes.DataLoss:
			// These are typically transient issues that warrant retry
			// fmt.Printf("Retryable gRPC error (code=%v): %v\n", st.Code(), st.Message())
			return true

		case codes.ResourceExhausted:
			// Rate limiting - worth retrying on another endpoint
			// fmt.Printf("Rate limit hit, will retry on another endpoint: %v\n", st.Message())
			return true

		case codes.Internal,
			codes.FailedPrecondition:
			// These might be temporary issues
			// fmt.Printf("Potentially retryable gRPC error (code=%v): %v\n", st.Code(), st.Message())
			return true

		case codes.InvalidArgument,
			codes.NotFound,
			codes.AlreadyExists,
			codes.PermissionDenied,
			codes.Unauthenticated,
			codes.OutOfRange,
			codes.Unimplemented:
			// These are definitely not retryable
			// fmt.Printf("Non-retryable gRPC error (code=%v): %v\n", st.Code(), st.Message())
			return false

		default:
			// Log unknown codes but treat them as retryable
			// fmt.Printf("Unknown gRPC error code %v, will retry: %v\n", st.Code(), st.Message())
			return true
		}
	}

	// Handle network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Handle connection closed
	if errors.Is(err, io.EOF) {
		// fmt.Printf("Connection closed unexpectedly, will retry\n")
		return true
	}

	// Unknown error type, treating as non-retryable
	return false
}

// priorityWorker handles high-priority tasks
func (c *Client) priorityWorker() {
	defer c.workerWg.Done()

	for task := range c.backlogChan {
		if task == nil {
			return
		}

		ctx := context.Background()
		if err := task(ctx); err != nil {
			// Log error but continue processing
			fmt.Printf("Priority worker error: %v\n", err)
		}
	}
}

// backlogWorker processes backlog tasks using remaining CPU capacity
func (c *Client) backlogWorker() {
	defer c.workerWg.Done()

	for task := range c.backlogChan {
		if task == nil {
			return
		}

		// Process backlog tasks with a background context
		ctx := context.Background()
		if err := task(ctx); err != nil {
			// Log error but continue processing
			fmt.Printf("Backlog worker error: %v\n", err)
		}

		// Small sleep to prevent CPU saturation
		time.Sleep(10 * time.Millisecond)
	}
}

// CreateTransaction2 creates a new transaction using the v2 API
func (c *Client) BuildTransaction(contract interface{}) (*api.TransactionExtention, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()

	apiExt := &api.TransactionExtention{}
	err := c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		switch v := contract.(type) {
		case *core.TransferContract:
			apiExt, err = c.wallet.CreateTransaction2(ctx, v)
		case *core.TriggerSmartContract:
			apiExt, err = c.wallet.TriggerConstantContract(ctx, v)
		case *core.FreezeBalanceV2Contract:
			apiExt, err = c.wallet.FreezeBalanceV2(ctx, v)
		case *core.UnfreezeBalanceV2Contract:
			apiExt, err = c.wallet.UnfreezeBalanceV2(ctx, v)
		case *core.DelegateResourceContract:
			apiExt, err = c.wallet.DelegateResource(ctx, v)
		case *core.UnDelegateResourceContract:
			apiExt, err = c.wallet.UnDelegateResource(ctx, v)
		case *core.WithdrawExpireUnfreezeContract:
			apiExt, err = c.wallet.WithdrawExpireUnfreeze(ctx, v)
		default:
			return fmt.Errorf("BuildTransaction failed: unsupported contract type: %T", contract)
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("BuildTransaction failed: %v", err)
	}
	if apiExt.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("BuildTransaction failed: %s", apiExt.GetResult().GetMessage())
	}
	return apiExt, nil
}

// BroadcastTransaction broadcasts a signed transaction to the network
func (c *Client) BroadcastTransaction(tx *core.Transaction) (*api.Return, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()

	var result *api.Return
	err := c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		result, err = c.wallet.BroadcastTransaction(ctx, tx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %v", err)
	}
	return result, nil
}

func (c *Client) connectWithNewClient(ctx context.Context) (*grpc.ClientConn, error) {
	// Configure connection parameters with timeouts and backoff
	connectParams := grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  100 * time.Millisecond,
			Multiplier: 1.6,
			Jitter:     0.2,
			MaxDelay:   c.opts.RetryConfig.MaxBackoff,
		},
		MinConnectTimeout: c.opts.Timeout,
	}

	// Configure keepalive parameters
	kp := keepalive.ClientParameters{
		Time:                15 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	target := c.endpoints[c.current]

	// Use DialContext with the provided context and make it blocking
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		grpc.WithConnectParams(connectParams),
		grpc.WithKeepaliveParams(kp),
		grpc.WithBlock(), // Make connection establishment blocking
	}

	conn, err := grpc.DialContext(ctx, target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", target, err)
	}

	return conn, nil
}
