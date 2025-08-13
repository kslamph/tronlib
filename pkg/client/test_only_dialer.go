// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
