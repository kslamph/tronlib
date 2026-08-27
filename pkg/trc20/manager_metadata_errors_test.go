package trc20_test

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	eabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc/test/bufconn"
)

// ---- ToWeiWithDecimals / FromWeiWithDecimals (trivial wrappers) ----

func TestToWeiWithDecimals_RoundTrip(t *testing.T) {
	amount := decimal.RequireFromString("123.456789")
	wei, err := trc20.ToWeiWithDecimals(amount, 6)
	if err != nil {
		t.Fatalf("ToWeiWithDecimals: %v", err)
	}
	if wei.String() != "123456789" {
		t.Fatalf("unexpected wei: %s", wei.String())
	}
	back, err := trc20.FromWeiWithDecimals(wei, 6)
	if err != nil {
		t.Fatalf("FromWeiWithDecimals: %v", err)
	}
	if back.String() != amount.String() {
		t.Fatalf("roundtrip mismatch: got %s want %s", back.String(), amount.String())
	}
}

func TestToWeiWithDecimals_ZeroDecimals(t *testing.T) {
	wei, err := trc20.ToWeiWithDecimals(decimal.NewFromInt(42), 0)
	if err != nil {
		t.Fatalf("ToWeiWithDecimals(42, 0): %v", err)
	}
	if wei.Int64() != 42 {
		t.Fatalf("expected 42, got %s", wei.String())
	}
}

func TestFromWeiWithDecimals_ZeroDecimals(t *testing.T) {
	d, err := trc20.FromWeiWithDecimals(big.NewInt(42), 0)
	if err != nil {
		t.Fatalf("FromWeiWithDecimals(42, 0): %v", err)
	}
	expected := decimal.NewFromInt(42)
	if !d.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected.String(), d.String())
	}
}

// ---- Flexible test server ----

var (
	selName        = [4]byte{0x06, 0xfd, 0xde, 0x03} // name()
	selSymbol      = [4]byte{0x95, 0xd8, 0x9b, 0x41} // symbol()
	selDecimals    = [4]byte{0x31, 0x3c, 0xe5, 0x67} // decimals()
	selTotalSupply = [4]byte{0x18, 0x16, 0x0d, 0xdd} // totalSupply()
	selBalanceOf   = [4]byte{0x70, 0xa0, 0x82, 0x31} // balanceOf(address)
	selAllowance   = [4]byte{0xdd, 0x62, 0xed, 0x3e} // allowance(address,address)
)

type flexServer struct {
	api.UnimplementedWalletServer
	constantHandlers map[[4]byte]func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	defaultConstant  func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	txHandler        func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	errorSelectors   map[[4]byte]string
}

func (s *flexServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	var sel [4]byte
	copy(sel[:], in.Data[:4])
	if errMsg, ok := s.errorSelectors[sel]; ok {
		return nil, status.Error(codes.Internal, errMsg)
	}
	if h, ok := s.constantHandlers[sel]; ok {
		return h(ctx, in)
	}
	if s.defaultConstant != nil {
		return s.defaultConstant(ctx, in)
	}
	return &api.TransactionExtention{
		Result:         &api.Return{Result: true, Code: api.Return_SUCCESS},
		ConstantResult: [][]byte{func() []byte { v := new(big.Int).SetInt64(1000000000); out, _ := packUint256(v); return out }()},
	}, nil
}

func (s *flexServer) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if s.txHandler != nil {
		return s.txHandler(ctx, in)
	}
	return &api.TransactionExtention{
		Result: &api.Return{Result: true, Code: api.Return_SUCCESS},
		Txid:   []byte{0x01, 0x02},
	}, nil
}

func (s *flexServer) GetAccountInfo(_ context.Context, _ *core.Account) (*core.Account, error) {
	return &core.Account{AccountName: []byte("test")}, nil
}

func newFlexClient(t *testing.T, impl api.WalletServer) (*client.Client, func()) {
	t.Helper()
	lis := bufconn.Listen(trc20BufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, impl)
	go func() { _ = srv.Serve(lis) }()
	c, err := client.NewClientWithDialer(
		"passthrough:///bufnet",
		func(ctx context.Context, s string) (net.Conn, error) { return lis.DialContext(ctx) },
		client.WithTimeout(500*time.Millisecond),
		client.WithPool(1, 1),
	)
	if err != nil {
		t.Fatalf("NewClientWithDialer: %v", err)
	}
	return c, func() { _ = lis.Close(); srv.Stop(); c.Close() }
}

