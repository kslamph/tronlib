package tx

// Tests for the bandwidth cost path: BandwidthSize (pure size),
// BandwidthPriceOf (price history), and BandwidthCostOf (full prediction
// with the staked → free → burn order and the account-creation branch).
//
// The charging order and formulas mirror java-tron's
// BandwidthProcessor.consume; the numbers below are hand-computed from
// those rules, and the live record (Nile TRC-20 transfer: 345 bytes →
// NetFee 345,000 at 1000 sun/byte) anchors the price shape.

import (
	"context"
	"errors"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/protobuf/proto"
)

func mustSignedTransfer(t *testing.T, f *fakeWalletServer) *NativeTx {
	t.Helper()
	ntx, _ := signedNative(t, f)
	return ntx
}

func TestBandwidthSizeCountsSignedBytesPlusOverhead(t *testing.T) {
	f := &fakeWalletServer{}
	ntx := mustSignedTransfer(t, f)
	got, err := BandwidthSize(ntx.Transaction())
	if err != nil {
		t.Fatalf("BandwidthSize: %v", err)
	}
	cleared := proto.Clone(ntx.Transaction()).(*core.Transaction)
	cleared.Ret = nil
	want := int64(proto.Size(cleared)) + ResultSizePerContract
	if got != want {
		t.Fatalf("BandwidthSize = %d, want size+64 = %d", got, want)
	}
	if got <= 64 {
		t.Fatalf("BandwidthSize = %d: a signed transfer must exceed the bare overhead", got)
	}
}

func TestBandwidthSizeRejectsUnsigned(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	ntx, err := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := BandwidthSize(ntx.Transaction()); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Fatalf("unsigned: want tx.invalid_argument, got %v", err)
	}
	if _, err := BandwidthSize(nil); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Fatalf("nil: want tx.invalid_argument, got %v", err)
	}
}

func TestBandwidthPriceOfLatestWins(t *testing.T) {
	f := &fakeWalletServer{
		BandwidthPrices: func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
			return &api.PricesResponseMessage{Prices: "0:10,1627279200000:1000"}, nil
		},
	}
	p, err := BandwidthPriceOf(newTxTestClient(t, f), t.Context())
	if err != nil {
		t.Fatalf("BandwidthPriceOf: %v", err)
	}
	if p.SunPerByte != 1000 {
		t.Fatalf("SunPerByte = %d, want 1000", p.SunPerByte)
	}
}

func TestBandwidthPriceOfMalformed(t *testing.T) {
	for _, prices := range []string{"no-colon", "", "abc:def"} {
		f := &fakeWalletServer{
			BandwidthPrices: func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
				return &api.PricesResponseMessage{Prices: prices}, nil
			},
		}
		if _, err := BandwidthPriceOf(newTxTestClient(t, f), t.Context()); !tron.HasCode(err, tron.CodeContractBadMetadata) {
			t.Fatalf("prices %q: want contract.bad_metadata, got %v", prices, err)
		}
	}
}

func richResource(stakedLimit, stakedUsed, freeLimit, freeUsed int64) func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
	return func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		return &api.AccountResourceMessage{
			NetLimit: stakedLimit, NetUsed: stakedUsed,
			FreeNetLimit: freeLimit, FreeNetUsed: freeUsed,
		}, nil
	}
}

func TestBandwidthCostCoveredByStake(t *testing.T) {
	f := &fakeWalletServer{AccountResource: richResource(100000, 100, 600, 600)}
	ntx := mustSignedTransfer(t, f)
	cost, err := BandwidthCostOf(newTxTestClient(t, f), t.Context(), ntx, testFrom)
	if err != nil {
		t.Fatalf("BandwidthCostOf: %v", err)
	}
	if cost.ToBurn != 0 || cost.Burn != 0 {
		t.Fatalf("covered call must not burn: %+v", cost)
	}
	if cost.NetUsage != cost.BytesNeeded {
		t.Fatalf("NetUsage = %d, want BytesNeeded %d", cost.NetUsage, cost.BytesNeeded)
	}
	if cost.CreatesAccount || cost.NewAccountFee != 0 {
		t.Fatalf("existing recipient must not create: %+v", cost)
	}
}

func TestBandwidthCostCoveredByFree(t *testing.T) {
	f := &fakeWalletServer{AccountResource: richResource(0, 0, 600, 0)}
	ntx := mustSignedTransfer(t, f)
	cost, err := BandwidthCostOf(newTxTestClient(t, f), t.Context(), ntx, testFrom)
	if err != nil {
		t.Fatalf("BandwidthCostOf: %v", err)
	}
	if cost.BytesNeeded > 600 {
		t.Skipf("signed transfer is %d bytes — exceeds the free quota, adjust the fixture", cost.BytesNeeded)
	}
	if cost.ToBurn != 0 {
		t.Fatalf("free quota should cover %d bytes: %+v", cost.BytesNeeded, cost)
	}
}

