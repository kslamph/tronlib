package token

import (
	"math/big"
	"strings"

	"github.com/kslamph/tronlib/v2/internal/format"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/shopspring/decimal"
)

// Amount is an exact token amount in the token's atomic unit: raw counts
// the smallest on-chain unit, decimals is the token's scale (raw units per
// whole token). The zero value is 0 with 0 decimals, and raw is never nil
// (a nil input normalizes to zero). Amounts are immutable value types —
// Raw() returns a copy, so no mutation of an Amount is expressible.
type Amount struct {
	raw      *big.Int
	decimals uint8
}

// newAmount normalizes a raw count against a scale. A nil raw becomes the
// zero big.Int (raw is never nil — see the type doc).
func newAmount(raw *big.Int, decimals uint8) Amount {
	if raw == nil {
		raw = new(big.Int)
	}
	return Amount{raw: raw, decimals: decimals}
}

// Raw returns a copy of the amount in the token's atomic unit. Mutating
// the returned value cannot affect the Amount.
func (a Amount) Raw() *big.Int {
	if a.raw == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(a.raw)
}

// Decimals returns the token's scale the amount is denominated in.
func (a Amount) Decimals() uint8 {
	return a.decimals
}

// scale is 10^decimals as a big.Int (exact big.Int exponentiation — no
// float, and a shift would give 2^decimals, not 10^decimals).
func scale(decimals uint8) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
}

// String returns the canonical decimal form of the amount in whole tokens:
// no separators, no rounding, no trailing zeros. It round-trips through
// Handle.Amount. Unlike tron.SUN's fixed 6-decimal format, a token's scale
// is arbitrary (0..255), so the fraction is rendered with exact big.Int
// division against 10^decimals.
func (a Amount) String() string {
	raw := a.Raw() // a copy — safe to split below
	if a.decimals == 0 {
		return raw.String()
	}
	neg := raw.Sign() < 0
	if neg {
		raw.Neg(raw)
	}
	intPart, frac := new(big.Int).QuoRem(raw, scale(a.decimals), new(big.Int))
	str := intPart.String()
	if frac.Sign() != 0 {
		fs := frac.String()
		// left-pad the fraction to exactly decimals digits, then trim the
		// trailing zeros the canonical form forbids.
		fs = strings.Repeat("0", int(a.decimals)-len(fs)) + fs
		fs = strings.TrimRight(fs, "0")
		str += "." + fs
	}
	if neg {
		str = "-" + str
	}
	return str
}

// Formatted returns a display form of the amount in whole tokens: thousands
// separators on the integer part, exact decimals (no rounding). It mirrors
// tron.SUN.Formatted. The zero value renders "0".
// Display only; parse with Handle.Amount, not Formatted — String() is the
// canonical round-trip form.
func (a Amount) Formatted() string {
	str := a.String()
	neg := strings.HasPrefix(str, "-")
	if neg {
		str = str[1:]
	}
	intPart, frac, _ := strings.Cut(str, ".")
	intPart = format.Thousands(intPart)
	if frac != "" {
		intPart += "." + frac
	}
	if neg {
		intPart = "-" + intPart
	}
	return intPart
}

// parseAmount parses s against the given scale with the same exact-decimal
// rules as tron.ParseTRX, parameterized by decimals: plain decimal strings
// only ("+", ",", "_" are rejected outright), no rounding — more
// fractional digits than the scale is amount.too_many_decimals, and a
// negative value is amount.invalid (token amounts are non-negative).
// No float ever touches this path: shopspring/decimal is exact.
func parseAmount(s string, decimals uint8, op string) (Amount, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "+,_") {
		return Amount{}, &tron.Error{Code: tron.CodeAmountInvalid, Op: op,
			Hint: `pass a plain decimal string like "1.6"`}
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Amount{}, &tron.Error{Code: tron.CodeAmountInvalid, Op: op,
			Hint: `pass a plain decimal string like "1.6"`, Cause: err}
	}
	if d.Sign() < 0 {
		return Amount{}, &tron.Error{Code: tron.CodeAmountInvalid, Op: op,
			Hint: "a token amount cannot be negative; pass a positive decimal string"}
	}
	scaled := d.Mul(decimal.New(1, int32(decimals))) // shift the decimal point by the token's scale
	if !scaled.IsInteger() {
		return Amount{}, &tron.Error{Code: tron.CodeAmountTooManyDecimals, Op: op,
			Hint: "more fractional digits than the token's decimals; this library rounds nothing"}
	}
	raw := scaled.BigInt()
	if raw.BitLen() > 256 {
		return Amount{}, &tron.Error{Code: tron.CodeAmountOverflow, Op: op,
			Hint: "the amount exceeds the uint256 the TRC-20 ABI carries"}
	}
	return newAmount(raw, decimals), nil
}
