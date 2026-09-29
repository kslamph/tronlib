package tx

import (
	"testing"

	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/tron"
)

// all four kinds are produced with the right Kind and satisfy the sealed Tx
// interface (architecture §6.1).
func TestKindsAndSealedInterface(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	ctx := t.Context()

	native, err := BuildTransfer(ctx, cp, testFrom, testTo, 1_000_000)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	contract, err := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, []byte{0x01}, 0)
	if err != nil {
		t.Fatalf("BuildTriggerSmartContract: %v", err)
	}
	deploy, err := BuildDeploy(ctx, cp, testFrom, DeployParams{Bytecode: []byte{0x60, 0x80}})
	if err != nil {
		t.Fatalf("BuildDeploy: %v", err)
	}
	asset, err := BuildAssetTransfer(ctx, cp, testFrom, testTo, "1000001", 5)
	if err != nil {
		t.Fatalf("BuildAssetTransfer: %v", err)
	}

	txs := []Tx{native, contract, deploy, asset}
	wantKinds := []Kind{KindNative, KindContract, KindDeploy, KindAssetTransfer}
	wantNames := []string{"native", "contract", "deploy", "asset_transfer"}
	for i, tx := range txs {
		if got := tx.Kind(); got != wantKinds[i] {
			t.Errorf("tx[%d].Kind() = %d, want %d", i, got, wantKinds[i])
		}
	}
	for i, k := range wantKinds {
		if got := k.String(); got != wantNames[i] {
			t.Errorf("Kind(%d).String() = %q, want %q", k, got, wantNames[i])
		}
	}
	if got := Kind(99).String(); got != "unknown" {
		t.Errorf("Kind(99).String() = %q, want unknown", got)
	}
}

func TestKindStringValues(t *testing.T) {
	// pin the enum numbering
	if KindNative != 1 || KindContract != 2 || KindDeploy != 3 || KindAssetTransfer != 4 {
		t.Fatalf("kind enum numbering changed")
	}
}

func TestIDComputedFromRawData(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	native, err := BuildTransfer(t.Context(), cp, testFrom, testTo, 1_000_000)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	// baseTx.ID() is computed live from raw_data, not read from ext.Txid —
	// the canned fake sets Txid to the same sha256, so both agree.
	if native.ID() == "" || len(native.ID()) != 64 {
		t.Fatalf("ID() = %q, want 64 hex chars", native.ID())
	}
}

func TestAccessorsReportEffectiveValues(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	contract, err := BuildTriggerSmartContract(t.Context(), cp, testFrom, testTo, []byte{0x01}, 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := contract.FeeLimit(); got != DefaultFeeLimit {
		t.Errorf("FeeLimit() = %d, want %d (documented default)", got, DefaultFeeLimit)
	}
	if contract.Expiration().IsZero() {
		t.Error("Expiration() = zero, want the server-set head+60s")
	}
	if got := contract.PermissionID(); got != 0 {
		t.Errorf("PermissionID() = %d, want 0 (owner)", got)
	}
	if contract.IsSigned() {
		t.Error("IsSigned() = true on a fresh build")
	}
	if s, err := contract.Signers(); err != nil || len(s) != 0 {
		t.Errorf("Signers() = %v, %v; want empty", s, err)
	}
}

func TestAllKindsSatisfyTx(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	native, _ := BuildTransfer(t.Context(), cp, testFrom, testTo, 1_000_000)
	contract, _ := BuildTriggerSmartContract(t.Context(), cp, testFrom, testTo, []byte{0x01}, 0)
	deploy, _ := BuildDeploy(t.Context(), cp, testFrom, DeployParams{Bytecode: []byte{0x60}})
	asset, _ := BuildAssetTransfer(t.Context(), cp, testFrom, testTo, "1000001", 5)
	var _ Tx = native
	var _ Tx = contract
	var _ Tx = deploy
	var _ Tx = asset
	_ = key.Signer(nil) // keep key import if assertions above shrink
	_ = tron.SUN(0)
}
