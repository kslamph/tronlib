package client

import (
	"context"
	"net"
	"sync/atomic"
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

	// For polling behavior in receipt waiting
	pollCount int32
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
	c, err := NewClientWithDialer("bufnet", dialer, WithTimeout(timeout), WithPool(1, 2))
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
func buildTriggerSmartContractTx(owner []byte, contract []byte, data []byte, expiration time.Time) *core.Transaction {
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
func atomicInc(v *int32) int32 { return atomic.AddInt32(v, 1) }
