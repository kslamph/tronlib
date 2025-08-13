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

package client

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// Bufconn size for in-memory gRPC server
const bufSize = 1024 * 1024

// testWalletServer is a minimal fake implementing api.WalletServer for unit tests.
type testWalletServer struct {
	api.UnimplementedWalletServer

	// Handlers can be set per test to customize behavior
	BroadcastHandler            func(ctx context.Context, in *core.Transaction) (*api.Return, error)
	TriggerConstantContractFunc func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	GetTxInfoByIdHandler        func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)
}

func (s *testWalletServer) BroadcastTransaction(ctx context.Context, in *core.Transaction) (*api.Return, error) {
	if s.BroadcastHandler != nil {
		return s.BroadcastHandler(ctx, in)
	}
	return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
}

func (s *testWalletServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if s.TriggerConstantContractFunc != nil {
		return s.TriggerConstantContractFunc(ctx, in)
	}
	return &api.TransactionExtention{
		Result:     &api.Return{Result: true, Code: api.Return_SUCCESS},
		EnergyUsed: 0,
	}, nil
}

func (s *testWalletServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	if s.GetTxInfoByIdHandler != nil {
		return s.GetTxInfoByIdHandler(ctx, in)
	}
	// default: none
	return nil, nil
}

// newBufconnServer spins up a bufconn-backed gRPC server.
// Returns listener, server, and cleanup that stops the server and closes the listener.
func newBufconnServer(t *testing.T, impl api.WalletServer) (*bufconn.Listener, *grpc.Server, func()) {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, impl)

	go func() { _ = srv.Serve(lis) }()

	cleanup := func() {
		_ = lis.Close()
		srv.Stop()
	}
	return lis, srv, cleanup
}

// newTestClientWithBufConn creates a *Client via NewClient and ensures a real pool and real *grpc.ClientConn using bufconn.
func newTestClientWithBufConn(t *testing.T, lis *bufconn.Listener, timeout time.Duration) (*Client, func()) {
	t.Helper()
	// Use the test-only dialer variant to avoid scheme validation
	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		// Use DialContext to honor context cancellation/timeouts
		return lis.DialContext(ctx)
	}
	c, err := NewClientWithDialer("passthrough:///bufnet", dialer, WithTimeout(timeout), WithPool(1, 2))
	if err != nil {
		t.Fatalf("NewClientWithDialer error: %v", err)
	}

	// Ensure cleanup closes client (which closes pool) after tests.
	cleanup := func() {
		c.Close()
	}

	return c, cleanup
}

// helper to build a minimal trigger smart contract core.Transaction with one contract and future expiration
func buildTriggerSmartContractTx(expiration time.Time) *core.Transaction {
	raw := &core.TransactionRaw{
		Contract: []*core.Transaction_Contract{
			{
				Type: core.Transaction_Contract_TriggerSmartContract,
			},
		},
		Expiration: expiration.UnixNano(),
	}
	return &core.Transaction{RawData: raw}
}

// atomicInc utility for polling tests
