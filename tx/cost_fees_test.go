package tx

// Tests for the governance fees in TotalCostOf: a multi-signature transaction
// pays the node's fixed multisig surcharge and an account-permission update
// pays the fixed update fee, both read live from the chain parameters. Before
// this, TotalCost called itself "all-in" while omitting two fees that can be
// two orders of magnitude larger than bandwidth.

import (
	"context"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// chainParamsWith returns a GetChainParameters answer carrying the given
// key/value pairs.
func chainParamsWith(pairs ...[2]any) *core.ChainParameters {
	out := &core.ChainParameters{}
	for _, p := range pairs {
		out.ChainParameter = append(out.ChainParameter, &core.ChainParameters_ChainParameter{
			Key:   p[0].(string),
			Value: int64(p[1].(int)),
		})
	}
	return out
}

// mainnetFees builds the parameter set Mainnet reports today.
func mainnetFees() *core.ChainParameters {
	return chainParamsWith(
		[2]any{"getUpdateAccountPermissionFee", 100_000_000},
		[2]any{"getMultiSignFee", 1_000_000},
		[2]any{"getUnfreezeDelayDays", 14},
		[2]any{"getMaxDelegateLockPeriod", 86_400},
	)
}

func TestChainParamsOfReadsFeesAndBounds(t *testing.T) {
	f := &fakeWalletServer{ChainParameters: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
		return mainnetFees(), nil
	}}
	cp := newTxTestClient(t, f)
	p, err := ChainParamsOf(t.Context(), cp)
	if err != nil {
		t.Fatalf("ChainParamsOf: %v", err)
	}
	if p.UpdateAccountPermissionFee != 100_000_000 || p.MultiSignFee != 1_000_000 {
		t.Fatalf("fees = %d/%d", p.UpdateAccountPermissionFee, p.MultiSignFee)
	}
	if p.UnfreezeDelayDays != 14 || p.MaxDelegateLockPeriod != 86_400 || len(p.Missing) != 0 {
		t.Fatalf("bounds = %d/%d missing %v", p.UnfreezeDelayDays, p.MaxDelegateLockPeriod, p.Missing)
	}
}

func TestChainParamsOfRejectsMissingFees(t *testing.T) {
	// Only the informational bounds: the fee parameters are absent, and a
	// cost model must not price them as zero.
	f := &fakeWalletServer{ChainParameters: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
		return chainParamsWith([2]any{"getUnfreezeDelayDays", 14}), nil
	}}
	cp := newTxTestClient(t, f)
	if _, err := ChainParamsOf(t.Context(), cp); !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("missing fee params err = %v, want contract.bad_metadata", err)
	}
	// An entirely empty answer is the same class.
	f2 := &fakeWalletServer{ChainParameters: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
		return &core.ChainParameters{}, nil
	}}
	if _, err := ChainParamsOf(t.Context(), newTxTestClient(t, f2)); !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("empty params err = %v, want contract.bad_metadata", err)
	}
}

func TestChainParamsOfRecordsMissingBounds(t *testing.T) {
	// Fees present, bounds absent: the read succeeds and records what the
	// node did not say, so a caller can tell "absent" from "zero".
	f := &fakeWalletServer{ChainParameters: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
		return chainParamsWith(
			[2]any{"getMultiSignFee", 1_000_000},
			[2]any{"getUpdateAccountPermissionFee", 100_000_000},
		), nil
	}}
	cp := newTxTestClient(t, f)
	p, err := ChainParamsOf(t.Context(), cp)
	if err != nil {
		t.Fatalf("ChainParamsOf: %v", err)
	}
	if len(p.Missing) != 2 {
		t.Fatalf("Missing = %v, want both bound parameters", p.Missing)
	}
}

