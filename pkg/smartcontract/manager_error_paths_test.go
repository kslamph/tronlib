package smartcontract

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testConstructorABI defines an ABI with a constructor that takes (string, uint256) params.
const testConstructorABI = `[
	{
		"type": "constructor",
		"inputs": [
			{"name": "name", "type": "string"},
			{"name": "supply", "type": "uint256"}
		]
	},
	{
		"type": "function",
		"name": "name",
		"inputs": [],
		"outputs": [{"name": "", "type": "string"}]
	}
]`

// --- Deploy coverage tests ---

func TestDeploy_ValidationAndEncoding(t *testing.T) {
	processor := utils.NewABIProcessor(nil)
	parsedABI, err := processor.ParseABI(testERC20ABI)
	require.NoError(t, err)

	tests := []struct {
		name         string
		contractName string
		abi          any
		resourcePct  int64
		params       []any
		wantErr      bool
	}{
		{"control characters in name", "\x01test", testERC20ABI, 100, nil, true},
		{"resource percent above 100", "Test", testERC20ABI, 101, nil, true},
		{"empty ABI string", "Test", "", 100, nil, true},
		{"invalid ABI JSON", "Test", "not-json", 100, nil, true},
		{"nil ABI object", "Test", (*core.SmartContract_ABI)(nil), 100, nil, true},
		{"parsed ABI object", "Test", parsedABI, 100, nil, false},
		{"unsupported ABI type", "Test", 123, 100, nil, true},
		{"constructor params with nil ABI", "Test", nil, 100, []any{"param1"}, true},
		{"constructor params but ABI has none", "Test", testERC20ABI, 100, []any{"param1"}, true},
		{"constructor param count mismatch", "Test", testConstructorABI, 100, []any{"only_one_param"}, true},
		{"valid constructor params", "Test", testConstructorABI, 100, []any{"MyToken", big.NewInt(1000000)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
			defer cleanup()
			result, err := mgr.Deploy(context.Background(), scTestAddr, tt.contractName, tt.abi, []byte{0x60}, 0, tt.resourcePct, 30000, tt.params...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestMockDeployServerTxError(t *testing.T) {
	fake := &fakeSCWalletServer{
		DeployContractFunc: func(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error) {
			return nil, status.Error(codes.Unavailable, "node unavailable")
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()
	_, err := mgr.Deploy(context.Background(), scTestAddr, "Test", testERC20ABI, []byte{0x60}, 0, 100, 30000)
	assert.Error(t, err, "should fail on server error")
}

// --- Call coverage tests ---

func TestMockCallEncodeError(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	require.NoError(t, err)
	// Call with nonexistent method → Encode fails
	_, err = inst.Call(context.Background(), scTestAddr, "nonexistent_method_xyz")
	assert.Error(t, err, "should fail with invalid method")
}

func TestMockCallEmptyResponse(t *testing.T) {
	// A zero-value extention is what the client actually receives when the
	// server returns no usable result — never a Go nil.
	fake := &fakeSCWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()
	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	require.NoError(t, err)
	_, err = inst.Call(context.Background(), scTestAddr, "name")
	assert.Error(t, err, "should fail with empty response")
}

func TestMockCallEmptyConstantResult(t *testing.T) {
	fake := &fakeSCWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true},
				ConstantResult: [][]byte{},
			}, nil
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()
	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	require.NoError(t, err)
	_, err = inst.Call(context.Background(), scTestAddr, "name")
	assert.Error(t, err, "should fail with empty constant result")
}

func TestMockCallDecodeError(t *testing.T) {
	fake := &fakeSCWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true},
				ConstantResult: [][]byte{[]byte{0xff, 0xff}}, // 2 bytes, uint256 expects 32
			}, nil
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()
	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	require.NoError(t, err)
	_, err = inst.Call(context.Background(), scTestAddr, "balanceOf", scTestAddr.String())
	assert.Error(t, err, "should fail when decode result fails")
}

func TestMockCallServerError(t *testing.T) {
	fake := &fakeSCWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return nil, status.Error(codes.Unavailable, "node unavailable")
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()
	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	require.NoError(t, err)
	_, err = inst.Call(context.Background(), scTestAddr, "name")
	assert.Error(t, err, "should fail on server error")
}

// --- Simulate coverage tests ---

func TestMockSimulateEncodeError(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	require.NoError(t, err)
	_, err = inst.Simulate(context.Background(), scTestAddr, 0, "nonexistent_method_xyz")
	assert.Error(t, err, "should fail with invalid method")
}

func TestMockSimulateServerError(t *testing.T) {
	fake := &fakeSCWalletServer{
		TriggerConstantContractFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return nil, status.Error(codes.Unavailable, "node unavailable")
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()
	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	require.NoError(t, err)
	_, err = inst.Simulate(context.Background(), scTestAddr, 0, "name")
	assert.Error(t, err, "should fail on server error")
}

// (TestMockSimulate_ServerError was removed: an exact duplicate of
// TestMockSimulateServerError above.)

// --- Invoke coverage tests ---

func TestMockInvokeEncodeError(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	inst, err := mgr.Instance(scTestAddr, testERC20ABI)
	require.NoError(t, err)
	_, err = inst.Invoke(context.Background(), scTestAddr, 0, "nonexistent_method_xyz")
	assert.Error(t, err, "should fail with invalid method")
}

// --- getConstructorTypes coverage tests ---

func TestMockGetConstructorTypesFound(t *testing.T) {
	inst, err := NewInstance(createMockClient(), scTestAddr, testConstructorABI)
	require.NoError(t, err)
	ctorTypes, err := inst.getConstructorTypes()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ctorTypes))
	assert.Equal(t, "string", ctorTypes[0])
	assert.Equal(t, "uint256", ctorTypes[1])
}

// --- Encode coverage tests (constructor path) ---

func TestMockEncodeConstructorNotFound(t *testing.T) {
	inst, err := NewInstance(createMockClient(), scTestAddr, testERC20ABI)
	require.NoError(t, err)
	// testERC20ABI has no constructor → getConstructorTypes fails
	_, err = inst.Encode("")
	assert.Error(t, err, "should fail when no constructor in ABI")
}

func TestMockEncodeConstructorSuccess(t *testing.T) {
	inst, err := NewInstance(createMockClient(), scTestAddr, testConstructorABI)
	require.NoError(t, err)
	data, err := inst.Encode("", "MyToken", big.NewInt(1000000))
	assert.NoError(t, err)
	assert.True(t, len(data) > 0, "encoded constructor data should be non-empty")
}

func TestMockEncodeConstructorEmpty(t *testing.T) {
	inst, err := NewInstance(createMockClient(), scTestAddr, testConstructorABI)
	require.NoError(t, err)
	// No params → EncodeMethod returns empty
	data, err := inst.Encode("")
	assert.NoError(t, err)
	assert.Empty(t, data, "no constructor params should return empty data")
}

// --- DecodeResult coverage tests ---

func TestMockDecodeResultInvalidMethod(t *testing.T) {
	inst, err := NewInstance(createMockClient(), scTestAddr, testERC20ABI)
	require.NoError(t, err)
	_, err = inst.DecodeResult("nonexistent_method_xyz", []byte{0x00, 0x00, 0x00, 0x00})
	assert.Error(t, err, "should fail with invalid method name")
}

func TestMockDecodeResultBadData(t *testing.T) {
	inst, err := NewInstance(createMockClient(), scTestAddr, testERC20ABI)
	require.NoError(t, err)
	// balanceOf returns uint256 (32 bytes expected). Pass 4 bytes → decode should fail
	_, err = inst.DecodeResult("balanceOf", []byte{0xff, 0xff, 0xff, 0xff})
	assert.Error(t, err, "should fail with insufficient data for decode")
}

// --- NewInstance network fetch error paths ---

func TestMockNewInstanceNetworkError(t *testing.T) {
	fake := &fakeSCWalletServer{
		GetContractFunc: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return nil, fmt.Errorf("network error")
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()
	_, err := mgr.Instance(scTestAddr) // no ABI → triggers network fetch
	assert.Error(t, err, "should fail on network error")
}

func TestMockNewInstanceNilABIFromNetwork(t *testing.T) {
	fake := &fakeSCWalletServer{
		GetContractFunc: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{
				Abi: nil,
			}, nil
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()
	_, err := mgr.Instance(scTestAddr) // no ABI → triggers network fetch → nil ABI
	assert.Error(t, err, "should fail when ABI is nil on network")
}

// --- mockClient coverage (test_helpers.go) ---

func TestMockMockClientGetConnError(t *testing.T) {
	// Calling NewInstance without ABI exercises the mockClient's
	// GetConnection, ReturnConnection, and GetTimeout methods.
	// Use a client whose GetConnection returns an error to avoid nil-pointer panics.
	mc := &mockClientWithConnError{}
	_, err := NewInstance(mc, scTestAddr)
	assert.Error(t, err, "mock client without ABI should fail")
}

type mockClientWithConnError struct{}

func (m *mockClientWithConnError) GetConnection(ctx context.Context) (*grpc.ClientConn, error) {
	return nil, fmt.Errorf("mock connection error")
}

func (m *mockClientWithConnError) ReturnConnection(conn *grpc.ClientConn) {}

func (m *mockClientWithConnError) GetTimeout() time.Duration {
	return 5 * time.Second
}

// --- Manager method coverage with nil addresses ---

func TestManager_NilContractAddress(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()

	tests := []struct {
		name string
		call func() error
	}{
		{"UpdateEnergyLimit", func() error {
			_, err := mgr.UpdateEnergyLimit(context.Background(), scTestAddr, nil, 10000)
			return err
		}},
		{"UpdateSetting", func() error { _, err := mgr.UpdateSetting(context.Background(), scTestAddr, nil, 100); return err }},
		{"ClearContractABI", func() error { _, err := mgr.ClearContractABI(context.Background(), scTestAddr, nil); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.call(), "should fail with nil contract address")
		})
	}
}

// --- EstimateEnergy coverage ---

func TestMockEstimateEnergyInvalidData(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	// Empty data should fail ValidateContractData
	_, err := mgr.EstimateEnergy(context.Background(), scTestAddr, scTestAddr, []byte{}, 0)
	assert.Error(t, err, "should fail with empty data")

	// Data too short for function selector (needs at least 4 bytes)
	_, err = mgr.EstimateEnergy(context.Background(), scTestAddr, scTestAddr, []byte{0x01, 0x02}, 0)
	assert.Error(t, err, "should fail with short data")
}

func TestMockEstimateEnergyServerTxError(t *testing.T) {
	fake := &fakeSCWalletServer{
		EstimateEnergyFunc: func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
			return nil, fmt.Errorf("estimate energy server error")
		},
	}
	mgr, cleanup := setupSCTestServer(t, fake)
	defer cleanup()
	// Use valid encoded data (balanceOf(address) selector + address)
	data := []byte{
		0x70, 0xa0, 0x82, 0x31, // balanceOf(address) selector
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0xa0, 0xb1, 0xc2, 0xd3,
		0xe4, 0xf5, 0xa6, 0xb7, 0xc8, 0xd9, 0xe0, 0xf1,
		0xa2, 0xb3, 0xc4, 0xd5, 0xe6, 0xf7, 0xa8, 0xb9,
	}
	_, err := mgr.EstimateEnergy(context.Background(), scTestAddr, scTestAddr, data, 0)
	assert.Error(t, err, "should fail on server error")
}

// --- GetContractInfo coverage ---

func TestMockGetContractInfoNilContract(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	_, err := mgr.GetContractInfo(context.Background(), nil)
	assert.Error(t, err, "should fail with nil address")
}

func TestMockGetContractNilAddress(t *testing.T) {
	mgr, cleanup := setupSCTestServer(t, &fakeSCWalletServer{})
	defer cleanup()
	_, err := mgr.GetContract(context.Background(), nil)
	assert.Error(t, err, "should fail with nil address")
}

// --- lowlevel.Call direct error paths ---

type testConnProvider struct {
	getConnFunc func(ctx context.Context) (*grpc.ClientConn, error)
	returnFunc  func(conn *grpc.ClientConn)
	timeout     time.Duration
}

func (m *testConnProvider) GetConnection(ctx context.Context) (*grpc.ClientConn, error) {
	if m.getConnFunc != nil {
		return m.getConnFunc(ctx)
	}
	return nil, nil
}

func (m *testConnProvider) ReturnConnection(conn *grpc.ClientConn) {
	if m.returnFunc != nil {
		m.returnFunc(conn)
	}
}

func (m *testConnProvider) GetTimeout() time.Duration {
	if m.timeout > 0 {
		return m.timeout
	}
	return 30 * time.Second
}

func TestMockLowlevelCallGetConnectionError(t *testing.T) {
	cp := &testConnProvider{
		getConnFunc: func(ctx context.Context) (*grpc.ClientConn, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	_, err := lowlevel.Call(cp, context.Background(), "test op",
		func(cl api.WalletClient, ctx context.Context) (*api.AccountResourceMessage, error) {
			return cl.GetAccountResource(ctx, &core.Account{})
		})
	assert.Error(t, err, "should fail when GetConnection returns error")
}

func TestMockLowlevelCallWithTimeout(t *testing.T) {
	cp := &testConnProvider{
		getConnFunc: func(ctx context.Context) (*grpc.ClientConn, error) {
			return nil, fmt.Errorf("always fail")
		},
	}
	// GetTimeout will be called since ctx has no deadline
	_, err := lowlevel.Call(cp, context.Background(), "test timeout",
		func(cl api.WalletClient, ctx context.Context) (*api.AccountResourceMessage, error) {
			return nil, fmt.Errorf("unreachable")
		})
	assert.Error(t, err)
}
