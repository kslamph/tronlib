package token

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"

	"github.com/kslamph/tronlib/v2/tron"
)

// TestAmountRawReturnsCopy: Raw() hands back an isolated copy — mutating
// the returned *big.Int must not change the Amount (immutable amounts).
func TestAmountRawReturnsCopy(t *testing.T) {
	a := newAmount(big.NewInt(1_600_000), 6)
	got := a.Raw()
	if got.Cmp(big.NewInt(1_600_000)) != 0 {
		t.Fatalf("Raw() = %v, want 1600000", got)
	}
	got.SetInt64(999)
	if again := a.Raw(); again.Int64() != 1_600_000 {
		t.Errorf("after mutating the first Raw() result, Raw() = %v, want 1600000 (the Amount must be unaffected)", again)
	}
}

// TestAmountZeroValue: the zero Amount is 0 with 0 decimals and Raw() on it
// is non-nil (raw is never nil).
func TestAmountZeroValue(t *testing.T) {
	var a Amount
	if a.Decimals() != 0 {
		t.Errorf("zero Amount Decimals() = %d, want 0", a.Decimals())
	}
	raw := a.Raw()
	if raw == nil || raw.Sign() != 0 {
		t.Errorf("zero Amount Raw() = %v, want non-nil 0", raw)
	}
	if got := a.String(); got != "0" {
		t.Errorf("zero Amount String() = %q, want \"0\"", got)
	}
}

// TestAmountStringCanonicalRoundTrip: String() is the canonical decimal
// form and round-trips through newAmount — including zero-padded fraction
// forms (1500000 raw @6 -> "1.5", not "1.500000") and 0-decimal tokens.
func TestAmountStringCanonicalRoundTrip(t *testing.T) {
	cases := []struct {
		raw      int64
		decimals uint8
		want     string
	}{
		{1_600_000, 6, "1.6"},
		{1_500_000, 6, "1.5"},
		{1, 6, "0.000001"},
		{1_234_567, 6, "1.234567"},
		{0, 6, "0"},
		{100, 0, "100"},
		{123, 2, "1.23"},
		{1_000_000_000, 9, "1"},
	}
	for _, tc := range cases {
		a := newAmount(big.NewInt(tc.raw), tc.decimals)
		if got := a.String(); got != tc.want {
			t.Errorf("newAmount(%d, %d).String() = %q, want %q", tc.raw, tc.decimals, got, tc.want)
		}
	}
}

// TestStringRoundTripsThroughHandle: for every Amount produced by
// h.Amount(s), h.Amount(a.String()) yields the same raw value — the
// canonical form is loss-free.
func TestStringRoundTripsThroughHandle(t *testing.T) {
	h, _ := newTestHandle(t)
	for _, s := range []string{"0", "1.6", "0.000001", "123.456789", "999999999999"} {
		a, err := h.Amount(s)
		if err != nil {
			t.Fatalf("Amount(%q): %v", s, err)
		}
		again, err := h.Amount(a.String())
		if err != nil {
			t.Fatalf("Amount(String()) of %q: %v", s, err)
		}
		if again.Raw().Cmp(a.Raw()) != 0 {
			t.Errorf("round-trip of %q: raw %v != %v", s, again.Raw(), a.Raw())
		}
	}
}

// TestWhole: Whole(3) on a 6-decimal handle is 3000000 raw. The concrete
// int64 parameter preserves the amount-safety property: float literals and
// foreign numeric types (tron.SUN, float64) are compile errors.
func TestWhole(t *testing.T) {
	h, _ := newTestHandle(t)
	got, err := h.Whole(3)
	if err != nil || got.Raw().Int64() != 3_000_000 || got.Decimals() != 6 {
		t.Errorf("Whole(3) = (%v, %d, %v), want (3000000, 6, nil)", got.Raw(), got.Decimals(), err)
	}
	got, err = h.Whole(int64(2))
	if err != nil || got.Raw().Int64() != 2_000_000 {
		t.Errorf("Whole(int64(2)) = (%v, %v), want (2000000, nil)", got.Raw(), err)
	}
	// zero-value handle (0 decimals): Whole(7) is 7.
	var zero Handle
	got, err = zero.Whole(7)
	if err != nil || got.Raw().Int64() != 7 || got.Decimals() != 0 {
		t.Errorf("zero-handle Whole(7) = (%v, %d, %v), want (7, 0, nil)", got.Raw(), got.Decimals(), err)
	}
	// negative is amount.invalid.
	if _, err := h.Whole(-1); !tron.HasCode(err, tron.CodeAmountInvalid) {
		t.Errorf("Whole(-1): err = %v, want amount.invalid", err)
	}
}

// TestWholeOverflow: a whole-token count whose scaled value exceeds the
// uint256 the TRC-20 ABI carries is amount.overflow. uint256's max is
// ~1.16e77, so at 76 decimals 1 whole token (10^76) fits while 100
// (10^78) does not; at 255 decimals even 1 whole token overflows.
func TestWholeOverflow(t *testing.T) {
	newHandleWithDecimals := func(d int64) *Handle {
		f := &fakeTRC20Wallet{
			TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
				ext := okExt()
				if string(in.GetData()[:4]) == string(mustSelector("decimals()")) {
					ext.ConstantResult = [][]byte{abiWord(d)}
				}
				return ext, nil
			},
		}
		h, err := New(newTokenTestClient(t, f), context.Background(), testContract)
		if err != nil {
			t.Fatalf("New with decimals=%d: %v", d, err)
		}
		return h
	}
	h := newHandleWithDecimals(76)
	if _, err := h.Whole(1); err != nil { // 10^76 fits uint256
		t.Errorf("Whole(1) at 76 decimals: %v", err)
	}
	if _, err := h.Whole(100); !tron.HasCode(err, tron.CodeAmountOverflow) { // 10^78 overflows
		t.Errorf("Whole(100) at 76 decimals: err = %v, want amount.overflow", err)
	}
	extreme := newHandleWithDecimals(255)
	if _, err := extreme.Whole(1); !tron.HasCode(err, tron.CodeAmountOverflow) { // 10^255 overflows
		t.Errorf("Whole(1) at 255 decimals: err = %v, want amount.overflow", err)
	}
}

