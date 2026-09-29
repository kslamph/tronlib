package tron

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/kslamph/tronlib/v2/internal/format"
	"github.com/shopspring/decimal"
)

// sunPerTRX is the atomic scale of TRX: 1 TRX = 1_000_000 SUN.
const sunPerTRX = 1_000_000

// SUN is a TRX amount in the atomic on-chain unit: 1 TRX = 1_000_000 SUN.
// It is the only in-memory representation of a TRX amount in v2; every
// transaction-construction API takes SUN, never float64 or raw int64, so a
// whole class of unit-confusion money-loss bugs cannot compile.
type SUN int64

// MaxSUN is the largest representable amount.
const MaxSUN = math.MaxInt64

// Whole admits the predeclared signed integer types and nothing else.
//
// Two deliberate exclusions, both verified by execution (architecture §5.1):
//   - no tilde: with ~int64, SUN's own underlying type satisfies the
//     constraint, so TRX(someSUN) compiles and re-scales an already-scaled
//     value, turning 1 TRX into 1,000,000 TRX while the overflow guard passes.
//   - no unsigned: int64(n) of a uint64 above MaxInt64 wraps negative and
//     passes the bound check, yielding a silently negative amount.
//
// v2/internal/compilecheck pins both exclusions as negative-compile tests.
type Whole interface {
	int | int8 | int16 | int32 | int64
}

// maxSafeTRX is the largest whole-TRX value whose SUN scaling fits in int64.
const maxSafeTRX = math.MaxInt64 / sunPerTRX // 9_223_372_036_854

// TRX converts a whole-number TRX literal to SUN. It panics on overflow and
// is for literals and constants only; dynamic input must use ParseTRX.
func TRX[T Whole](n T) SUN {
	v := int64(n)
	if v > maxSafeTRX || v < -maxSafeTRX {
		panic(formatOverflow(v))
	}
	return SUN(v * sunPerTRX)
}

func formatOverflow(v int64) string {
	if v < 0 {
		return fmt.Sprintf("tronlib: TRX(%d) overflows SUN (min %d)", v, -maxSafeTRX)
	}
	return fmt.Sprintf("tronlib: TRX(%d) overflows SUN (max %d)", v, maxSafeTRX)
}

// MustTRX parses s and panics on error. For package-level amount literals in
// tests and examples; anything that handles user input must use ParseTRX.
func MustTRX(s string) SUN {
	v, err := ParseTRX(s)
	if err != nil {
		panic(err)
	}
	return v
}

// ParseTRX parses an exact decimal TRX string into SUN.
// Rejects: unparseable input, a leading '+', thousands separators, more
// than 6 decimal places, and values beyond the int64 SUN range.
// Accepts scientific notation down to 1 SUN ("1e-6").
// No float64 ever touches this path: shopspring/decimal is exact.
func ParseTRX(s string) (SUN, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "+,_") {
		return 0, &Error{Code: CodeAmountInvalid, Op: "ParseTRX",
			Hint: `pass a plain decimal string like "1.6"`}
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return 0, &Error{Code: CodeAmountInvalid, Op: "ParseTRX",
			Hint: `pass a plain decimal string like "1.6"`, Cause: err}
	}
	scaled := d.Mul(decimal.New(1, 6)) // TRX -> SUN
	if !scaled.IsInteger() {
		return 0, &Error{Code: CodeAmountTooManyDecimals, Op: "ParseTRX",
			Hint: "1 SUN = 1e-6 TRX; round to 6 decimal places"}
	}
	if scaled.Abs().GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0, &Error{Code: CodeAmountOverflow, Op: "ParseTRX"}
	}
	return SUN(scaled.IntPart()), nil
}

// ParseSUN parses a raw sun count ("123" means 123 SUN, no scaling).
// Negative values are accepted; malformed input is CodeAmountInvalid and
// out-of-range input is CodeAmountOverflow.
func ParseSUN(s string) (SUN, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "+,_") {
		return 0, &Error{Code: CodeAmountInvalid, Op: "ParseSUN",
			Hint: "pass a plain integer sun count, e.g. \"1000000\" for 1 TRX"}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		code := CodeAmountInvalid
		if errors.Is(err, strconv.ErrRange) {
			code = CodeAmountOverflow
		}
		return 0, &Error{Code: code, Op: "ParseSUN", Cause: err}
	}
	return SUN(n), nil
}

// String returns the canonical decimal TRX form: no separators, no rounding,
// no trailing zeros. It round-trips with ParseTRX. Value receiver.
func (s SUN) String() string {
	neg := s < 0
	u := uint64(s)
	if neg {
		u = -u // two's-complement negate in unsigned space: exact for MinInt64
	}
	intPart, frac := u/sunPerTRX, u%sunPerTRX
	str := strconv.FormatUint(intPart, 10)
	if frac != 0 {
		str += "." + trimFracZeros(frac)
	}
	if neg {
		str = "-" + str
	}
	return str
}

// trimFracZeros renders frac (0 <= frac < 1e6) as its shortest exact
// fraction digits, e.g. 500 -> "0005", 0 is handled by the caller.
func trimFracZeros(frac uint64) string {
	digits := strconv.FormatUint(frac+sunPerTRX, 10)[1:] // zero-pad to 6 digits
	return strings.TrimRight(digits, "0")
}

// Formatted returns a display form: thousands separators on the integer part.
// The fractional digits are the exact canonical ones — SUN is atomic, so a
// value never has more than 6 decimals and rounding is never needed.
// Display only; parse with ParseTRX, not Formatted.
func (s SUN) Formatted() string {
	str := s.String()
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

// Add returns s + o. Overflow and underflow are CodeAmountOverflow; SUN has
// no unchecked arithmetic, so a silent wrap can never produce a wrong amount.
func (s SUN) Add(o SUN) (SUN, error) {
	a, b := int64(s), int64(o)
	if b > 0 && a > math.MaxInt64-b {
		return 0, &Error{Code: CodeAmountOverflow, Op: "SUN.Add"}
	}
	if b < 0 && a < math.MinInt64-b {
		return 0, &Error{Code: CodeAmountOverflow, Op: "SUN.Add"}
	}
	return SUN(a + b), nil
}

// Sub returns s - o, checked like Add.
func (s SUN) Sub(o SUN) (SUN, error) {
	a, b := int64(s), int64(o)
	if b < 0 && a > math.MaxInt64+b {
		return 0, &Error{Code: CodeAmountOverflow, Op: "SUN.Sub"}
	}
	if b > 0 && a < math.MinInt64+b {
		return 0, &Error{Code: CodeAmountOverflow, Op: "SUN.Sub"}
	}
	return SUN(a - b), nil
}

// Mul returns s * n, checked like Add. The MinInt64 * -1 product is 2^63,
// one above MaxInt64; the naive prod/n != s check cannot see it because
// MinInt64 / -1 overflows back to MinInt64, so it is guarded explicitly.
func (s SUN) Mul(n int64) (SUN, error) {
	a, b := int64(s), n
	if a == 0 || b == 0 {
		return 0, nil
	}
	if a == math.MinInt64 && b == -1 {
		return 0, &Error{Code: CodeAmountOverflow, Op: "SUN.Mul"}
	}
	prod := a * b
	if prod/b != a {
		return 0, &Error{Code: CodeAmountOverflow, Op: "SUN.Mul"}
	}
	return SUN(prod), nil
}

// Int64 returns the raw sun count.
func (s SUN) Int64() int64 { return int64(s) }