func packTRC20Result(data interface{}) ([]byte, error) {
	switch v := data.(type) {
	case string:
		typ, _ := eabi.NewType("string", "", nil)
		return eabi.Arguments{{Type: typ}}.Pack(v)
	case *big.Int:
		return packUint256(v)
	case uint8:
		return packUint8(v)
	default:
		return nil, fmt.Errorf("unsupported type %T", data)
	}
}

func defaultOKServer() *flexServer {
	s := &flexServer{
		constantHandlers: make(map[[4]byte]func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)),
		errorSelectors:   make(map[[4]byte]string),
	}
	s.constantHandlers[selName] = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		out, _ := packTRC20Result("TRONUSD")
		return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: [][]byte{out}}, nil
	}
	s.constantHandlers[selSymbol] = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		out, _ := packTRC20Result("USDT")
		return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: [][]byte{out}}, nil
	}
	s.constantHandlers[selDecimals] = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		out, _ := packTRC20Result(uint8(6))
		return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: [][]byte{out}}, nil
	}
	return s
}

func makeManager(t *testing.T, srv *flexServer) *trc20.TRC20Manager {
	t.Helper()
	c, cleanup := newFlexClient(t, srv)
	t.Cleanup(cleanup)
	token := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	m, err := trc20.NewManager(c, token)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return m
}

// ---- TotalSupply ----

func TestTRC20Manager_TotalSupply_Success(t *testing.T) {
	srv := defaultOKServer()
	m := makeManager(t, srv)
	supply, err := m.TotalSupply(context.Background())
	if err != nil {
		t.Fatalf("TotalSupply: %v", err)
	}
	expected := decimal.RequireFromString("1000")
	if supply.String() != expected.String() {
		t.Fatalf("TotalSupply = %s, want %s", supply.String(), expected.String())
	}
}

func TestTRC20Manager_TotalSupply_RPCError(t *testing.T) {
	srv := defaultOKServer()
	srv.errorSelectors[selTotalSupply] = "totalSupply failed"
	m := makeManager(t, srv)
	_, err := m.TotalSupply(context.Background())
	if err == nil {
		t.Fatal("expected error from TotalSupply with RPC error")
	}
}

func TestTRC20Manager_TotalSupply_EmptyConstantResult(t *testing.T) {
	srv := defaultOKServer()
	srv.constantHandlers[selTotalSupply] = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: [][]byte{}}, nil
	}
	m := makeManager(t, srv)
	_, err := m.TotalSupply(context.Background())
	if err == nil {
		t.Fatal("expected error from TotalSupply with empty constant result")
	}
}

// ---- Name / Symbol unexpected types (ABI decode failure path) ----

func TestTRC20Manager_Name_UnexpectedType(t *testing.T) {
	srv := defaultOKServer()
	srv.constantHandlers[selName] = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		out, _ := packTRC20Result(uint8(42))
		return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: [][]byte{out}}, nil
	}
	c, cleanup := newFlexClient(t, srv)
	defer cleanup()
	token := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := trc20.NewManager(c, token)
	if err == nil {
		t.Fatal("expected error from NewManager when Name returns unexpected type")
	}
}

func TestTRC20Manager_Symbol_UnexpectedType(t *testing.T) {
	srv := defaultOKServer()
	srv.constantHandlers[selSymbol] = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		out, _ := packTRC20Result(uint8(42))
		return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: [][]byte{out}}, nil
	}
	c, cleanup := newFlexClient(t, srv)
	defer cleanup()
	token := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := trc20.NewManager(c, token)
	if err == nil {
		t.Fatal("expected error from NewManager when Symbol returns unexpected type")
	}
}

// ---- RPC errors for Name / Symbol / Decimals ----

func TestTRC20Manager_Name_RPCError(t *testing.T) {
	srv := defaultOKServer()
	srv.errorSelectors[selName] = "name call failed"
	c, cleanup := newFlexClient(t, srv)
	defer cleanup()
	token := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := trc20.NewManager(c, token)
	if err == nil {
		t.Fatal("expected error from NewManager when Name RPC fails")
	}
}

func TestTRC20Manager_Symbol_RPCError(t *testing.T) {
	srv := defaultOKServer()
	srv.errorSelectors[selSymbol] = "symbol call failed"
	c, cleanup := newFlexClient(t, srv)
	defer cleanup()
	token := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := trc20.NewManager(c, token)
	if err == nil {
		t.Fatal("expected error from NewManager when Symbol RPC fails")
	}
}

