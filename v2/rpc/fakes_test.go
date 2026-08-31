package rpc

// Bufconn fake WalletServer — ported from v1 pkg/client/test_fakes_test.go and
// internal/testutil. This is Phase 2's hermetic-test foundation: Tasks 4-10
// build their tests on this fake instead of a live node.

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

const bufSize = 1024 * 1024

// testWalletServer is a minimal fake implementing api.WalletServer for unit tests.
type testWalletServer struct {
	api.UnimplementedWalletServer

	// Handlers can be set per test to customize behavior.
	BroadcastHandler     func(ctx context.Context, in *core.Transaction) (*api.Return, error)
	GetNowBlockHandler   func(ctx context.Context, in *api.EmptyMessage) (*core.Block, error)
	GetBlockByNumHandler func(ctx context.Context, in *api.NumberMessage) (*core.Block, error)
	// TriggerConstantContractFunc customizes the constant-contract call used by
	// contract/token tests (Task 6-8).
	TriggerConstantContractFunc func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	GetTxInfoByIdHandler        func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)
}

func (s *testWalletServer) BroadcastTransaction(ctx context.Context, in *core.Transaction) (*api.Return, error) {
	if s.BroadcastHandler != nil {
		return s.BroadcastHandler(ctx, in)
	}
	return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
}

func (s *testWalletServer) GetNowBlock(ctx context.Context, in *api.EmptyMessage) (*core.Block, error) {
	if s.GetNowBlockHandler != nil {
		return s.GetNowBlockHandler(ctx, in)
	}
	return &core.Block{}, nil
}

func (s *testWalletServer) GetBlockByNum(ctx context.Context, in *api.NumberMessage) (*core.Block, error) {
	if s.GetBlockByNumHandler != nil {
		return s.GetBlockByNumHandler(ctx, in)
	}
	return &core.Block{}, nil
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
	// Default: empty info (an unconfirmed transaction), never a Go nil.
	return &core.TransactionInfo{}, nil
}

// testWalletSolidityServer serves the WalletSolidity service by delegating to
// the same *testWalletServer handlers, so both services share canned behavior.
type testWalletSolidityServer struct {
	api.UnimplementedWalletSolidityServer
	ws *testWalletServer
}

func (s *testWalletSolidityServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	return s.ws.GetTransactionInfoById(ctx, in)
}

func (s *testWalletSolidityServer) GetNowBlock(ctx context.Context, in *api.EmptyMessage) (*core.Block, error) {
	return s.ws.GetNowBlock(ctx, in)
}

func (s *testWalletSolidityServer) GetBlockByNum(ctx context.Context, in *api.NumberMessage) (*core.Block, error) {
	return s.ws.GetBlockByNum(ctx, in)
}

func (s *testWalletSolidityServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return s.ws.TriggerConstantContract(ctx, in)
}

// newBufconnServer spins up a bufconn-backed gRPC server serving impl and
// registers automatic teardown. When impl is a *testWalletServer, the
// WalletSolidity service is registered on the same server, backed by the same
// handlers (Solidity-side wrappers and WaitForSolid tests hit this).
func newBufconnServer(t *testing.T, impl api.WalletServer) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, impl)
	if ws, ok := impl.(*testWalletServer); ok {
		api.RegisterWalletSolidityServer(srv, &testWalletSolidityServer{ws: ws})
	}
	go func() { _ = srv.Serve(lis) }()

	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})
	return lis
}

// newBufconnClient creates a *Client whose pool dials the bufconn listener via
// the test-only constructor (bypassing scheme validation for passthrough:///
// addresses). Cleanup closes the client after the test.
func newBufconnClient(t *testing.T, impl api.WalletServer, timeout time.Duration) *Client {
	t.Helper()
	lis := newBufconnServer(t, impl)
	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		// DialContext honors context cancellation/timeouts.
		return lis.DialContext(ctx)
	}
	c, err := NewClientWithDialer("passthrough:///bufnet", dialer, WithTimeout(timeout), WithPool(1, 2))
	if err != nil {
		t.Fatalf("NewClientWithDialer error: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}
