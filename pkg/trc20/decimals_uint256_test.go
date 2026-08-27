package trc20_test

import (
	"context"
	"math/big"
	"testing"

	eabi "github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/kslamph/tronlib/internal/testutil"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
)

// uint256DecimalsServer returns decimals data packed as uint256 instead of uint8.
// Some non-standard TRC20 contracts encode decimals as uint256 on-chain.
// This server verifies that trc20.NewManager correctly handles that case.
type uint256DecimalsServer struct {
	api.UnimplementedWalletServer
}

func (s *uint256DecimalsServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	var result [][]byte
	selector := in.Data[:4]
	if selector[0] == 0x31 && selector[1] == 0x3c {
		// Pack decimals as uint256 (not uint8) to exercise the edge case that
		// non-standard TRC20 contracts may use.
		out, _ := packUint256(big.NewInt(6))
		result = [][]byte{out}
	} else {
		out, _ := handleTRC20ConstantResult(selector, "TestToken", "TT", 6)
		result = [][]byte{out}
	}

	return &api.TransactionExtention{
		Result:         &api.Return{Result: true, Code: api.Return_SUCCESS},
		ConstantResult: result,
	}, nil
}

func (s *uint256DecimalsServer) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{
		Result: &api.Return{Result: true, Code: api.Return_SUCCESS},
		Txid:   []byte{0x01, 0x02},
	}, nil
}

// TestDecimals_WithUint256PackedResponse verifies that when a TRC20 contract
// packs decimals as uint256 (not uint8), the go-ethereum ABI library still
// returns the correct uint8 value, because the code asserts the result via
// a type assertion that works across uint8→uint256 representations.
// See pkg/trc20/client.go:224-226 for the associated code comment.
//
// VERDICT: This test documents a non-reproducible scenario with go-ethereum
// v1.16.5. go-ethereum's Unpack correctly returns uint8 even when the raw
// contract data is packed as uint256.
func TestDecimals_WithUint256PackedResponse(t *testing.T) {
	lis := testutil.NewBufconnServer(t, &uint256DecimalsServer{})

	c := testutil.NewMockConnProvider(testutil.DialBufconn(t, lis))

	addr := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")

	mgr, err := trc20.NewManager(c, addr)
	if err != nil {
		t.Fatalf("NewManager should succeed: %v", err)
	}

	decimals, err := mgr.Decimals(context.Background())
	if err != nil {
		t.Fatalf("Decimals should succeed: %v", err)
	}
	if decimals != 6 {
		t.Fatalf("expected decimals=6, got %d", decimals)
	}

	t.Log("VERIFIED: go-ethereum v1.16.5 correctly returns uint8 even for uint256-packed data")
	t.Log("  The code is fragile — if go-ethereum changes this unpack behavior, it will break")
}

// TestDecimals_ABITypeCheck verifies the underlying go-ethereum ABI behavior
// to confirm that uint8 assertion is safe across encoding scenarios.
func TestDecimals_ABITypeCheck(t *testing.T) {
	uint8Type, _ := eabi.NewType("uint8", "", nil)
	uint8Args := eabi.Arguments{{Type: uint8Type}}
	uint256Type, _ := eabi.NewType("uint256", "", nil)
	uint256Args := eabi.Arguments{{Type: uint256Type}}

	// pack uint8, unpack uint8
	packedUint8, _ := uint8Args.Pack(uint8(6))
	values, _ := uint8Args.Unpack(packedUint8)
	_, ok := values[0].(uint8)
	if !ok {
		t.Error("pack uint8 → unpack uint8: expected uint8 type")
	}

	// pack uint256, unpack with uint8 ABI type
	packedUint256, _ := uint256Args.Pack(big.NewInt(6))
	values2, _ := uint8Args.Unpack(packedUint256)
	_, ok = values2[0].(uint8)
	if !ok {
		t.Error("pack uint256 → unpack uint8: expected uint8 type")
	}

	t.Log("VERIFIED: go-ethereum v1.16.5 always returns uint8 for uint8 ABI types")
}
