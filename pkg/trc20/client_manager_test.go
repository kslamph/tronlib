package trc20_test

import (
	"context"
	"math/big"
	"testing"

	eabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/shopspring/decimal"

	"github.com/kslamph/tronlib/internal/testutil"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc/test/bufconn"
)

type trc20Server struct {
	api.UnimplementedWalletServer
}

func (s *trc20Server) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	out, _ := handleTRC20ConstantResult(in.Data[:4], "TRONUSD", "USDT", 6)
	result := [][]byte{out}
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: result}, nil
}

func (s *trc20Server) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, Txid: []byte{0x01, 0x02}}, nil
}

func newTRC20BufServer(t *testing.T, impl api.WalletServer) *bufconn.Listener {
	t.Helper()
	return testutil.NewBufconnServer(t, impl)
}

func packStr(s string) ([]byte, error) {
	typ, err := eabi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	return eabi.Arguments{{Type: typ}}.Pack(s)
}

// handleTRC20ConstantResult dispatches the common ERC20 read-method selectors
// (name, symbol, decimals, balanceOf/allowance) and returns the packed result.
// Shared by the standard trc20Server and the edge-case server so the selector
// patterns are defined in exactly one place.
func handleTRC20ConstantResult(selector []byte, name, symbol string, decimals uint64) ([]byte, error) {
	switch {
	case selector[0] == 0x06 && selector[1] == 0xfd: // name()
		return packStr(name)
	case selector[0] == 0x95 && selector[1] == 0xd8: // symbol()
		return packStr(symbol)
	case selector[0] == 0x31 && selector[1] == 0x3c: // decimals()
		return packUint8(uint8(decimals))
	default:
		// balanceOf or allowance → return 1000 * 10^6
		return packUint256(new(big.Int).SetUint64(1000000000))
	}
}

func packUint8(u uint8) ([]byte, error) {
	typ, err := eabi.NewType("uint8", "", nil)
	if err != nil {
		return nil, err
	}
	return eabi.Arguments{{Type: typ}}.Pack(u)
}

func packUint256(v *big.Int) ([]byte, error) {
	typ, err := eabi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	return eabi.Arguments{{Type: typ}}.Pack(v)
}

// small helpers removed; using go-ethereum abi directly

func TestTRC20Manager_ReadMethodsAndCaching(t *testing.T) {
	lis := newTRC20BufServer(t, &trc20Server{})

	c := testutil.NewMockConnProvider(testutil.DialBufconn(t, lis))

	addr := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	m, err := trc20.NewManager(c, addr)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	ctx := context.Background()
	name1, _ := m.Name(ctx)
	sym1, _ := m.Symbol(ctx)
	dec1, _ := m.Decimals(ctx)

	// Call again to hit cache
	name2, _ := m.Name(ctx)
	sym2, _ := m.Symbol(ctx)
	dec2, _ := m.Decimals(ctx)

	if name1 != "TRONUSD" || name2 != name1 {
		t.Fatalf("name cache mismatch")
	}
	if sym1 != "USDT" || sym2 != sym1 {
		t.Fatalf("symbol cache mismatch")
	}
	if dec1 != 6 || dec2 != dec1 {
		t.Fatalf("decimals cache mismatch")
	}
}

func TestTRC20Manager_BalanceAllowanceTransferApprove(t *testing.T) {
	lis := newTRC20BufServer(t, &trc20Server{})

	c := testutil.NewMockConnProvider(testutil.DialBufconn(t, lis))

	token := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	spender := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")

	m, err := trc20.NewManager(c, token)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	ctx := context.Background()
	bal, err := m.BalanceOf(ctx, owner)
	if err != nil {
		t.Fatalf("BalanceOf: %v", err)
	}
	if bal.IsZero() {
		t.Fatalf("expected non-zero balance")
	}

	alw, err := m.Allowance(ctx, owner, spender)
	if err != nil {
		t.Fatalf("Allowance: %v", err)
	}
	if alw.IsZero() {
		t.Fatalf("expected non-zero allowance")
	}

	amt := decimal.RequireFromString("1.5")
	txext, err := m.Transfer(ctx, owner, spender, amt)
	if err != nil || txext == nil {
		t.Fatalf("Transfer failed: %v", err)
	}

	txext2, err := m.Approve(ctx, owner, spender, amt)
	if err != nil || txext2 == nil {
		t.Fatalf("Approve failed: %v", err)
	}
}
