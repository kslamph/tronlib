package client

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Client represents a Tron network client
type Client struct {
	conn   *grpc.ClientConn
	opts   *ClientOptions
	wallet api.WalletClient

	endpoints []string     // List of available endpoints
	mu        sync.RWMutex // Protects current connection state
	current   int          // Current endpoint index
}

// NewClient creates a new Tron client with the given options
func NewClient(ctx context.Context, opts *ClientOptions) (*Client, error) {
	if err := opts.validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	client := &Client{
		opts:      opts,
		endpoints: opts.Endpoints,
		current:   rand.Intn(len(opts.Endpoints)), // Start with random endpoint
	}

	// Establish initial connection
	if err := client.connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to establish initial connection: %w", err)
	}

	return client, nil
}

// connect establishes a gRPC connection to the current endpoint
func (c *Client) connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
	}

	// Create connection with timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		c.endpoints[c.current],
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.endpoints[c.current], err)
	}

	c.conn = conn
	c.wallet = api.NewWalletClient(conn)

	return nil
}

// reconnect attempts to establish a new connection to the specified endpoint
func (c *Client) reconnect(ctx context.Context, endpoint string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find endpoint index
	for i, ep := range c.endpoints {
		if ep == endpoint {
			c.current = i
			break
		}
	}

	return c.connect(ctx)
}

// Close closes the client connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// executeWithFailover executes an operation with automatic failover and retry logic
func (c *Client) executeWithFailover(ctx context.Context, op func(context.Context) error) error {
	var lastErr error
	attempts := 0

	for attempts <= c.opts.RetryConfig.MaxAttempts {
		err := op(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		attempts++

		// If we've exhausted all attempts, return the last error
		if attempts > c.opts.RetryConfig.MaxAttempts {
			return fmt.Errorf("all attempts failed, last error: %w", lastErr)
		}

		// Check if error is retryable
		if !c.isRetryableError(err) {
			return err
		}

		// Try next endpoint
		endpoint := c.getNextEndpoint()
		if err := c.reconnect(ctx, endpoint); err != nil {
			continue
		}

		// Apply backoff before retry
		c.applyBackoff(attempts - 1) // -1 because attempts is 1-based
	}

	return fmt.Errorf("all attempts failed, last error: %w", lastErr)
}

// getNextEndpoint returns the next endpoint to try
func (c *Client) getNextEndpoint() string {
	c.mu.Lock()
	defer c.mu.Unlock()

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
	// Check if the error is a gRPC status error
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error, might be a connection error during Dial.
		// Consider context deadline exceeded or other net errors if needed,
		// but Unavailable usually covers transient connection issues post-dial.
		// For simplicity, we'll primarily rely on gRPC codes for now.
		// A more robust implementation could check for specific net error types.
		return false
	}

	// Retry on Unavailable status code, which often indicates transient network issues
	// or the server being temporarily down.
	return st.Code() == codes.Unavailable
}

// CreateTransaction2 creates a new transaction using the v2 API
func (c *Client) BuildTransaction(contract interface{}) (*api.TransactionExtention, error) {
	ctx := context.Background()
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
	ctx := context.Background()
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
