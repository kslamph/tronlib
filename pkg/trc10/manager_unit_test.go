package trc10_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/trc10"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// ---- bufconn test infrastructure ----

const trc10BufSize = 1024 * 1024

type trc10FakeWalletServer struct {
	api.UnimplementedWalletServer

	CreateAssetIssue2Func             func(ctx context.Context, in *core.AssetIssueContract) (*api.TransactionExtention, error)
	UpdateAsset2Func                  func(ctx context.Context, in *core.UpdateAssetContract) (*api.TransactionExtention, error)
	TransferAsset2Func                func(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error)
	ParticipateAssetIssue2Func        func(ctx context.Context, in *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error)
	UnfreezeAsset2Func                func(ctx context.Context, in *core.UnfreezeAssetContract) (*api.TransactionExtention, error)
	GetAssetIssueByAccountFunc        func(ctx context.Context, in *core.Account) (*api.AssetIssueList, error)
	GetAssetIssueByNameFunc           func(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error)
	GetAssetIssueListByNameFunc       func(ctx context.Context, in *api.BytesMessage) (*api.AssetIssueList, error)
	GetAssetIssueByIdFunc             func(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error)
	GetAssetIssueListFunc             func(ctx context.Context, in *api.EmptyMessage) (*api.AssetIssueList, error)
	GetPaginatedAssetIssueListFunc    func(ctx context.Context, in *api.PaginatedMessage) (*api.AssetIssueList, error)
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
	lis := bufconn.Listen(trc10BufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.Dial("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
	}

	mgr := trc10.NewManager(&trc10MockConnProvider{conn: conn})
	cleanup := func() {
		conn.Close()
		srv.Stop()
	}
	return mgr, cleanup
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

func TestTRC10_CreateAssetIssue2_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("rpc error: server busy")
	fake := &trc10FakeWalletServer{
		CreateAssetIssue2Func: func(_ context.Context, _ *core.AssetIssueContract) (*api.TransactionExtention, error) {
			return nil, serverErr
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
		t.Fatal("expected error from server")
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

func TestTRC10_UpdateAsset2_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("update failed")
	fake := &trc10FakeWalletServer{
		UpdateAsset2Func: func(_ context.Context, _ *core.UpdateAssetContract) (*api.TransactionExtention, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	_, err := mgr.UpdateAsset2(context.Background(), owner, "desc", "url", 1000, 2000)
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_TransferAsset2_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("transfer failed")
	fake := &trc10FakeWalletServer{
		TransferAsset2Func: func(_ context.Context, _ *core.TransferAssetContract) (*api.TransactionExtention, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	to := mustTRC10Addr(t, "TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := mgr.TransferAsset2(context.Background(), owner, to, "TestAsset", 100)
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_ParticipateAssetIssue2_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("participate failed")
	fake := &trc10FakeWalletServer{
		ParticipateAssetIssue2Func: func(_ context.Context, _ *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	to := mustTRC10Addr(t, "TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := mgr.ParticipateAssetIssue2(context.Background(), owner, to, "TestAsset", 1000)
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_UnfreezeAsset2_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("unfreeze failed")
	fake := &trc10FakeWalletServer{
		UnfreezeAsset2Func: func(_ context.Context, _ *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	_, err := mgr.UnfreezeAsset2(context.Background(), owner)
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_GetAssetIssueByAccount_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("query failed")
	fake := &trc10FakeWalletServer{
		GetAssetIssueByAccountFunc: func(_ context.Context, _ *core.Account) (*api.AssetIssueList, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	addr := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	_, err := mgr.GetAssetIssueByAccount(context.Background(), addr)
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_GetAssetIssueByName_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("not found")
	fake := &trc10FakeWalletServer{
		GetAssetIssueByNameFunc: func(_ context.Context, _ *api.BytesMessage) (*core.AssetIssueContract, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	_, err := mgr.GetAssetIssueByName(context.Background(), "TestAsset")
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_GetAssetIssueListByName_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("list query failed")
	fake := &trc10FakeWalletServer{
		GetAssetIssueListByNameFunc: func(_ context.Context, _ *api.BytesMessage) (*api.AssetIssueList, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	_, err := mgr.GetAssetIssueListByName(context.Background(), "TestAsset")
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_GetAssetIssueById_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("asset not found")
	fake := &trc10FakeWalletServer{
		GetAssetIssueByIdFunc: func(_ context.Context, _ *api.BytesMessage) (*core.AssetIssueContract, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	_, err := mgr.GetAssetIssueById(context.Background(), []byte("1000001"))
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_GetAssetIssueList_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("list failed")
	fake := &trc10FakeWalletServer{
		GetAssetIssueListFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.AssetIssueList, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	_, err := mgr.GetAssetIssueList(context.Background())
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_GetPaginatedAssetIssueList_ServerError(t *testing.T) {
	serverErr := fmt.Errorf("pagination failed")
	fake := &trc10FakeWalletServer{
		GetPaginatedAssetIssueListFunc: func(_ context.Context, _ *api.PaginatedMessage) (*api.AssetIssueList, error) {
			return nil, serverErr
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	_, err := mgr.GetPaginatedAssetIssueList(context.Background(), 0, 10)
	if err == nil {
		t.Fatal("expected server error")
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

func TestTRC10_CreateAssetIssue2_NilResultNoError(t *testing.T) {
	// Server returns nil TransactionExtention with no error - ValidateTransactionResult will catch
	fake := &trc10FakeWalletServer{
		CreateAssetIssue2Func: func(_ context.Context, _ *core.AssetIssueContract) (*api.TransactionExtention, error) {
			return nil, nil
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
		t.Fatal("expected error for nil result")
	}
}

func TestTRC10_UpdateAsset2_NilResultNoError(t *testing.T) {
	fake := &trc10FakeWalletServer{
		UpdateAsset2Func: func(_ context.Context, _ *core.UpdateAssetContract) (*api.TransactionExtention, error) {
			return nil, nil
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	_, err := mgr.UpdateAsset2(context.Background(), owner, "desc", "url", 1000, 2000)
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestTRC10_TransferAsset2_NilResultNoError(t *testing.T) {
	fake := &trc10FakeWalletServer{
		TransferAsset2Func: func(_ context.Context, _ *core.TransferAssetContract) (*api.TransactionExtention, error) {
			return nil, nil
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	to := mustTRC10Addr(t, "TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := mgr.TransferAsset2(context.Background(), owner, to, "TestAsset", 100)
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestTRC10_ParticipateAssetIssue2_NilResultNoError(t *testing.T) {
	fake := &trc10FakeWalletServer{
		ParticipateAssetIssue2Func: func(_ context.Context, _ *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
			return nil, nil
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	to := mustTRC10Addr(t, "TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := mgr.ParticipateAssetIssue2(context.Background(), owner, to, "TestAsset", 1000)
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestTRC10_UnfreezeAsset2_NilResultNoError(t *testing.T) {
	fake := &trc10FakeWalletServer{
		UnfreezeAsset2Func: func(_ context.Context, _ *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
			return nil, nil
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	_, err := mgr.UnfreezeAsset2(context.Background(), owner)
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestTRC10_NilResultValidation_TransactionWithNilResultField(t *testing.T) {
	fake := &trc10FakeWalletServer{
		CreateAssetIssue2Func: func(_ context.Context, _ *core.AssetIssueContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: nil}, nil
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
		t.Fatal("expected error for nil Result field")
	}
}

func TestTRC10_UpdateAsset2_NilResultField(t *testing.T) {
	fake := &trc10FakeWalletServer{
		UpdateAsset2Func: func(_ context.Context, _ *core.UpdateAssetContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: nil}, nil
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	_, err := mgr.UpdateAsset2(context.Background(), owner, "desc", "url", 1000, 2000)
	if err == nil {
		t.Fatal("expected error for nil Result field")
	}
}

func TestTRC10_TransferAsset2_NilResultField(t *testing.T) {
	fake := &trc10FakeWalletServer{
		TransferAsset2Func: func(_ context.Context, _ *core.TransferAssetContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: nil}, nil
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	to := mustTRC10Addr(t, "TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := mgr.TransferAsset2(context.Background(), owner, to, "TestAsset", 100)
	if err == nil {
		t.Fatal("expected error for nil Result field")
	}
}

func TestTRC10_ParticipateAssetIssue2_NilResultField(t *testing.T) {
	fake := &trc10FakeWalletServer{
		ParticipateAssetIssue2Func: func(_ context.Context, _ *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: nil}, nil
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	to := mustTRC10Addr(t, "TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := mgr.ParticipateAssetIssue2(context.Background(), owner, to, "TestAsset", 1000)
	if err == nil {
		t.Fatal("expected error for nil Result field")
	}
}

func TestTRC10_UnfreezeAsset2_NilResultField(t *testing.T) {
	fake := &trc10FakeWalletServer{
		UnfreezeAsset2Func: func(_ context.Context, _ *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: nil}, nil
		},
	}
	mgr, cleanup := setupTRC10TestServer(t, fake)
	defer cleanup()

	owner := mustTRC10Addr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	_, err := mgr.UnfreezeAsset2(context.Background(), owner)
	if err == nil {
		t.Fatal("expected error for nil Result field")
	}
}
