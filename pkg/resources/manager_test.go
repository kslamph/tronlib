package resources

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// fakeWalletServer implements api.WalletServer for resource tests
type fakeWalletServer struct {
	api.UnimplementedWalletServer

	FreezeBalanceV2Func                    func(ctx context.Context, in *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error)
	UnfreezeBalanceV2Func                  func(ctx context.Context, in *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error)
	DelegateResourceFunc                   func(ctx context.Context, in *core.DelegateResourceContract) (*api.TransactionExtention, error)
	UnDelegateResourceFunc                 func(ctx context.Context, in *core.UnDelegateResourceContract) (*api.TransactionExtention, error)
	CancelAllUnfreezeV2Func                func(ctx context.Context, in *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error)
	WithdrawExpireUnfreezeFunc             func(ctx context.Context, in *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error)
	GetDelegatedResourceV2Func             func(ctx context.Context, in *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
	GetDelegatedResourceAccountIndexV2Func func(ctx context.Context, in *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error)
	GetCanDelegatedMaxSizeFunc             func(ctx context.Context, in *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error)
	GetAvailableUnfreezeCountFunc          func(ctx context.Context, in *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error)
	GetCanWithdrawUnfreezeAmountFunc       func(ctx context.Context, in *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error)
}

func (s *fakeWalletServer) FreezeBalanceV2(ctx context.Context, in *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	if s.FreezeBalanceV2Func != nil {
		return s.FreezeBalanceV2Func(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeWalletServer) UnfreezeBalanceV2(ctx context.Context, in *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	if s.UnfreezeBalanceV2Func != nil {
		return s.UnfreezeBalanceV2Func(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeWalletServer) DelegateResource(ctx context.Context, in *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	if s.DelegateResourceFunc != nil {
		return s.DelegateResourceFunc(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeWalletServer) UnDelegateResource(ctx context.Context, in *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	if s.UnDelegateResourceFunc != nil {
		return s.UnDelegateResourceFunc(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeWalletServer) CancelAllUnfreezeV2(ctx context.Context, in *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
	if s.CancelAllUnfreezeV2Func != nil {
		return s.CancelAllUnfreezeV2Func(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeWalletServer) WithdrawExpireUnfreeze(ctx context.Context, in *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
	if s.WithdrawExpireUnfreezeFunc != nil {
		return s.WithdrawExpireUnfreezeFunc(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeWalletServer) GetDelegatedResourceV2(ctx context.Context, in *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	if s.GetDelegatedResourceV2Func != nil {
		return s.GetDelegatedResourceV2Func(ctx, in)
	}
	return &api.DelegatedResourceList{}, nil
}

func (s *fakeWalletServer) GetDelegatedResourceAccountIndexV2(ctx context.Context, in *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
	if s.GetDelegatedResourceAccountIndexV2Func != nil {
		return s.GetDelegatedResourceAccountIndexV2Func(ctx, in)
	}
	return &core.DelegatedResourceAccountIndex{}, nil
}

func (s *fakeWalletServer) GetCanDelegatedMaxSize(ctx context.Context, in *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	if s.GetCanDelegatedMaxSizeFunc != nil {
		return s.GetCanDelegatedMaxSizeFunc(ctx, in)
	}
	return &api.CanDelegatedMaxSizeResponseMessage{}, nil
}

func (s *fakeWalletServer) GetAvailableUnfreezeCount(ctx context.Context, in *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	if s.GetAvailableUnfreezeCountFunc != nil {
		return s.GetAvailableUnfreezeCountFunc(ctx, in)
	}
	return &api.GetAvailableUnfreezeCountResponseMessage{}, nil
}

func (s *fakeWalletServer) GetCanWithdrawUnfreezeAmount(ctx context.Context, in *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	if s.GetCanWithdrawUnfreezeAmountFunc != nil {
		return s.GetCanWithdrawUnfreezeAmountFunc(ctx, in)
	}
	return &api.CanWithdrawUnfreezeAmountResponseMessage{}, nil
}

// setupTestServer creates a bufconn gRPC server and returns a ResourcesManager connected to it.
func setupTestServer(t *testing.T, fake *fakeWalletServer) (*ResourcesManager, func()) {
	t.Helper()
	lis := bufconn.Listen(bufSize)
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

	cp := &mockConnProvider{conn: conn}
	mgr := NewManager(cp)

	cleanup := func() {
		conn.Close()
		srv.Stop()
		lis.Close()
	}
	return mgr, cleanup
}

type mockConnProvider struct {
	conn *grpc.ClientConn
}

func (m *mockConnProvider) GetConnection(_ context.Context) (*grpc.ClientConn, error) {
	return m.conn, nil
}

func (m *mockConnProvider) ReturnConnection(_ *grpc.ClientConn) {}

func (m *mockConnProvider) GetTimeout() time.Duration {
	return 30 * time.Second
}

func mustAddr(s string) *types.Address {
	addr, err := types.NewAddressFromBase58(s)
	if err != nil {
		panic(err)
	}
	return addr
}

var (
	testAddr  = mustAddr("TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY")
	testAddr2 = mustAddr("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
)

func TestNewManager(t *testing.T) {
	mgr := NewManager(&mockConnProvider{})
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestFreezeBalanceV2(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.FreezeBalanceV2(ctx, testAddr, 1_000_000, ResourceTypeEnergy)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.FreezeBalanceV2(ctx, nil, 1_000_000, ResourceTypeEnergy)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("zero balance", func(t *testing.T) {
		_, err := mgr.FreezeBalanceV2(ctx, testAddr, 0, ResourceTypeEnergy)
		if err == nil {
			t.Fatal("expected error for zero balance")
		}
	})

	t.Run("negative balance", func(t *testing.T) {
		_, err := mgr.FreezeBalanceV2(ctx, testAddr, -100, ResourceTypeEnergy)
		if err == nil {
			t.Fatal("expected error for negative balance")
		}
	})
}

func TestUnfreezeBalanceV2(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.UnfreezeBalanceV2(ctx, testAddr, 500_000, ResourceTypeBandwidth)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.UnfreezeBalanceV2(ctx, nil, 500_000, ResourceTypeBandwidth)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("zero balance", func(t *testing.T) {
		_, err := mgr.UnfreezeBalanceV2(ctx, testAddr, 0, ResourceTypeBandwidth)
		if err == nil {
			t.Fatal("expected error for zero balance")
		}
	})
}

func TestDelegateResource(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.DelegateResource(ctx, testAddr, testAddr2, 1_000_000, ResourceTypeEnergy, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.DelegateResource(ctx, nil, testAddr2, 1_000_000, ResourceTypeEnergy, false)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("nil receiver", func(t *testing.T) {
		_, err := mgr.DelegateResource(ctx, testAddr, nil, 1_000_000, ResourceTypeEnergy, false)
		if err == nil {
			t.Fatal("expected error for nil receiver")
		}
	})

	t.Run("same owner and receiver", func(t *testing.T) {
		_, err := mgr.DelegateResource(ctx, testAddr, testAddr, 1_000_000, ResourceTypeEnergy, false)
		if err == nil {
			t.Fatal("expected error for same addresses")
		}
	})

	t.Run("zero balance", func(t *testing.T) {
		_, err := mgr.DelegateResource(ctx, testAddr, testAddr2, 0, ResourceTypeEnergy, false)
		if err == nil {
			t.Fatal("expected error for zero balance")
		}
	})
}

func TestUnDelegateResource(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.UnDelegateResource(ctx, testAddr, testAddr2, 500_000, ResourceTypeEnergy)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.UnDelegateResource(ctx, nil, testAddr2, 500_000, ResourceTypeEnergy)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("nil receiver", func(t *testing.T) {
		_, err := mgr.UnDelegateResource(ctx, testAddr, nil, 500_000, ResourceTypeEnergy)
		if err == nil {
			t.Fatal("expected error for nil receiver")
		}
	})

	t.Run("zero balance", func(t *testing.T) {
		_, err := mgr.UnDelegateResource(ctx, testAddr, testAddr2, 0, ResourceTypeEnergy)
		if err == nil {
			t.Fatal("expected error for zero balance")
		}
	})
}

func TestCancelAllUnfreezeV2(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.CancelAllUnfreezeV2(ctx, testAddr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.CancelAllUnfreezeV2(ctx, nil)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})
}

func TestWithdrawExpireUnfreeze(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.WithdrawExpireUnfreeze(ctx, testAddr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.WithdrawExpireUnfreeze(ctx, nil)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})
}

func TestGetDelegatedResourceV2(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetDelegatedResourceV2(ctx, testAddr, testAddr2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil from", func(t *testing.T) {
		_, err := mgr.GetDelegatedResourceV2(ctx, nil, testAddr2)
		if err == nil {
			t.Fatal("expected error for nil from address")
		}
	})

	t.Run("nil to", func(t *testing.T) {
		_, err := mgr.GetDelegatedResourceV2(ctx, testAddr, nil)
		if err == nil {
			t.Fatal("expected error for nil to address")
		}
	})
}

func TestGetDelegatedResourceAccountIndexV2(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetDelegatedResourceAccountIndexV2(ctx, testAddr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil address", func(t *testing.T) {
		_, err := mgr.GetDelegatedResourceAccountIndexV2(ctx, nil)
		if err == nil {
			t.Fatal("expected error for nil address")
		}
	})
}

func TestGetCanDelegatedMaxSize(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetCanDelegatedMaxSize(ctx, testAddr, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil address", func(t *testing.T) {
		_, err := mgr.GetCanDelegatedMaxSize(ctx, nil, 0)
		if err == nil {
			t.Fatal("expected error for nil address")
		}
	})
}

func TestGetAvailableUnfreezeCount(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetAvailableUnfreezeCount(ctx, testAddr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil address", func(t *testing.T) {
		_, err := mgr.GetAvailableUnfreezeCount(ctx, nil)
		if err == nil {
			t.Fatal("expected error for nil address")
		}
	})
}

func TestGetCanWithdrawUnfreezeAmount(t *testing.T) {
	fake := &fakeWalletServer{}
	mgr, cleanup := setupTestServer(t, fake)
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetCanWithdrawUnfreezeAmount(ctx, testAddr, time.Now().UnixMilli())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil address", func(t *testing.T) {
		_, err := mgr.GetCanWithdrawUnfreezeAmount(ctx, nil, 0)
		if err == nil {
			t.Fatal("expected error for nil address")
		}
	})
}