func TestTRC20Manager_Decimals_RPCError(t *testing.T) {
	srv := defaultOKServer()
	srv.errorSelectors[selDecimals] = "decimals call failed"
	c, cleanup := newFlexClient(t, srv)
	defer cleanup()
	token := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := trc20.NewManager(c, token)
	if err == nil {
		t.Fatal("expected error from NewManager when Decimals RPC fails")
	}
}

// ---- BalanceOf error branches ----

func TestTRC20Manager_BalanceOf_RPCError(t *testing.T) {
	srv := defaultOKServer()
	srv.errorSelectors[selBalanceOf] = "balanceOf failed"
	m := makeManager(t, srv)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.BalanceOf(context.Background(), owner)
	if err == nil {
		t.Fatal("expected error from BalanceOf with RPC error")
	}
}

func TestTRC20Manager_BalanceOf_EmptyConstantResult(t *testing.T) {
	srv := defaultOKServer()
	srv.constantHandlers[selBalanceOf] = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: [][]byte{}}, nil
	}
	m := makeManager(t, srv)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.BalanceOf(context.Background(), owner)
	if err == nil {
		t.Fatal("expected error from BalanceOf with empty constant result")
	}
}

// ---- Transfer error branches ----

func TestTRC20Manager_Transfer_RPCError(t *testing.T) {
	srv := defaultOKServer()
	srv.txHandler = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return nil, status.Error(codes.Internal, "transfer failed")
	}
	m := makeManager(t, srv)
	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	_, err := m.Transfer(context.Background(), from, to, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected error from Transfer with RPC error")
	}
}

func TestTRC20Manager_Transfer_InvalidAmount(t *testing.T) {
	srv := defaultOKServer()
	m := makeManager(t, srv)
	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	_, err := m.Transfer(context.Background(), from, to, decimal.RequireFromString("1.1234567"))
	if err == nil {
		t.Fatal("expected error from Transfer with too many decimal places")
	}
}

// ---- Approve error branches ----

func TestTRC20Manager_Approve_RPCError(t *testing.T) {
	srv := defaultOKServer()
	srv.txHandler = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return nil, status.Error(codes.Internal, "approve failed")
	}
	m := makeManager(t, srv)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	spender := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	_, err := m.Approve(context.Background(), owner, spender, decimal.NewFromInt(100))
	if err == nil {
		t.Fatal("expected error from Approve with RPC error")
	}
}

func TestTRC20Manager_Approve_InvalidAmount(t *testing.T) {
	srv := defaultOKServer()
	m := makeManager(t, srv)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	spender := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	_, err := m.Approve(context.Background(), owner, spender, decimal.RequireFromString("1.1234567"))
	if err == nil {
		t.Fatal("expected error from Approve with too many decimal places")
	}
}

// ---- Allowance error branches ----

func TestTRC20Manager_Allowance_RPCError(t *testing.T) {
	srv := defaultOKServer()
	srv.errorSelectors[selAllowance] = "allowance failed"
	m := makeManager(t, srv)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	spender := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	_, err := m.Allowance(context.Background(), owner, spender)
	if err == nil {
		t.Fatal("expected error from Allowance with RPC error")
	}
}

func TestTRC20Manager_Allowance_EmptyConstantResult(t *testing.T) {
	srv := defaultOKServer()
	srv.constantHandlers[selAllowance] = func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: [][]byte{}}, nil
	}
	m := makeManager(t, srv)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	spender := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	_, err := m.Allowance(context.Background(), owner, spender)
	if err == nil {
		t.Fatal("expected error from Allowance with empty constant result")
	}
}

// ---- Transfer / Approve zero amount ----

func TestTRC20Manager_Transfer_ZeroAmount(t *testing.T) {
	srv := defaultOKServer()
	m := makeManager(t, srv)
	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	txext, err := m.Transfer(context.Background(), from, to, decimal.NewFromInt(0))
	if err != nil {
		t.Fatalf("Transfer(0): %v", err)
	}
	if txext == nil {
		t.Fatal("expected non-nil TransactionExtention")
	}
}

func TestTRC20Manager_Approve_ZeroAmount(t *testing.T) {
	srv := defaultOKServer()
	m := makeManager(t, srv)
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	spender := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	txext, err := m.Approve(context.Background(), owner, spender, decimal.NewFromInt(0))
	if err != nil {
		t.Fatalf("Approve(0): %v", err)
	}
	if txext == nil {
		t.Fatal("expected non-nil TransactionExtention")
	}
}
