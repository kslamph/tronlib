package smartcontract

import (
	"context"
	"encoding/json"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// Minimal bufconn helpers local to this package

const managerBufSize = 1024 * 1024

type deployCaptureServer struct {
	api.UnimplementedWalletServer
	reqCh chan *core.CreateSmartContract
}

func (s *deployCaptureServer) DeployContract(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error) {
	// Non-blocking send to avoid deadlocks if test exits early
	select {
	case s.reqCh <- in:
	default:
	}
	return &api.TransactionExtention{
		Result: &api.Return{Result: true, Code: api.Return_SUCCESS},
	}, nil
}

func newBufconnServer(t *testing.T, impl api.WalletServer) (*bufconn.Listener, *grpc.Server, func()) {
	t.Helper()
	lis := bufconn.Listen(managerBufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, impl)
	go func() { _ = srv.Serve(lis) }()
	cleanup := func() {
		_ = lis.Close()
		srv.Stop()
	}
	return lis, srv, cleanup
}

func newTestClientWithBufConn(t *testing.T, lis *bufconn.Listener, timeout time.Duration) (*client.Client, func()) {
	t.Helper()
	// Build a test client using the approved test-only dialer helper.
	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
	cfg := client.ClientConfig{
		NodeAddress:     "bufnet",
		Timeout:         timeout,
		InitConnections: 1,
		MaxConnections:  1,
		// IdleTimeout:     time.Second,
	}
	c, err := client.NewClientWithDialer(cfg, dialer)
	if err != nil {
		t.Fatalf("NewClientWithDialer error: %v", err)
	}
	cleanup := func() {
		c.Close()
	}
	return c, cleanup
}

// Test 1: ABI string with constructor(address,uint256) encodes params appended to bytecode.
func TestDeployContract_ABIString_ConstructorEncoding(t *testing.T) {
	// ABI JSON with constructor(address,uint256)
	type abiEntry struct {
		Type            string `json:"type"`
		StateMutability string `json:"stateMutability,omitempty"`
		Inputs          []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"inputs,omitempty"`
	}
	abiJSONBytes, _ := json.Marshal([]abiEntry{
		{
			Type: "constructor",
			Inputs: []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			}{
				{Name: "addr", Type: "address"},
				{Name: "amount", Type: "uint256"},
			},
		},
	})
	abiJSON := string(abiJSONBytes)

	// Fake server to capture request
	srv := &deployCaptureServer{reqCh: make(chan *core.CreateSmartContract, 1)}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)

	cli, cleanupCli := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupCli)

	mgr := NewManager(cli)

	owner, err := types.NewAddress("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	if err != nil {
		t.Fatalf("owner addr parse: %v", err)
	}

	bytecode := []byte{0xde, 0xad, 0xbe, 0xef}
	addrParam := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	u256 := big.NewInt(123)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	consume := int64(50)
	originLimit := int64(100000)
	_, err = mgr.DeployContract(ctx, owner, "MyContract", abiJSON, bytecode, 0, consume, originLimit, addrParam, u256)
	if err != nil {
		t.Fatalf("DeployContract error: %v", err)
	}

	select {
	case req := <-srv.reqCh:
		if req.NewContract == nil {
			t.Fatalf("NewContract nil")
		}
		if len(req.NewContract.Bytecode) <= len(bytecode) {
			t.Fatalf("expected encoded params appended, got len %d base %d", len(req.NewContract.Bytecode), len(bytecode))
		}
		// Suffix should differ to indicate params appended (cannot fully decode here)
		if string(req.NewContract.Bytecode[len(req.NewContract.Bytecode)-4:]) == string(bytecode[len(bytecode)-4:]) {
			t.Fatalf("expected different suffix due to encoded params")
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for captured request")
	}
}

// Test 2: Parsed *core.SmartContract_ABI mirrored constructor.
func TestDeployContract_ParsedABI_ConstructorEncoding(t *testing.T) {
	parsed := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Type: core.SmartContract_ABI_Entry_Constructor,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "addr", Type: "address"},
					{Name: "amount", Type: "uint256"},
				},
			},
		},
	}

	srv := &deployCaptureServer{reqCh: make(chan *core.CreateSmartContract, 1)}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)

	cli, cleanupCli := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupCli)

	mgr := NewManager(cli)

	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	bytecode := []byte{0xde, 0xad, 0xbe, 0xef}
	addrParam := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	u256 := big.NewInt(123)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := mgr.DeployContract(ctx, owner, "Another", parsed, bytecode, 0, 0, 1, addrParam, u256)
	if err != nil {
		t.Fatalf("DeployContract error: %v", err)
	}

	select {
	case req := <-srv.reqCh:
		if req.NewContract == nil {
			t.Fatalf("NewContract nil")
		}
		if len(req.NewContract.Bytecode) <= len(bytecode) {
			t.Fatalf("expected encoded params appended")
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting capture")
	}
}

// Test 3: Parameter count mismatch error path
func TestDeployContract_ParamCountMismatch(t *testing.T) {
	parsed := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Type: core.SmartContract_ABI_Entry_Constructor,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "addr", Type: "address"},
					{Name: "amount", Type: "uint256"},
				},
			},
		},
	}

	srv := &deployCaptureServer{reqCh: make(chan *core.CreateSmartContract, 1)}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)

	cli, cleanupCli := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupCli)

	mgr := NewManager(cli)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	bytecode := []byte{0x01, 0x02}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Only 1 param, expected 2
	_, err := mgr.DeployContract(ctx, owner, "Mismatch", parsed, bytecode, 0, 0, 1, owner)
	if err == nil {
		t.Fatalf("expected error on parameter count mismatch")
	}
	if err != nil && !contains(err.Error(), "constructor parameter count mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test 4: abi=nil with non-empty constructor params error
func TestDeployContract_NilABIWithParams(t *testing.T) {
	srv := &deployCaptureServer{reqCh: make(chan *core.CreateSmartContract, 1)}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)

	cli, cleanupCli := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupCli)

	mgr := NewManager(cli)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	bytecode := []byte{0xaa}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := mgr.DeployContract(ctx, owner, "NilABI", nil, bytecode, 0, 0, 1, owner)
	if err == nil {
		t.Fatalf("expected error when abi is nil but params provided")
	}
	if !contains(err.Error(), "ABI is required when constructor parameters are provided") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test 5: Invalid parameters validation (consumeUserResourcePercent out of range OR originEnergyLimit negative)
func TestDeployContract_InvalidParameters(t *testing.T) {
	srv := &deployCaptureServer{reqCh: make(chan *core.CreateSmartContract, 1)}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)

	cli, cleanupCli := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupCli)

	mgr := NewManager(cli)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	bytecode := []byte{0x00}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// consumeUserResourcePercent = 101
	if _, err := mgr.DeployContract(ctx, owner, "BadConsume", nil, bytecode, 0, 101, 1); err == nil {
		t.Fatalf("expected error for consume user resource percent out of range")
	}

	// originEnergyLimit = -1
	if _, err := mgr.DeployContract(ctx, owner, "BadEnergy", nil, bytecode, 0, 0, -1); err == nil {
		t.Fatalf("expected error for negative origin energy limit")
	}
}

// contains helper (avoid strings import to keep single block request spec tight)
func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

// naive index to avoid extra imports
func indexOf(s, sub string) int {
outer:
	for i := 0; i+len(sub) <= len(s); i++ {
		for j := 0; j < len(sub); j++ {
			if s[i+j] != sub[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}
