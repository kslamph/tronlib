package tx

// Tests for the energy price history parse behind CostPreview: latestEnergyPrice
// (via the exported EnergyPriceOf) and the preview that consumes it.
//
// The parse is the only place a node answer becomes a price, so every shape the
// node could answer is pinned here: the documented "timestamp:price" comma-list,
// the tie-breaking rule (greatest timestamp wins, ties keep the last entry), and
// contract.bad_metadata for anything a silent zero or negative would smuggle past
// (energy.go: a zero price understates every cost, and Client.EnergyPrice caches
// the result for a maintenance period).

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

func pricePrices(prices string) func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	return func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: prices}, nil
	}
}

// previewFixture builds a contract transaction on a fake whose energy-price
// read answers with prices, so a preview runs the documented three reads and
// stops (or does not) at read 3.
func previewFixture(t *testing.T, prices string) (*ContractTx, *rpc.Client) {
	t.Helper()
	f := &fakeWalletServer{EnergyPrices: pricePrices(prices)}
	cp := newTxTestClient(t, f)
	ctxTx, err := BuildTriggerSmartContract(t.Context(), cp, testFrom, testTo, nil, 0)
	if err != nil {
		t.Fatalf("BuildTriggerSmartContract: %v", err)
	}
	return ctxTx, cp
}

// TestEnergyPriceOfLatestWins: the entry with the greatest timestamp wins, not
// the last entry in the list, and both fields carry the winning entry's values.
func TestEnergyPriceOfLatestWins(t *testing.T) {
	f := &fakeWalletServer{EnergyPrices: pricePrices("1691500000000:420,1691400000000:410")}
	p, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
	if err != nil {
		t.Fatalf("EnergyPriceOf: %v", err)
	}
	if p.SunPerEnergy != 420 {
		t.Fatalf("SunPerEnergy = %d, want 420 (greatest timestamp wins)", p.SunPerEnergy)
	}
	if want := time.UnixMilli(1691500000000); !p.EffectiveAt.Equal(want) {
		t.Fatalf("EffectiveAt = %s, want %s", p.EffectiveAt, want)
	}
	if p.FetchedAt.IsZero() {
		t.Fatal("FetchedAt must be set")
	}
}

// TestEnergyPriceOfTieKeepsLast pins the ordering rule: equal timestamps are
// resolved by list position, so the LAST entry wins (the node appends on each
// governance vote, so a re-vote at the same millisecond is the newer price).
func TestEnergyPriceOfTieKeepsLast(t *testing.T) {
	f := &fakeWalletServer{EnergyPrices: pricePrices("1691500000000:420,1691500000000:430")}
	p, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
	if err != nil {
		t.Fatalf("EnergyPriceOf: %v", err)
	}
	if p.SunPerEnergy != 430 {
		t.Fatalf("SunPerEnergy = %d, want 430 (ties keep the last entry)", p.SunPerEnergy)
	}
}

// TestEnergyPriceOfSeparatorOnlyIsBadMetadata is the regression: empty entries
// are skipped, so ",," parsed zero entries and the old guard — which tested the
// length of the ORIGINAL string — let it through as a silent price of 0 at
// timestamp 0. That zero priced every burn at nothing and was cached for six
// hours by Client.EnergyPrice.
func TestEnergyPriceOfSeparatorOnlyIsBadMetadata(t *testing.T) {
	f := &fakeWalletServer{EnergyPrices: pricePrices(",,")}
	_, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
	if !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Fatalf(`prices ",,": want contract.bad_metadata, got %v`, err)
	}
}

// TestEnergyPriceOfWhitespaceOnlyIsBadMetadata: same hole, different filler —
// " " has length 1, so it was not "empty", and its single entry was skipped.
func TestEnergyPriceOfWhitespaceOnlyIsBadMetadata(t *testing.T) {
	f := &fakeWalletServer{EnergyPrices: pricePrices(" ")}
	_, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
	if !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Fatalf(`prices " ": want contract.bad_metadata, got %v`, err)
	}
}

// TestEnergyPriceOfPaddedBlankEntriesIsBadMetadata pins that padding does not
// launder a blank list either.
func TestEnergyPriceOfPaddedBlankEntriesIsBadMetadata(t *testing.T) {
	f := &fakeWalletServer{EnergyPrices: pricePrices(" , , ")}
	_, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
	if !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Fatalf(`prices " , , ": want contract.bad_metadata, got %v`, err)
	}
}