// TestAmountParse: "1.6" against a 6-decimal handle parses to 1600000 raw.
func TestAmountParse(t *testing.T) {
	h, _ := newTestHandle(t)
	a, err := h.Amount("1.6")
	if err != nil {
		t.Fatalf("Amount(\"1.6\"): %v", err)
	}
	if a.Raw().Int64() != 1_600_000 {
		t.Errorf("Amount(\"1.6\").Raw() = %v, want 1600000", a.Raw())
	}
	if a.Decimals() != 6 {
		t.Errorf("Amount(\"1.6\").Decimals() = %d, want 6", a.Decimals())
	}
}

// TestAmountTooManyDecimals: more fractional digits than the token's
// decimals is amount.too_many_decimals — exact, no rounding.
func TestAmountTooManyDecimals(t *testing.T) {
	h, _ := newTestHandle(t)
	_, err := h.Amount("1.6666666") // 7 places on a 6-decimal token
	if !tron.HasCode(err, tron.CodeAmountTooManyDecimals) {
		t.Errorf("Amount(\"1.6666666\") on 6 decimals: err = %v, want amount.too_many_decimals", err)
	}
}

// TestAmountParseInvalid: garbage and scientific notation rejections —
// the parse path mirrors ParseTRX's rules.
func TestAmountParseInvalid(t *testing.T) {
	h, _ := newTestHandle(t)
	for _, s := range []string{"", "abc", "1,5", "+1.5", "  "} {
		if _, err := h.Amount(s); !tron.HasCode(err, tron.CodeAmountInvalid) {
			t.Errorf("Amount(%q): err = %v, want amount.invalid", s, err)
		}
	}
}

// TestAmountNegativeRejected: a negative amount is amount.invalid —
// token amounts are non-negative (balances and transfers cannot be).
func TestAmountNegativeRejected(t *testing.T) {
	h, _ := newTestHandle(t)
	if _, err := h.Amount("-1"); !tron.HasCode(err, tron.CodeAmountInvalid) {
		t.Errorf("Amount(\"-1\"): err = %v, want amount.invalid", err)
	}
}

// TestAmountAcceptsMaxDecimals: 255 decimals is inside the accepted
// uint8 range (controller finding: rejecting >18 conflates unusual-but-
// valid with malformed). Parsing a whole number still works at that scale.
func TestAmountAcceptsMaxDecimals(t *testing.T) {
	f := &fakeTRC20Wallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExt()
			if string(in.GetData()[:4]) == string(mustSelector("decimals()")) {
				ext.ConstantResult = [][]byte{abiWord(255)}
			}
			return ext, nil
		},
	}
	h, err := New(newTokenTestClient(t, f), context.Background(), testContract)
	if err != nil {
		t.Fatalf("New with decimals=255: %v", err)
	}
	if h.Decimals() != 255 {
		t.Errorf("Decimals() = %d, want 255", h.Decimals())
	}
	smallest := "0." + strings.Repeat("0", 254) + "1" // 10^-255: raw 1
	a, err := h.Amount(smallest)
	if err != nil {
		t.Fatalf("Amount(10^-255): %v", err)
	}
	if a.Decimals() != 255 || a.Raw().Int64() != 1 {
		t.Errorf("Amount(10^-255) = (%v, %d), want (1, 255)", a.Raw(), a.Decimals())
	}
	if got := a.String(); got != smallest {
		t.Errorf("String() = %q, want the parsed form back", got)
	}
	// 5 whole tokens at 255 decimals exceeds uint256 — amount.overflow.
	if _, err := h.Amount("5"); !tron.HasCode(err, tron.CodeAmountOverflow) {
		t.Errorf("Amount(\"5\") at 255 decimals: err = %v, want amount.overflow", err)
	}
}

// TestAmountFormatted pins the display form: thousands separators on the
// integer part, exact decimals, and a nil-safe zero value.
func TestAmountFormatted(t *testing.T) {
	cases := []struct {
		raw      int64
		decimals uint8
		want     string
	}{
		{0, 6, "0"},
		{1_500_000, 6, "1.5"},
		{1_234_567_891, 6, "1,234.567891"},
	}
	for _, c := range cases {
		if got := newAmount(big.NewInt(c.raw), c.decimals).Formatted(); got != c.want {
			t.Errorf("Formatted(raw %d, decimals %d) = %q, want %q", c.raw, c.decimals, got, c.want)
		}
	}

	// The zero value has a nil raw; Formatted must not panic.
	var z Amount
	if got := z.Formatted(); got != "0" {
		t.Errorf("zero Amount.Formatted() = %q, want %q", got, "0")
	}

	// 255 decimals: the smallest unit renders without panic and keeps the
	// exact fraction.
	smallest := "0." + strings.Repeat("0", 254) + "1"
	if got := newAmount(big.NewInt(1), 255).Formatted(); got != smallest {
		t.Errorf("Formatted(1, 255) = %q, want %q", got, smallest)
	}
}
