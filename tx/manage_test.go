package tx

// Tests for the contract management builders: UpdateSetting,
// UpdateEnergyLimit and ClearABI. All three are bandwidth-only NativeTx
// (no energy, no simulation path); validation mirrors v1's Manager.

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

func TestBuildUpdateSetting(t *testing.T) {
	var gotReq *core.UpdateSettingContract
	f := &fakeWalletServer{
		UpdateSettingFn: func(ctx context.Context, in *core.UpdateSettingContract) (*api.TransactionExtention, error) {
			gotReq = in
			return manageExt(), nil
		},
	}
	ntx, err := BuildUpdateSetting(newTxTestClient(t, f), t.Context(), testFrom, testTo, 30)
	if err != nil {
		t.Fatalf("BuildUpdateSetting: %v", err)
	}
	var _ *NativeTx = ntx
	if ntx.Kind() != KindNative {
		t.Fatalf("Kind = %v, want native (no energy, no simulate)", ntx.Kind())
	}
	if string(gotReq.GetOwnerAddress()) != string(testFrom.Bytes()) {
		t.Errorf("owner = %x, want %x", gotReq.GetOwnerAddress(), testFrom.Bytes())
	}
	if string(gotReq.GetContractAddress()) != string(testTo.Bytes()) {
		t.Errorf("contract = %x, want %x", gotReq.GetContractAddress(), testTo.Bytes())
	}
	if gotReq.GetConsumeUserResourcePercent() != 30 {
		t.Errorf("percent = %d, want 30", gotReq.GetConsumeUserResourcePercent())
	}
	// Simulate must not exist on the management kind (the F1 rule extended:
	// no TriggerSmartContract lives here).
	iface := Tx(ntx)
	if _, ok := iface.(*ContractTx); ok {
		t.Fatal("management tx must not be a *ContractTx")
	}
}

func TestBuildUpdateSettingValidation(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	for _, percent := range []int64{-1, 101, 1 << 40} {
		if _, err := BuildUpdateSetting(cp, t.Context(), testFrom, testTo, percent); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
			t.Errorf("percent %d: want tx.invalid_argument, got %v", percent, err)
		}
	}
	// Boundaries are accepted (validation only, no network assertion here —
	// the fake answers manageExt for any percent in range).
	for _, percent := range []int64{0, 100} {
		if _, err := BuildUpdateSetting(cp, t.Context(), testFrom, testTo, percent); err != nil {
			t.Errorf("percent %d: %v", percent, err)
		}
	}
	if _, err := BuildUpdateSetting(cp, t.Context(), tron.Address{}, testTo, 30); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("zero owner: want address.invalid, got %v", err)
	}
}

func TestBuildUpdateEnergyLimit(t *testing.T) {
	var gotReq *core.UpdateEnergyLimitContract
	f := &fakeWalletServer{
		UpdateEnergyFn: func(ctx context.Context, in *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
			gotReq = in
			return manageExt(), nil
		},
	}
	ntx, err := BuildUpdateEnergyLimit(newTxTestClient(t, f), t.Context(), testFrom, testTo, 1_000_000)
	if err != nil {
		t.Fatalf("BuildUpdateEnergyLimit: %v", err)
	}
	if ntx.Kind() != KindNative {
		t.Fatalf("Kind = %v, want native", ntx.Kind())
	}
	if gotReq.GetOriginEnergyLimit() != 1_000_000 {
		t.Errorf("limit = %d, want 1000000", gotReq.GetOriginEnergyLimit())
	}
	if _, err := BuildUpdateEnergyLimit(newTxTestClient(t, f), t.Context(), testFrom, testTo, -1); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("negative limit: want tx.invalid_argument, got %v", err)
	}
}

func TestBuildClearABI(t *testing.T) {
	var gotReq *core.ClearABIContract
	f := &fakeWalletServer{
		ClearABIFn: func(ctx context.Context, in *core.ClearABIContract) (*api.TransactionExtention, error) {
			gotReq = in
			return manageExt(), nil
		},
	}
	ntx, err := BuildClearABI(newTxTestClient(t, f), t.Context(), testFrom, testTo)
	if err != nil {
		t.Fatalf("BuildClearABI: %v", err)
	}
	if ntx.Kind() != KindNative {
		t.Fatalf("Kind = %v, want native", ntx.Kind())
	}
	if string(gotReq.GetContractAddress()) != string(testTo.Bytes()) {
		t.Errorf("contract = %x, want %x", gotReq.GetContractAddress(), testTo.Bytes())
	}
	if _, err := BuildClearABI(newTxTestClient(t, f), t.Context(), testFrom, tron.Address{}); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("zero contract: want address.invalid, got %v", err)
	}
}
