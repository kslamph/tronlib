package network

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

type fakeWalletServer struct {
	api.UnimplementedWalletServer

	GetNodeInfoFunc                  func(ctx context.Context, in *api.EmptyMessage) (*core.NodeInfo, error)
	GetChainParametersFunc           func(ctx context.Context, in *api.EmptyMessage) (*core.ChainParameters, error)
	ListNodesFunc                    func(ctx context.Context, in *api.EmptyMessage) (*api.NodeList, error)
	GetBlockByNum2Func               func(ctx context.Context, in *api.NumberMessage) (*api.BlockExtention, error)
	GetTransactionInfoByBlockNumFunc func(ctx context.Context, in *api.NumberMessage) (*api.TransactionInfoList, error)
	GetBlockByIdFunc                 func(ctx context.Context, in *api.BytesMessage) (*core.Block, error)
	GetBlockByLimitNext2Func         func(ctx context.Context, in *api.BlockLimit) (*api.BlockListExtention, error)
	GetBlockByLatestNum2Func         func(ctx context.Context, in *api.NumberMessage) (*api.BlockListExtention, error)
	GetNowBlock2Func                 func(ctx context.Context, in *api.EmptyMessage) (*api.BlockExtention, error)
	GetTransactionInfoByIdFunc       func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)
	GetTransactionByIdFunc           func(ctx context.Context, in *api.BytesMessage) (*core.Transaction, error)
}

func (s *fakeWalletServer) GetNodeInfo(ctx context.Context, in *api.EmptyMessage) (*core.NodeInfo, error) {
	if s.GetNodeInfoFunc != nil {
		return s.GetNodeInfoFunc(ctx, in)
	}
	return &core.NodeInfo{}, nil
}

func (s *fakeWalletServer) GetChainParameters(ctx context.Context, in *api.EmptyMessage) (*core.ChainParameters, error) {
	if s.GetChainParametersFunc != nil {
		return s.GetChainParametersFunc(ctx, in)
	}
	return &core.ChainParameters{}, nil
}

func (s *fakeWalletServer) ListNodes(ctx context.Context, in *api.EmptyMessage) (*api.NodeList, error) {
	if s.ListNodesFunc != nil {
		return s.ListNodesFunc(ctx, in)
	}
	return &api.NodeList{}, nil
}

func (s *fakeWalletServer) GetBlockByNum2(ctx context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
	if s.GetBlockByNum2Func != nil {
		return s.GetBlockByNum2Func(ctx, in)
	}
	return &api.BlockExtention{}, nil
}

func (s *fakeWalletServer) GetTransactionInfoByBlockNum(ctx context.Context, in *api.NumberMessage) (*api.TransactionInfoList, error) {
	if s.GetTransactionInfoByBlockNumFunc != nil {
		return s.GetTransactionInfoByBlockNumFunc(ctx, in)
	}
	return &api.TransactionInfoList{}, nil
}

func (s *fakeWalletServer) GetBlockById(ctx context.Context, in *api.BytesMessage) (*core.Block, error) {
	if s.GetBlockByIdFunc != nil {
		return s.GetBlockByIdFunc(ctx, in)
	}
	return &core.Block{}, nil
}

func (s *fakeWalletServer) GetBlockByLimitNext2(ctx context.Context, in *api.BlockLimit) (*api.BlockListExtention, error) {
	if s.GetBlockByLimitNext2Func != nil {
		return s.GetBlockByLimitNext2Func(ctx, in)
	}
	return &api.BlockListExtention{}, nil
}

func (s *fakeWalletServer) GetBlockByLatestNum2(ctx context.Context, in *api.NumberMessage) (*api.BlockListExtention, error) {
	if s.GetBlockByLatestNum2Func != nil {
		return s.GetBlockByLatestNum2Func(ctx, in)
	}
	return &api.BlockListExtention{}, nil
}

func (s *fakeWalletServer) GetNowBlock2(ctx context.Context, in *api.EmptyMessage) (*api.BlockExtention, error) {
	if s.GetNowBlock2Func != nil {
		return s.GetNowBlock2Func(ctx, in)
	}
	return &api.BlockExtention{}, nil
}

func (s *fakeWalletServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	if s.GetTransactionInfoByIdFunc != nil {
		return s.GetTransactionInfoByIdFunc(ctx, in)
	}
	return &core.TransactionInfo{}, nil
}

func (s *fakeWalletServer) GetTransactionById(ctx context.Context, in *api.BytesMessage) (*core.Transaction, error) {
	if s.GetTransactionByIdFunc != nil {
		return s.GetTransactionByIdFunc(ctx, in)
	}
	return &core.Transaction{}, nil
}

type mockConnProvider struct {
	conn *grpc.ClientConn
}

func (m *mockConnProvider) GetConnection(_ context.Context) (*grpc.ClientConn, error) {
	return m.conn, nil
}
func (m *mockConnProvider) ReturnConnection(_ *grpc.ClientConn) {}
func (m *mockConnProvider) GetTimeout() time.Duration            { return 30 * time.Second }

