package tx

// Tests for the post-sign all-in preview: TotalCostOf combines the energy
// preview (ContractTx only) with the bandwidth prediction, failing fast on
// unsigned transactions before any simulation RPC is spent.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

func mustSignedContract(t *testing.T, f *fakeWalletServer) *ContractTx {
	t.Helper()
	cp := newTxTestClient(t, f)
	ct, err := BuildTriggerSmartContract(cp, t.Context(), testFrom, testTo, []byte{0xa9, 0x05}, 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	signed, err := ct.Sign(mustSigner(t, testKeyHex))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func TestTotalCostContractCombinesBoth(t *testing.T) {
	f := &fakeWalletServer{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := triggerExt()
			ext.EnergyUsed = 13569 // the live-verified figure
			return ext, nil
		},
		AccountResource: richResource(100000, 100, 600, 600),
	}
	ct := mustSignedContract(t, f)
	got, err := TotalCostOf(newTxTestClient(t, f), t.Context(), ct, testFrom)
	if err != nil {
		t.Fatalf("TotalCostOf: %v", err)
	}
	if got.Energy == nil {
		t.Fatal("ContractTx total must carry the energy preview")
	}
	// Energy price default latest is 420: 13569 × 420.
	if want := tron.SUN(13569 * 420); got.Energy.TronToBurn != want {
		t.Fatalf("energy burn = %v, want %v", got.Energy.TronToBurn, want)
	}
	if got.Bandwidth == nil || got.Bandwidth.ToBurn != 0 {
		t.Fatalf("bandwidth should be stake-covered: %+v", got.Bandwidth)
	}
	if want := got.Energy.TronToBurn; got.Total != want {
		t.Fatalf("total = %v, want energy-only %v", got.Total, want)
	}
}

func TestTotalCostNativeIsBandwidthOnly(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: richResource(0, 0, 600, 600),
		Account: func(ctx context.Context, in *core.Account) (*core.Account, error) {
			if string(in.GetAddress()) == string(testFrom.Bytes()) {
				return &core.Account{Address: in.GetAddress(), Balance: 10_000_000}, nil
			}
			return &core.Account{Address: in.GetAddress()}, nil
		},
	}
	ntx := mustSignedTransfer(t, f)
	got, err := TotalCostOf(newTxTestClient(t, f), t.Context(), ntx, testFrom)
	if err != nil {
		t.Fatalf("TotalCostOf: %v", err)
	}
	if got.Energy != nil {
		t.Fatal("NativeTx total must not carry energy")
	}
	if want := got.Bandwidth.Burn; got.Total != want {
		t.Fatalf("total = %v, want bandwidth-only %v", got.Total, want)
	}
}

func TestTotalCostCreationAddsFee(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: richResource(0, 0, 0, 0),
		ChainParameters: stdChainParams(),
	}
	missingRecipient(f, 50_000_000)
	ntx := mustSignedTransfer(t, f)
	got, err := TotalCostOf(newTxTestClient(t, f), t.Context(), ntx, testFrom)
	if err != nil {
		t.Fatalf("TotalCostOf: %v", err)
	}
	if want := tron.SUN(100_000 + 1_000_000); got.Total != want {
		t.Fatalf("total = %v, want flat fee + creation %v", got.Total, want)
	}
}

func TestTotalCostUnsignedFailsBeforeSimulate(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ct, err := BuildTriggerSmartContract(cp, t.Context(), testFrom, testTo, []byte{0xa9}, 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := TotalCostOf(cp, t.Context(), ct, testFrom); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Fatalf("unsigned: want tx.invalid_argument, got %v", err)
	}
	if n := f.simulateCalls.Load(); n != 0 {
		t.Fatalf("bandwidth must fail before simulation: %d simulate calls", n)
	}
}

func TestTotalCostUnsignedHintNamesSize(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ntx, err := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	_, err = TotalCostOf(cp, t.Context(), ntx, testFrom)
	if !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Fatalf("want tx.invalid_argument, got %v", err)
	}
	// The Hint (not the table-lookup message) carries the teaching:
	// unsigned size plus the per-signature delta.
	var terr *tron.Error
	if !errors.As(err, &terr) {
		t.Fatalf("want *tron.Error, got %T", err)
	}
	for _, want := range []string{"unsigned at", "sign first", "~65 bytes"} {
		if !strings.Contains(terr.Hint, want) {
			t.Errorf("hint %q should contain %q", terr.Hint, want)
		}
	}
}

func TestTotalCostValidation(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ntx := mustSignedTransfer(t, f)
	if _, err := TotalCostOf(nil, t.Context(), ntx, testFrom); !tron.HasCode(err, tron.CodeChainConnection) {
		t.Errorf("nil cp: want chain.connection, got %v", err)
	}
	if _, err := TotalCostOf(cp, t.Context(), nil, testFrom); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("nil tx: want tx.invalid_argument, got %v", err)
	}
	if _, err := TotalCostOf(cp, t.Context(), ntx, tron.Address{}); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("zero owner: want address.invalid, got %v", err)
	}
}
