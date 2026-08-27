package trc10_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/internal/testutil"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/trc10"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- bufconn test infrastructure ----

type trc10FakeWalletServer struct {
	api.UnimplementedWalletServer

	CreateAssetIssue2Func          func(ctx context.Context, in *core.AssetIssueContract) (*api.TransactionExtention, error)
	UpdateAsset2Func               func(ctx context.Context, in *core.UpdateAssetContract) (*api.TransactionExtention, error)
	TransferAsset2Func             func(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error)
	ParticipateAssetIssue2Func     func(ctx context.Context, in *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error)
	UnfreezeAsset2Func             func(ctx context.Context, in *core.UnfreezeAssetContract) (*api.TransactionExtention, error)
	GetAssetIssueByAccountFunc     func(ctx context.Context, in *core.Account) (*api.AssetIssueList, error)
	GetAssetIssueByNameFunc        func(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error)
	GetAssetIssueListByNameFunc    func(ctx context.Context, in *api.BytesMessage) (*api.AssetIssueList, error)
	GetAssetIssueByIdFunc          func(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error)
	GetAssetIssueListFunc          func(ctx context.Context, in *api.EmptyMessage) (*api.AssetIssueList, error)
	GetPaginatedAssetIssueListFunc func(ctx context.Context, in *api.PaginatedMessage) (*api.AssetIssueList, error)
}