func TestBandwidthCostBurnsShortfall(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: richResource(0, 0, 600, 600),
		Account: func(ctx context.Context, in *core.Account) (*core.Account, error) {
			// Recipient exists (echoed address); owner holds 10 TRX.
			if string(in.GetAddress()) == string(testFrom.Bytes()) {
				return &core.Account{Address: in.GetAddress(), Balance: 10_000_000}, nil
			}
			return &core.Account{Address: in.GetAddress()}, nil
		},
	}
	ntx := mustSignedTransfer(t, f)
	cost, err := BandwidthCostOf(newTxTestClient(t, f), t.Context(), ntx, testFrom)
	if err != nil {
		t.Fatalf("BandwidthCostOf: %v", err)
	}
	if cost.ToBurn != cost.BytesNeeded {
		t.Fatalf("ToBurn = %d, want full %d bytes", cost.ToBurn, cost.BytesNeeded)
	}
	if int64(cost.Burn) != cost.BytesNeeded*1000 {
		t.Fatalf("Burn = %v, want %d bytes × 1000", cost.Burn, cost.BytesNeeded)
	}
	if cost.NetUsage != 0 {
		t.Fatalf("burn path reports NetUsage 0, got %d", cost.NetUsage)
	}
}

func TestBandwidthCostInsufficientBalance(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: richResource(0, 0, 600, 600),
		Account: func(ctx context.Context, in *core.Account) (*core.Account, error) {
			return &core.Account{Address: in.GetAddress()}, nil // zero balance everywhere
		},
	}
	ntx := mustSignedTransfer(t, f)
	_, err := BandwidthCostOf(newTxTestClient(t, f), t.Context(), ntx, testFrom)
	if !tron.HasCode(err, tron.CodeAccountInsufficientBandwidth) {
		t.Fatalf("want account.insufficient_bandwidth, got %v", err)
	}
	var terr *tron.Error
	if !errors.As(err, &terr) || terr.Next != tron.ActionFund {
		t.Fatalf("want Next=ActionFund, got %v", err)
	}
}

func stdChainParams() func(ctx context.Context, in *api.EmptyMessage) (*core.ChainParameters, error) {
	return func(ctx context.Context, in *api.EmptyMessage) (*core.ChainParameters, error) {
		return &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
			{Key: "getCreateNewAccountFeeInSystemContract", Value: 1_000_000},
			{Key: "getCreateAccountFee", Value: 100_000},
			{Key: "getCreateNewAccountBandwidthRate", Value: 1},
		}}, nil
	}
}

func missingRecipient(f *fakeWalletServer, ownerBalance int64) {
	f.Account = func(ctx context.Context, in *core.Account) (*core.Account, error) {
		if string(in.GetAddress()) == string(testFrom.Bytes()) {
			return &core.Account{Address: in.GetAddress(), Balance: ownerBalance}, nil
		}
		return &core.Account{}, nil // recipient: no account
	}
}

func TestBandwidthCostCreationCoveredByStake(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: richResource(100000, 0, 600, 600), // free exhausted, stake covers
		ChainParameters: stdChainParams(),
	}
	missingRecipient(f, 50_000_000)
	ntx := mustSignedTransfer(t, f)
	cost, err := BandwidthCostOf(newTxTestClient(t, f), t.Context(), ntx, testFrom)
	if err != nil {
		t.Fatalf("BandwidthCostOf: %v", err)
	}
	if !cost.CreatesAccount {
		t.Fatal("missing recipient must set CreatesAccount")
	}
	if cost.NewAccountFee != tron.SUN(1_000_000) {
		t.Fatalf("NewAccountFee = %v, want 1 TRX", cost.NewAccountFee)
	}
	if cost.Burn != 0 || cost.ToBurn != 0 {
		t.Fatalf("staked creation must not burn: %+v", cost)
	}
	if cost.NetUsage != cost.BytesNeeded {
		t.Fatalf("NetUsage = %d, want ratio-scaled %d (rate 1)", cost.NetUsage, cost.BytesNeeded)
	}
}

func TestBandwidthCostCreationFeeBranch(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: richResource(0, 0, 0, 0), // no stake, no free
		ChainParameters: stdChainParams(),
	}
	missingRecipient(f, 50_000_000)
	ntx := mustSignedTransfer(t, f)
	cost, err := BandwidthCostOf(newTxTestClient(t, f), t.Context(), ntx, testFrom)
	if err != nil {
		t.Fatalf("BandwidthCostOf: %v", err)
	}
	if cost.Burn != tron.SUN(100_000) {
		t.Fatalf("Burn = %v, want flat 100,000 creation fee", cost.Burn)
	}
	if cost.NewAccountFee != tron.SUN(1_000_000) {
		t.Fatalf("NewAccountFee = %v, want 1 TRX", cost.NewAccountFee)
	}
	if cost.NetUsage != 0 {
		t.Fatalf("fee branch reports NetUsage 0, got %d", cost.NetUsage)
	}
}

func TestBandwidthCostCreationShortBalance(t *testing.T) {
	f := &fakeWalletServer{
		AccountResource: richResource(0, 0, 0, 0),
		ChainParameters: stdChainParams(),
	}
	missingRecipient(f, 500_000) // covers the 100k fee but not the 1M creation
	ntx := mustSignedTransfer(t, f)
	if _, err := BandwidthCostOf(newTxTestClient(t, f), t.Context(), ntx, testFrom); !tron.HasCode(err, tron.CodeAccountInsufficientBandwidth) {
		t.Fatalf("want account.insufficient_bandwidth, got %v", err)
	}
}

func TestTransferRecipientShapes(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()

	ntx, err := BuildTransfer(cp, ctx, testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	to, ok := transferRecipient(ntx)
	if !ok || to != testTo {
		t.Fatalf("NativeTx recipient = %v, %v; want %v, true", to, ok, testTo)
	}
}