func setupTestServer(t *testing.T, fake *fakeWalletServer) (*NetworkManager, func()) {
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

	mgr := NewManager(&mockConnProvider{conn: conn})
	cleanup := func() {
		conn.Close()
		srv.Stop()
		lis.Close()
	}
	return mgr, cleanup
}

// Valid 64-char hex tx ID for tests
const testTxID = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"

func TestNewManager(t *testing.T) {
	mgr := NewManager(&mockConnProvider{})
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestGetNodeInfo(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()

	result, err := mgr.GetNodeInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestGetChainParameters(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()

	result, err := mgr.GetChainParameters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestListNodes(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()

	result, err := mgr.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestGetBlockByNumber(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetBlockByNumber(ctx, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("negative block number", func(t *testing.T) {
		_, err := mgr.GetBlockByNumber(ctx, -1)
		if err == nil {
			t.Fatal("expected error for negative block number")
		}
	})
}

func TestGetTransactionInfoByBlockNum(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetTransactionInfoByBlockNum(ctx, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("negative block number", func(t *testing.T) {
		_, err := mgr.GetTransactionInfoByBlockNum(ctx, -1)
		if err == nil {
			t.Fatal("expected error for negative block number")
		}
	})
}

func TestGetBlockById(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetBlockById(ctx, []byte("someblockid123456789"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("empty block ID", func(t *testing.T) {
		_, err := mgr.GetBlockById(ctx, nil)
		if err == nil {
			t.Fatal("expected error for nil block ID")
		}
	})
}

func TestGetBlocksByLimit(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetBlocksByLimit(ctx, 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("negative start", func(t *testing.T) {
		_, err := mgr.GetBlocksByLimit(ctx, -1, 10)
		if err == nil {
			t.Fatal("expected error for negative start")
		}
	})

	t.Run("end less than start", func(t *testing.T) {
		_, err := mgr.GetBlocksByLimit(ctx, 10, 5)
		if err == nil {
			t.Fatal("expected error for end < start")
		}
	})

	t.Run("range too large", func(t *testing.T) {
		_, err := mgr.GetBlocksByLimit(ctx, 0, 200)
		if err == nil {
			t.Fatal("expected error for range > 100")
		}
	})

	t.Run("range exactly 100", func(t *testing.T) {
		_, err := mgr.GetBlocksByLimit(ctx, 0, 100)
		if err != nil {
			t.Fatalf("unexpected error for range=100: %v", err)
		}
	})
}

func TestGetLatestBlocks(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetLatestBlocks(ctx, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("zero count", func(t *testing.T) {
		_, err := mgr.GetLatestBlocks(ctx, 0)
		if err == nil {
			t.Fatal("expected error for zero count")
		}
	})

	t.Run("negative count", func(t *testing.T) {
		_, err := mgr.GetLatestBlocks(ctx, -1)
		if err == nil {
			t.Fatal("expected error for negative count")
		}
	})

	t.Run("count too large", func(t *testing.T) {
		_, err := mgr.GetLatestBlocks(ctx, 101)
		if err == nil {
			t.Fatal("expected error for count > 100")
		}
	})

	t.Run("count exactly 100", func(t *testing.T) {
		_, err := mgr.GetLatestBlocks(ctx, 100)
		if err != nil {
			t.Fatalf("unexpected error for count=100: %v", err)
		}
	})
}

func TestGetNowBlock(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()

	result, err := mgr.GetNowBlock(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestGetTransactionInfoById(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetTransactionInfoById(ctx, testTxID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("with 0x prefix", func(t *testing.T) {
		result, err := mgr.GetTransactionInfoById(ctx, "0x"+testTxID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("empty tx ID", func(t *testing.T) {
		_, err := mgr.GetTransactionInfoById(ctx, "")
		if err == nil {
			t.Fatal("expected error for empty tx ID")
		}
	})

	t.Run("wrong length", func(t *testing.T) {
		_, err := mgr.GetTransactionInfoById(ctx, "abc123")
		if err == nil {
			t.Fatal("expected error for short tx ID")
		}
	})

	t.Run("invalid hex", func(t *testing.T) {
		_, err := mgr.GetTransactionInfoById(ctx, "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
		if err == nil {
			t.Fatal("expected error for invalid hex")
		}
	})
}

func TestGetTransactionById(t *testing.T) {
	mgr, cleanup := setupTestServer(t, &fakeWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetTransactionById(ctx, testTxID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("with 0X prefix", func(t *testing.T) {
		result, err := mgr.GetTransactionById(ctx, "0X"+testTxID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("empty tx ID", func(t *testing.T) {
		_, err := mgr.GetTransactionById(ctx, "")
		if err == nil {
			t.Fatal("expected error for empty tx ID")
		}
	})

	t.Run("wrong length", func(t *testing.T) {
		_, err := mgr.GetTransactionById(ctx, "abc123")
		if err == nil {
			t.Fatal("expected error for short tx ID")
		}
	})
}
