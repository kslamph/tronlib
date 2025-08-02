 //go:build !release

package client

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClientWithDialer is a test-only constructor that builds a *Client whose
// internal pool dials using the provided dialer (e.g., bufconn) rather than a real network address.
func NewClientWithDialer(config ClientConfig, dialer func(ctx context.Context, s string) (net.Conn, error)) (*Client, error) {
	// Clone config to avoid mutation
	cfg := config

	// Build a factory that uses the provided dialer
	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		return grpc.DialContext(
			ctx,
			cfg.NodeAddress,
			grpc.WithContextDialer(dialer),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	// Set sane defaults mirroring NewClient
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultClientConfig(cfg.NodeAddress).Timeout
	}
	maxConnections := cfg.MaxConnections
	if maxConnections <= 0 {
		maxConnections = DefaultClientConfig(cfg.NodeAddress).MaxConnections
	}
	initConnections := cfg.InitConnections
	if initConnections <= 0 {
		initConnections = DefaultClientConfig(cfg.NodeAddress).InitConnections
	}

	pool, err := newConnPool(factory, initConnections, maxConnections)
	if err != nil {
		return nil, err
	}

	return &Client{
		pool:        pool,
		timeout:     timeout,
		nodeAddress: cfg.NodeAddress,
	}, nil
}