func (s *trc10FakeWalletServer) CreateAssetIssue2(ctx context.Context, in *core.AssetIssueContract) (*api.TransactionExtention, error) {
	if s.CreateAssetIssue2Func != nil {
		return s.CreateAssetIssue2Func(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *trc10FakeWalletServer) UpdateAsset2(ctx context.Context, in *core.UpdateAssetContract) (*api.TransactionExtention, error) {
	if s.UpdateAsset2Func != nil {
		return s.UpdateAsset2Func(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *trc10FakeWalletServer) TransferAsset2(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error) {
	if s.TransferAsset2Func != nil {
		return s.TransferAsset2Func(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *trc10FakeWalletServer) ParticipateAssetIssue2(ctx context.Context, in *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
	if s.ParticipateAssetIssue2Func != nil {
		return s.ParticipateAssetIssue2Func(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *trc10FakeWalletServer) UnfreezeAsset2(ctx context.Context, in *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
	if s.UnfreezeAsset2Func != nil {
		return s.UnfreezeAsset2Func(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *trc10FakeWalletServer) GetAssetIssueByAccount(ctx context.Context, in *core.Account) (*api.AssetIssueList, error) {
	if s.GetAssetIssueByAccountFunc != nil {
		return s.GetAssetIssueByAccountFunc(ctx, in)
	}
	return &api.AssetIssueList{}, nil
}

func (s *trc10FakeWalletServer) GetAssetIssueByName(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error) {
	if s.GetAssetIssueByNameFunc != nil {
		return s.GetAssetIssueByNameFunc(ctx, in)
	}
	return &core.AssetIssueContract{Name: in.Value}, nil
}

func (s *trc10FakeWalletServer) GetAssetIssueListByName(ctx context.Context, in *api.BytesMessage) (*api.AssetIssueList, error) {
	if s.GetAssetIssueListByNameFunc != nil {
		return s.GetAssetIssueListByNameFunc(ctx, in)
	}
	return &api.AssetIssueList{}, nil
}

func (s *trc10FakeWalletServer) GetAssetIssueById(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error) {
	if s.GetAssetIssueByIdFunc != nil {
		return s.GetAssetIssueByIdFunc(ctx, in)
	}
	return &core.AssetIssueContract{}, nil
}

func (s *trc10FakeWalletServer) GetAssetIssueList(ctx context.Context, in *api.EmptyMessage) (*api.AssetIssueList, error) {
	if s.GetAssetIssueListFunc != nil {
		return s.GetAssetIssueListFunc(ctx, in)
	}
	return &api.AssetIssueList{}, nil
}

func (s *trc10FakeWalletServer) GetPaginatedAssetIssueList(ctx context.Context, in *api.PaginatedMessage) (*api.AssetIssueList, error) {
	if s.GetPaginatedAssetIssueListFunc != nil {
		return s.GetPaginatedAssetIssueListFunc(ctx, in)
	}
	return &api.AssetIssueList{}, nil
}

type trc10MockConnProvider struct {
	conn *grpc.ClientConn
}

func (m *trc10MockConnProvider) GetConnection(_ context.Context) (*grpc.ClientConn, error) {
	return m.conn, nil
}
func (m *trc10MockConnProvider) ReturnConnection(_ *grpc.ClientConn) {}
func (m *trc10MockConnProvider) GetTimeout() time.Duration           { return 5 * time.Second }

func setupTRC10TestServer(t *testing.T, fake api.WalletServer) (*trc10.TRC10Manager, func()) {
	t.Helper()
	lis := testutil.NewBufconnServer(t, fake)
	conn := testutil.DialBufconn(t, lis)
	mgr := trc10.NewManager(&trc10MockConnProvider{conn: conn})
	return mgr, func() {}
}

func mustTRC10Addr(t *testing.T, s string) *types.Address {
	t.Helper()
	a, err := types.NewAddress(s)
	if err != nil {
		t.Fatalf("NewAddress(%q): %v", s, err)
	}
	return a
}

// ---- Success path tests ----

func TestTRC10_CreateAssetIssue2_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	result, err := mgr.CreateAssetIssue2(context.Background(), owner,
		"MyToken", "MTK", 1000000, 100, 100,
		1640995200000, 1640995300000, "A test token", "https://example.com",
		1000, 1000, nil)
	if err != nil {
		t.Fatalf("CreateAssetIssue2: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_CreateAssetIssue2_WithFrozenSupply(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	frozenSupply := []trc10.FrozenSupply{{FrozenAmount: 1000, FrozenDays: 30}}
	result, err := mgr.CreateAssetIssue2(context.Background(), owner,
		"MyToken", "MTK", 1000000, 100, 100,
		1640995200000, 1640995300000, "A test token", "https://example.com",
		1000, 1000, frozenSupply)
	if err != nil {
		t.Fatalf("CreateAssetIssue2 with frozen supply: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_CreateAssetIssue2_TransactionErrorResult(t *testing.T) {
	fake := &trc10FakeWalletServer{
		CreateAssetIssue2Func: func(_ context.Context, _ *core.AssetIssueContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{Result: false, Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("contract error")},
			}, nil
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	_, err := mgr.CreateAssetIssue2(context.Background(), owner,
		"MyToken", "MTK", 1000000, 100, 100,
		1640995200000, 1640995300000, "A test token", "https://example.com",
		1000, 1000, nil)
	if err == nil {
		t.Fatal("expected error from failed transaction result")
	}
}

func TestTRC10_UpdateAsset2_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	result, err := mgr.UpdateAsset2(context.Background(), owner, "desc", "url", 1000, 2000)
	if err != nil {
		t.Fatalf("UpdateAsset2: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_TransferAsset2_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	to := mustTRC10Addr(t, "TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	result, err := mgr.TransferAsset2(context.Background(), owner, to, "TestAsset", 100)
	if err != nil {
		t.Fatalf("TransferAsset2: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_ParticipateAssetIssue2_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	to := mustTRC10Addr(t, "TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	result, err := mgr.ParticipateAssetIssue2(context.Background(), owner, to, "TestAsset", 1000)
	if err != nil {
		t.Fatalf("ParticipateAssetIssue2: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_UnfreezeAsset2_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	result, err := mgr.UnfreezeAsset2(context.Background(), owner)
	if err != nil {
		t.Fatalf("UnfreezeAsset2: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_GetAssetIssueByAccount_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	addr := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	result, err := mgr.GetAssetIssueByAccount(context.Background(), addr)
	if err != nil {
		t.Fatalf("GetAssetIssueByAccount: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_GetAssetIssueByName_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	result, err := mgr.GetAssetIssueByName(context.Background(), "TestAsset")
	if err != nil {
		t.Fatalf("GetAssetIssueByName: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_GetAssetIssueListByName_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	result, err := mgr.GetAssetIssueListByName(context.Background(), "TestAsset")
	if err != nil {
		t.Fatalf("GetAssetIssueListByName: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_GetAssetIssueById_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	result, err := mgr.GetAssetIssueById(context.Background(), []byte("1000001"))
	if err != nil {
		t.Fatalf("GetAssetIssueById: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_GetAssetIssueList_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	result, err := mgr.GetAssetIssueList(context.Background())
	if err != nil {
		t.Fatalf("GetAssetIssueList: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_GetPaginatedAssetIssueList_Success(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	result, err := mgr.GetPaginatedAssetIssueList(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("GetPaginatedAssetIssueList: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTRC10_GetPaginatedAssetIssueList_MaxLimit(t *testing.T) {
	fake := &trc10FakeWalletServer{}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	// limit == 100 should succeed (boundary)
	result, err := mgr.GetPaginatedAssetIssueList(context.Background(), 0, 100)
	if err != nil {
		t.Fatalf("GetPaginatedAssetIssueList with limit=100: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// trc10RPCCase drives one manager RPC against a fake wallet server, so one
// table exercises both failure modes across every method:
//   - "server error": the handler returns a gRPC status error
//   - "empty response": the handler returns a zero-value TransactionExtention,
//     which is exactly what a client observes when a server returns a nil
//     extention or one without a Result (the former NilResult tests covered
//     this same wire shape twice)
type trc10RPCCase struct {
	name string
	// serverErr wires this RPC's handler to return a transport error.
	serverErr func() *trc10FakeWalletServer
	// emptyResp wires this RPC's handler to return a zero-value
	// TransactionExtention. nil means an empty response is legitimate for
	// this RPC (read-only getters return typed lists, not extenders).
	emptyResp func() *trc10FakeWalletServer
	call      func(m *trc10.TRC10Manager) error
}

func buildTRC10RPCCases(t *testing.T) []trc10RPCCase {
	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	to := mustTRC10Addr(t, "TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	ctx := context.Background()

	return []trc10RPCCase{
		{
			name: "CreateAssetIssue2",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{CreateAssetIssue2Func: func(_ context.Context, _ *core.AssetIssueContract) (*api.TransactionExtention, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			emptyResp: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{CreateAssetIssue2Func: func(_ context.Context, _ *core.AssetIssueContract) (*api.TransactionExtention, error) {
					return &api.TransactionExtention{}, nil
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.CreateAssetIssue2(ctx, owner,
					"MyToken", "MTK", 1000000, 100, 100,
					1640995200000, 1640995300000, "A test token", "https://example.com",
					1000, 1000, nil)
				return err
			},
		},
		{
			name: "UpdateAsset2",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{UpdateAsset2Func: func(_ context.Context, _ *core.UpdateAssetContract) (*api.TransactionExtention, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			emptyResp: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{UpdateAsset2Func: func(_ context.Context, _ *core.UpdateAssetContract) (*api.TransactionExtention, error) {
					return &api.TransactionExtention{}, nil
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.UpdateAsset2(ctx, owner, "desc", "url", 1000, 2000)
				return err
			},
		},
		{
			name: "TransferAsset2",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{TransferAsset2Func: func(_ context.Context, _ *core.TransferAssetContract) (*api.TransactionExtention, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			emptyResp: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{TransferAsset2Func: func(_ context.Context, _ *core.TransferAssetContract) (*api.TransactionExtention, error) {
					return &api.TransactionExtention{}, nil
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.TransferAsset2(ctx, owner, to, "TestAsset", 100)
				return err
			},
		},
		{
			name: "ParticipateAssetIssue2",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{ParticipateAssetIssue2Func: func(_ context.Context, _ *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			emptyResp: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{ParticipateAssetIssue2Func: func(_ context.Context, _ *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
					return &api.TransactionExtention{}, nil
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.ParticipateAssetIssue2(ctx, owner, to, "TestAsset", 1000)
				return err
			},
		},
		{
			name: "UnfreezeAsset2",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{UnfreezeAsset2Func: func(_ context.Context, _ *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			emptyResp: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{UnfreezeAsset2Func: func(_ context.Context, _ *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
					return &api.TransactionExtention{}, nil
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.UnfreezeAsset2(ctx, owner)
				return err
			},
		},
		{
			name: "GetAssetIssueByAccount",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{GetAssetIssueByAccountFunc: func(_ context.Context, _ *core.Account) (*api.AssetIssueList, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.GetAssetIssueByAccount(ctx, owner)
				return err
			},
		},
		{
			name: "GetAssetIssueByName",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{GetAssetIssueByNameFunc: func(_ context.Context, _ *api.BytesMessage) (*core.AssetIssueContract, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.GetAssetIssueByName(ctx, "TestAsset")
				return err
			},
		},
		{
			name: "GetAssetIssueListByName",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{GetAssetIssueListByNameFunc: func(_ context.Context, _ *api.BytesMessage) (*api.AssetIssueList, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.GetAssetIssueListByName(ctx, "TestAsset")
				return err
			},
		},
		{
			name: "GetAssetIssueById",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{GetAssetIssueByIdFunc: func(_ context.Context, _ *api.BytesMessage) (*core.AssetIssueContract, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.GetAssetIssueById(ctx, []byte("1000001"))
				return err
			},
		},
		{
			name: "GetAssetIssueList",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{GetAssetIssueListFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.AssetIssueList, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.GetAssetIssueList(ctx)
				return err
			},
		},
		{
			name: "GetPaginatedAssetIssueList",
			serverErr: func() *trc10FakeWalletServer {
				return &trc10FakeWalletServer{GetPaginatedAssetIssueListFunc: func(_ context.Context, _ *api.PaginatedMessage) (*api.AssetIssueList, error) {
					return nil, status.Error(codes.Unavailable, "node unavailable")
				}}
			},
			call: func(m *trc10.TRC10Manager) error {
				_, err := m.GetPaginatedAssetIssueList(ctx, 0, 10)
				return err
			},
		},
	}
}

// TestTRC10_RPC_FailureModes runs every RPC through both canned failure
// shapes and asserts the manager surfaces an error each time.
func TestTRC10_RPC_FailureModes(t *testing.T) {
	modes := []struct {
		name string
		fake func(c trc10RPCCase) *trc10FakeWalletServer
	}{
		{"server error", func(c trc10RPCCase) *trc10FakeWalletServer { return c.serverErr() }},
		{"empty response", func(c trc10RPCCase) *trc10FakeWalletServer {
			if c.emptyResp == nil {
				return nil // empty list is a legitimate response for getters
			}
			return c.emptyResp()
		}},
	}
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			for _, tc := range buildTRC10RPCCases(t) {
				t.Run(tc.name, func(t *testing.T) {
					fake := mode.fake(tc)
					if fake == nil {
						t.Skipf("empty response is legitimate for %s", tc.name)
					}
					mgr, cleanup := setupTRC10TestServer(t, fake)
					defer cleanup()
					if err := tc.call(mgr); err == nil {
						t.Fatalf("expected error from %s", tc.name)
					}
				})
			}
		})
	}
}
