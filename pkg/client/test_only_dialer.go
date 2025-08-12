//go:build !release

package client

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClientWithDialer is a test-only constructor that builds a *Client whose
// internal pool dials using the provided dialer (e.g., bufconn) rather than a real network address.
func NewClientWithDialer(endpoint string, dialer func(ctx context.Context, s string) (net.Conn, error), opts ...Option) (*Client, error) {
	// Prepare options
	co := &clientOptions{timeout: 30 * time.Second, initConnections: 1, maxConnections: 2}
	for _, opt := range opts {
		opt(co)
	}

	// Build a factory that uses the provided dialer
	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		return grpc.NewClient(
			endpoint,
			grpc.WithContextDialer(dialer),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	// Set sane defaults mirroring NewClient
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