// TestEnergyPriceOfNegativePriceIsBadMetadata is the second regression: a
// negative SUN-per-energy price has no meaning (the node stores governance
// values as positive integers) and was accepted silently, producing a negative
// TronToBurn that passes every fee-limit floor check.
func TestEnergyPriceOfNegativePriceIsBadMetadata(t *testing.T) {
	f := &fakeWalletServer{EnergyPrices: pricePrices("1691400000000:410,1691500000000:-5")}
	_, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
	if !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Fatalf("negative winning price: want contract.bad_metadata, got %v", err)
	}
}

// TestEnergyPriceOfNegativeTimestampIsBadMetadata: a pre-epoch effective time is
// outside the documented shape. It also has to be rejected rather than skipped,
// because the greatest-timestamp rule compares against a zero baseline — an
// all-negative list would otherwise select nothing and return price 0.
func TestEnergyPriceOfNegativeTimestampIsBadMetadata(t *testing.T) {
	f := &fakeWalletServer{EnergyPrices: pricePrices("-1:420")}
	_, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
	if !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Fatalf("negative timestamp: want contract.bad_metadata, got %v", err)
	}
}

// TestEnergyPriceOfZeroPriceAllowed pins what the validation must NOT reject: a
// price of 0 is a legitimate governance value (an operator can set energy free),
// and the epoch timestamp 0 appears in real bandwidth history. Only negative or
// unparseable values are bad_metadata.
func TestEnergyPriceOfZeroPriceAllowed(t *testing.T) {
	f := &fakeWalletServer{EnergyPrices: pricePrices("0:0")}
	p, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
	if err != nil {
		t.Fatalf("EnergyPriceOf(0:0): %v", err)
	}
	if p.SunPerEnergy != 0 || !p.EffectiveAt.Equal(time.UnixMilli(0)) {
		t.Fatalf("zero entry = %+v, want price 0 effective at the epoch", p)
	}
}

// TestEnergyPriceOfTrailingSeparatorAllowed: a trailing comma is cosmetic, not
// malformed, as long as at least one real entry parses.
func TestEnergyPriceOfTrailingSeparatorAllowed(t *testing.T) {
	f := &fakeWalletServer{EnergyPrices: pricePrices("1691400000000:410,1691500000000:420,")}
	p, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
	if err != nil {
		t.Fatalf("EnergyPriceOf with trailing separator: %v", err)
	}
	if p.SunPerEnergy != 420 {
		t.Fatalf("SunPerEnergy = %d, want 420", p.SunPerEnergy)
	}
}

// TestEnergyPriceOfMalformedStillBadMetadata keeps the pre-existing rejections
// in force alongside the new ones.
func TestEnergyPriceOfMalformedStillBadMetadata(t *testing.T) {
	for _, prices := range []string{"", "no-colon", "abc:def", "1691500000000:"} {
		f := &fakeWalletServer{EnergyPrices: pricePrices(prices)}
		_, err := EnergyPriceOf(t.Context(), newTxTestClient(t, f))
		if !tron.HasCode(err, tron.CodeContractBadMetadata) {
			t.Fatalf("prices %q: want contract.bad_metadata, got %v", prices, err)
		}
	}
}

// TestPreviewCostSeparatorOnlyPricesIsBadMetadata proves the fix reaches the
// cost path: the same ",," answer that used to produce a preview with
// TronToBurn 0 now fails the preview outright.
func TestPreviewCostSeparatorOnlyPricesIsBadMetadata(t *testing.T) {
	ctxTx, cp := previewFixture(t, ",,")
	_, err := PreviewCost(t.Context(), cp, ctxTx, testFrom)
	if !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Fatalf(`preview with prices ",,": want contract.bad_metadata, got %v`, err)
	}
}

// TestPreviewCostNegativePriceIsBadMetadata: a negative price must not surface
// as a negative TronToBurn that clears the §6.4 floor check vacuously.
func TestPreviewCostNegativePriceIsBadMetadata(t *testing.T) {
	ctxTx, cp := previewFixture(t, "1691500000000:-5")
	_, err := PreviewCost(t.Context(), cp, ctxTx, testFrom)
	if !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Fatalf("preview with negative price: want contract.bad_metadata, got %v", err)
	}
}
