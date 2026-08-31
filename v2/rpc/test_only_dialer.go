//go:build !release

package rpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClientWithDialer is a test-only constructor (stripped from release builds
// via the release build tag, mirroring v1 pkg/client/test_only_dialer.go) that
// builds a *Client whose internal pool dials using the provided dialer (e.g.,
// bufconn) rather than a real network address. Endpoint validation is bypassed
// so passthrough:///bufnet works.
func NewClientWithDialer(endpoint string, dialer func(ctx context.Context, s string) (net.Conn, error), opts ...DialOption) (*Client, error) {
	co := &clientOptions{timeout: 30 * time.Second, initConnections: 1, maxConnections: 2}
	for _, opt := range opts {
		if opt != nil {
			opt(co)
		}
	}

	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		return grpc.NewClient(
			endpoint,
			grpc.WithContextDialer(dialer),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	pool, err := newConnPool(factory, co.initConnections, co.maxConnections)
	if err != nil {
		return nil, classifyConnError("rpc.NewClientWithDialer", err)
	}

	return &Client{
		pool:        pool,
		timeout:     co.timeout,
		nodeAddress: endpoint,
	}, nil
}