func TestTotalCostAddsMultiSignFee(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: richResource(1_000_000, 0, 600, 600),
		ChainParameters: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
			return mainnetFees(), nil
		},
	}
	cp := newTxTestClient(t, f)
	native, err := BuildTransfer(t.Context(), cp, testFrom, testTo, tron.TRX(1))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	// One signature: no surcharge, and no chain-parameter read at all.
	single, err := native.Sign(mustSigner(t, testKeyHex))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	one, err := TotalCostOf(t.Context(), cp, single, testFrom)
	if err != nil {
		t.Fatalf("TotalCostOf(single): %v", err)
	}
	if one.MultiSignFee != 0 || one.ChainParams != nil {
		t.Fatalf("single-signature total must not carry a multisig fee: %+v", one)
	}
	if one.Total != one.Bandwidth.Burn+one.Bandwidth.NewAccountFee {
		t.Fatalf("single-signature total = %d, want bandwidth-only", one.Total)
	}

	// Two signatures: the surcharge applies.
	multi, err := single.Sign(mustSigner(t, testKeyHex2))
	if err != nil {
		t.Fatalf("second sign: %v", err)
	}
	two, err := TotalCostOf(t.Context(), cp, multi, testFrom)
	if err != nil {
		t.Fatalf("TotalCostOf(multi): %v", err)
	}
	if two.MultiSignFee != 1_000_000 {
		t.Fatalf("MultiSignFee = %d, want 1_000_000", two.MultiSignFee)
	}
	if want := one.Total + 1_000_000; two.Total != want {
		t.Fatalf("total = %d, want %d (single + surcharge)", two.Total, want)
	}
	if two.ChainParams == nil || two.ChainParams.MultiSignFee != 1_000_000 {
		t.Fatalf("ChainParams provenance missing: %+v", two.ChainParams)
	}
	// The multisig surcharge is not part of the bandwidth or creation lines.
	if two.Bandwidth.Burn != one.Bandwidth.Burn {
		t.Fatalf("bandwidth burn changed with the signature count: %d vs %d", two.Bandwidth.Burn, one.Bandwidth.Burn)
	}
	if !strings.Contains(two.String(), "multisig") {
		t.Errorf("String() must surface the governance fee: %q", two.String())
	}
}

func TestTotalCostAddsPermissionUpdateFee(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: richResource(1_000_000, 0, 600, 600),
		ChainParameters: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
			return mainnetFees(), nil
		},
	}
	cp := newTxTestClient(t, f)
	bitmap, err := OperationsBitmap(TypeTransfer)
	if err != nil {
		t.Fatalf("OperationsBitmap: %v", err)
	}
	upd, err := BuildAccountPermissionUpdate(t.Context(), cp, testFrom, twoOfThree(t, bitmap))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	signed, err := upd.Sign(mustSigner(t, testKeyHex))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	cost, err := TotalCostOf(t.Context(), cp, signed, testFrom)
	if err != nil {
		t.Fatalf("TotalCostOf: %v", err)
	}
	if cost.PermissionUpdateFee != 100_000_000 {
		t.Fatalf("PermissionUpdateFee = %d, want 100_000_000", cost.PermissionUpdateFee)
	}
	if cost.MultiSignFee != 0 {
		t.Fatalf("one signature must not add the multisig surcharge, got %d", cost.MultiSignFee)
	}
	if want := cost.Bandwidth.Burn + cost.Bandwidth.NewAccountFee + 100_000_000; cost.Total != want {
		t.Fatalf("total = %d, want bandwidth + permission fee %d", cost.Total, want)
	}

	// Both fees together: two signatures on the permission update.
	two, err := signed.Sign(mustSigner(t, testKeyHex2))
	if err != nil {
		t.Fatalf("second sign: %v", err)
	}
	both, err := TotalCostOf(t.Context(), cp, two, testFrom)
	if err != nil {
		t.Fatalf("TotalCostOf(two): %v", err)
	}
	if both.PermissionUpdateFee != 100_000_000 || both.MultiSignFee != 1_000_000 {
		t.Fatalf("fees = %d/%d, want 100_000_000/1_000_000", both.PermissionUpdateFee, both.MultiSignFee)
	}
	if want := cost.Bandwidth.Burn + cost.Bandwidth.NewAccountFee + 101_000_000; both.Total != want {
		t.Fatalf("total = %d, want %d", both.Total, want)
	}
}

func TestTotalCostFailsLoudlyWhenFeesUnreadable(t *testing.T) {
	// A node that omits the fee parameters cannot be priced honestly: the
	// prediction must fail rather than silently report a bandwidth-only total
	// for a multi-signature transaction.
	f := &fakeWalletServer{
		AccountResource: richResource(1_000_000, 0, 600, 600),
		ChainParameters: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
			return &core.ChainParameters{}, nil
		},
	}
	cp := newTxTestClient(t, f)
	native, err := BuildTransfer(t.Context(), cp, testFrom, testTo, tron.TRX(1))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	signed, err := native.Sign(mustSigner(t, testKeyHex))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	multi, err := signed.Sign(mustSigner(t, testKeyHex2))
	if err != nil {
		t.Fatalf("second sign: %v", err)
	}
	if _, err := TotalCostOf(t.Context(), cp, multi, testFrom); !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("total with unreadable fees err = %v, want contract.bad_metadata", err)
	}
}
