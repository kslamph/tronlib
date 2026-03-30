package smartcontract

import (
	"context"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const scBufSize = 1024 * 1024

type fakeSCWalletServer struct {
	api.UnimplementedWalletServer

	TriggerConstantContractFunc func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	TriggerContractFunc         func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	EstimateEnergyFunc          func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error)
	GetContractFunc             func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error)
	GetContractInfoFunc         func(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error)
	DeployContractFunc          func(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error)
	UpdateSettingFunc           func(ctx context.Context, in *core.UpdateSettingContract) (*api.TransactionExtention, error)
	UpdateEnergyLimitFunc       func(ctx context.Context, in *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error)
	ClearContractABIFunc        func(ctx context.Context, in *core.ClearABIContract) (*api.TransactionExtention, error)
}

func (s *fakeSCWalletServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if s.TriggerConstantContractFunc != nil {
		return s.TriggerConstantContractFunc(ctx, in)
	}
	return &api.TransactionExtention{
		Result:         &api.Return{Result: true},
		EnergyUsed:     5000,
		ConstantResult: [][]byte{[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 42}},
	}, nil
}

func (s *fakeSCWalletServer) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if s.TriggerContractFunc != nil {
		return s.TriggerContractFunc(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeSCWalletServer) EstimateEnergy(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	if s.EstimateEnergyFunc != nil {
		return s.EstimateEnergyFunc(ctx, in)
	}
	return &api.EstimateEnergyMessage{}, nil
}

func (s *fakeSCWalletServer) GetContract(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
	if s.GetContractFunc != nil {
		return s.GetContractFunc(ctx, in)
	}
	return &core.SmartContract{}, nil
}

func (s *fakeSCWalletServer) GetContractInfo(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
	if s.GetContractInfoFunc != nil {
		return s.GetContractInfoFunc(ctx, in)
	}
	return &core.SmartContractDataWrapper{}, nil
}

func (s *fakeSCWalletServer) DeployContract(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error) {
	if s.DeployContractFunc != nil {
		return s.DeployContractFunc(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeSCWalletServer) UpdateSetting(ctx context.Context, in *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	if s.UpdateSettingFunc != nil {
		return s.UpdateSettingFunc(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeSCWalletServer) UpdateEnergyLimit(ctx context.Context, in *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	if s.UpdateEnergyLimitFunc != nil {
		return s.UpdateEnergyLimitFunc(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

func (s *fakeSCWalletServer) ClearContractABI(ctx context.Context, in *core.ClearABIContract) (*api.TransactionExtention, error) {
	if s.ClearContractABIFunc != nil {
		return s.ClearContractABIFunc(ctx, in)
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true}}, nil
}

type scMockConnProvider struct {
	conn *grpc.ClientConn
}

func (m *scMockConnProvider) GetConnection(_ context.Context) (*grpc.ClientConn, error) {
	return m.conn, nil
}
func (m *scMockConnProvider) ReturnConnection(_ *grpc.ClientConn) {}
func (m *scMockConnProvider) GetTimeout() time.Duration            { return 30 * time.Second }

func setupSCTestServer(t *testing.T, fake *fakeSCWalletServer) (*Manager, func()) {
	t.Helper()
	lis := bufconn.Listen(scBufSize)
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

	mgr := NewManager(&scMockConnProvider{conn: conn})
	cleanup := func() {
		conn.Close()
		srv.Stop()
		lis.Close()
	}
	return mgr, cleanup
}

var (
	scTestAddr  = scMustAddr("TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY")
	scTestAddr2 = scMustAddr("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
)

func scMustAddr(s string) *types.Address {
	addr, err := types.NewAddressFromBase58(s)
	if err != nil {
		panic(err)
	}
	return addr
}

func TestManagerNew(t *testing.T) {
	mgr := NewManager(&scMockConnProvider{})
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestManagerSimulate(t *testing.T) {
	fake := &fakeSCWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:     &api.Return{Result: true},
				EnergyUsed: 25000,
			}, nil
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()

	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	result, err := inst.Simulate(context.Background(), scTestAddr, 0, "name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Energy != 25000 {
		t.Fatalf("expected energy 25000, got %d", result.Energy)
	}
}

func TestManagerSimulateNilOwner(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()

	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	_, err = inst.Simulate(context.Background(), nil, 0, "name")
	if err == nil {
		t.Fatal("expected error for nil owner")
	}
}

func TestManagerEncode(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()

	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	t.Run("name() no params", func(t *testing.T) {
		data, err := inst.Encode("name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) < 4 {
			t.Fatal("expected at least 4 bytes (method selector)")
		}
	})

	t.Run("transfer(address,uint256)", func(t *testing.T) {
		data, err := inst.Encode("transfer", scTestAddr.String(), big.NewInt(1000))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 68 {
			t.Fatalf("expected 68 bytes, got %d", len(data))
		}
	})

	t.Run("unknown method", func(t *testing.T) {
		_, err := inst.Encode("nonexistent")
		if err == nil {
			t.Fatal("expected error for unknown method")
		}
	})
}

func TestManagerInvoke(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	ctx := context.Background()

	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		result, err := inst.Invoke(ctx, scTestAddr, 0, "transfer", scTestAddr2.String(), big.NewInt(1000))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := inst.Invoke(ctx, nil, 0, "transfer")
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("negative call value", func(t *testing.T) {
		_, err := inst.Invoke(ctx, scTestAddr, -1, "transfer")
		if err == nil {
			t.Fatal("expected error for negative call value")
		}
	})
}

func TestManagerCall(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	ctx := context.Background()

	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		result, err := inst.Call(ctx, scTestAddr, "balanceOf", scTestAddr.String())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := inst.Call(ctx, nil, "balanceOf")
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})
}

func TestManagerEstimateEnergy(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.EstimateEnergy(ctx, scTestAddr, scTestAddr2, []byte{0, 0, 0, 0}, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		_, err := mgr.EstimateEnergy(ctx, scTestAddr, scTestAddr2, nil, 0)
		if err == nil {
			t.Fatal("expected error for empty data")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.EstimateEnergy(ctx, nil, scTestAddr2, []byte{1, 2, 3, 4}, 0)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("nil contract", func(t *testing.T) {
		_, err := mgr.EstimateEnergy(ctx, scTestAddr, nil, []byte{1, 2, 3, 4}, 0)
		if err == nil {
			t.Fatal("expected error for nil contract")
		}
	})

	t.Run("negative call value", func(t *testing.T) {
		_, err := mgr.EstimateEnergy(ctx, scTestAddr, scTestAddr2, []byte{1, 2, 3, 4}, -1)
		if err == nil {
			t.Fatal("expected error for negative call value")
		}
	})
}

func TestManagerGetContract(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetContract(ctx, scTestAddr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil address", func(t *testing.T) {
		_, err := mgr.GetContract(ctx, nil)
		if err == nil {
			t.Fatal("expected error for nil address")
		}
	})
}

func TestManagerGetContractInfo(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.GetContractInfo(ctx, scTestAddr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil address", func(t *testing.T) {
		_, err := mgr.GetContractInfo(ctx, nil)
		if err == nil {
			t.Fatal("expected error for nil address")
		}
	})
}

func TestManagerUpdateSetting(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.UpdateSetting(ctx, scTestAddr, scTestAddr2, 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.UpdateSetting(ctx, nil, scTestAddr2, 50)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("nil contract", func(t *testing.T) {
		_, err := mgr.UpdateSetting(ctx, scTestAddr, nil, 50)
		if err == nil {
			t.Fatal("expected error for nil contract")
		}
	})

	t.Run("invalid percent", func(t *testing.T) {
		_, err := mgr.UpdateSetting(ctx, scTestAddr, scTestAddr2, -1)
		if err == nil {
			t.Fatal("expected error for negative percent")
		}
		_, err = mgr.UpdateSetting(ctx, scTestAddr, scTestAddr2, 101)
		if err == nil {
			t.Fatal("expected error for percent > 100")
		}
	})
}

func TestManagerUpdateEnergyLimit(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.UpdateEnergyLimit(ctx, scTestAddr, scTestAddr2, 30000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.UpdateEnergyLimit(ctx, nil, scTestAddr2, 30000)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("negative limit", func(t *testing.T) {
		_, err := mgr.UpdateEnergyLimit(ctx, scTestAddr, scTestAddr2, -1)
		if err == nil {
			t.Fatal("expected error for negative limit")
		}
	})
}

func TestManagerClearContractABI(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		result, err := mgr.ClearContractABI(ctx, scTestAddr, scTestAddr2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.ClearContractABI(ctx, nil, scTestAddr2)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("nil contract", func(t *testing.T) {
		_, err := mgr.ClearContractABI(ctx, scTestAddr, nil)
		if err == nil {
			t.Fatal("expected error for nil contract")
		}
	})
}

func TestManagerGetContractFromNetworkNilAddr(t *testing.T) {
	_, err := getContractFromNetwork(context.Background(), &scMockConnProvider{}, nil)
	if err == nil {
		t.Fatal("expected error for nil address")
	}
}

func TestManagerGetConstructorTypes(t *testing.T) {
	inst, err := NewInstance(createMockClient(), scTestAddr, testERC20ABI)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	// ERC20 ABI has no constructor, so this should return an error
	_, err = inst.getConstructorTypes()
	if err == nil {
		t.Fatal("expected error for missing constructor")
	}
}

func TestManagerInstanceWithABIFromNetwork(t *testing.T) {
	fake := &fakeSCWalletServer{
		GetContractFunc: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{
				Abi: &core.SmartContract_ABI{
					Entrys: []*core.SmartContract_ABI_Entry{
						{
							Name: "name",
							Type: core.SmartContract_ABI_Entry_Function,
							Outputs: []*core.SmartContract_ABI_Entry_Param{
								{Name: "", Type: "string"},
							},
						},
					},
				},
			}, nil
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()

	inst, err := mgr.Instance(scTestAddr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst == nil {
		t.Fatal("expected non-nil instance")
	}
	if inst.ABI == nil {
		t.Fatal("expected non-nil ABI")
	}
}

func TestManagerDeploy(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	ctx := context.Background()

	t.Run("success with ABI string", func(t *testing.T) {
		result, err := mgr.Deploy(ctx, scTestAddr, "TestContract", testERC20ABI, []byte{0x60, 0x00}, 0, 100, 30000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("nil owner", func(t *testing.T) {
		_, err := mgr.Deploy(ctx, nil, "TestContract", testERC20ABI, []byte{0x60}, 0, 100, 30000)
		if err == nil {
			t.Fatal("expected error for nil owner")
		}
	})

	t.Run("empty bytecode", func(t *testing.T) {
		_, err := mgr.Deploy(ctx, scTestAddr, "TestContract", testERC20ABI, nil, 0, 100, 30000)
		if err == nil {
			t.Fatal("expected error for empty bytecode")
		}
	})

	t.Run("negative call value", func(t *testing.T) {
		_, err := mgr.Deploy(ctx, scTestAddr, "TestContract", testERC20ABI, []byte{0x60}, -1, 100, 30000)
		if err == nil {
			t.Fatal("expected error for negative call value")
		}
	})

	t.Run("negative energy limit", func(t *testing.T) {
		_, err := mgr.Deploy(ctx, scTestAddr, "TestContract", testERC20ABI, []byte{0x60}, 0, 100, -1)
		if err == nil {
			t.Fatal("expected error for negative energy limit")
		}
	})
}
