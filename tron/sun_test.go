package tron

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTRXWholeUnits(t *testing.T) {
	assert.Equal(t, SUN(1_000_000), TRX(1))
	assert.Equal(t, SUN(-2_000_000), TRX(-2))
}

func TestTRXBoundary(t *testing.T) {
	const maxSafeTRX = math.MaxInt64 / 1_000_000 // 9_223_372_036_854
	assert.Equal(t, SUN(maxSafeTRX*1_000_000), TRX(maxSafeTRX))
	assert.PanicsWithValue(t,
		"tronlib: TRX(9223372036855) overflows SUN (max 9223372036854)",
		func() { TRX(maxSafeTRX + 1) })
	// The guard is symmetric: one TRX below the negative boundary panics too.
	assert.Equal(t, SUN(-maxSafeTRX*1_000_000), TRX(-maxSafeTRX))
	assert.PanicsWithValue(t,
		"tronlib: TRX(-9223372036855) overflows SUN (min -9223372036854)",
		func() { TRX(-maxSafeTRX - 1) })
}

func TestParseTRX(t *testing.T) {
	cases := []struct {
		in      string
		want    SUN
		errCode Code
	}{
		{"1.6", 1_600_000, ""}, {"0.1", 100_000, ""}, {"100.000001", 100_000_001, ""},
		{"1e-6", 1, ""}, {"0.300000", 300_000, ""},
		{"-1.6", -1_600_000, ""}, {"-0.000001", -1, ""}, {"-1e-6", -1, ""},
		// The negative band is bounded by -(MaxInt64) sun, not MinInt64:
		// ParseTRX rejects any |value| > MaxInt64, so the most negative
		// representable TRX string is -9223372036854.775807.
		{"-9223372036854.775807", SUN(math.MinInt64 + 1), ""},
		{"-9223372036854.775808", 0, CodeAmountOverflow},
		{"1.6666666", 0, CodeAmountTooManyDecimals},
		{"-1.6666666", 0, CodeAmountTooManyDecimals},
		{"1e-7", 0, CodeAmountTooManyDecimals},
		{"-1e-7", 0, CodeAmountTooManyDecimals},
		{"+1.6", 0, CodeAmountInvalid},
		{"1,234", 0, CodeAmountInvalid},
		{"", 0, CodeAmountInvalid},
		{"abc", 0, CodeAmountInvalid},
		{"9223372036854775808", 0, CodeAmountOverflow}, // > MaxInt64 sun
	}
	for _, tc := range cases {
		got, err := ParseTRX(tc.in)
		if tc.errCode == "" {
			require.NoError(t, err, tc.in)
			assert.Equal(t, tc.want, got, tc.in)
		} else {
			assert.Error(t, err, tc.in)
			assert.True(t, HasCode(err, tc.errCode), "%s: want code %v, got %v", tc.in, tc.errCode, err)
		}
	}
}

func TestSUNStringRoundTrips(t *testing.T) {
	for _, s := range []string{"1.6", "0.000001", "100.000001", "9223372036.854775"} {
		v, err := ParseTRX(s)
		require.NoError(t, err)
		assert.Equal(t, s, v.String(), "canonical String must round-trip")
	}
}

func TestSUNStringCanonicalForms(t *testing.T) {
	// String() must not add separators and must not print trailing zeros.
	assert.Equal(t, "0", SUN(0).String())
	assert.Equal(t, "2", TRX(2).String())
	assert.Equal(t, "-2", TRX(-2).String())
	assert.Equal(t, "-1.6", SUN(-1_600_000).String())
	assert.Equal(t, "-0.000001", SUN(-1).String())
}

func TestSUNAddOverflowChecked(t *testing.T) {
	a := SUN(math.MaxInt64)
	_, err := a.Add(1)
	assert.True(t, HasCode(err, CodeAmountOverflow))
	_, err = a.Sub(SUN(math.MaxInt64))
	assert.NoError(t, err) // equal values are fine

	// Underflow is checked with the same discipline.
	b := SUN(math.MinInt64)
	_, err = b.Sub(1)
	assert.True(t, HasCode(err, CodeAmountOverflow))
	_, err = b.Add(SUN(math.MaxInt64))
	assert.NoError(t, err)
}

func TestSUNMulOverflowChecked(t *testing.T) {
	got, err := SUN(2).Mul(3)
	require.NoError(t, err)
	assert.Equal(t, SUN(6), got)
	got, err = SUN(1_000_000).Mul(0)
	require.NoError(t, err)
	assert.Equal(t, SUN(0), got)

	// Negative multipliers: every sign combination is checked arithmetic,
	// not a wrap.
	for _, tc := range []struct {
		s    SUN
		n    int64
		want SUN
	}{
		{5, -2, -10},
		{-5, 2, -10},
		{-5, -2, 10},
		{math.MaxInt64, -1, SUN(-math.MaxInt64)},
	} {
		v, err := tc.s.Mul(tc.n)
		require.NoError(t, err, "%d * %d", tc.s, tc.n)
		assert.Equal(t, tc.want, v, "%d * %d", tc.s, tc.n)
	}

	_, err = SUN(math.MaxInt64).Mul(2)
	assert.True(t, HasCode(err, CodeAmountOverflow))
	_, err = SUN(math.MaxInt64).Mul(-2)
	assert.True(t, HasCode(err, CodeAmountOverflow))
	_, err = SUN(math.MinInt64).Mul(2)
	assert.True(t, HasCode(err, CodeAmountOverflow))
	// MinInt64 * -1 = 2^63, one above MaxInt64: the wrap case the naive
	// prod/n != n check misses (MinInt64/-1 division overflows back to MinInt64).
	_, err = SUN(math.MinInt64).Mul(-1)
	assert.True(t, HasCode(err, CodeAmountOverflow))
}

func TestSUNFormatted(t *testing.T) {
	v, _ := ParseTRX("1234567.891234")
	assert.Equal(t, "1,234,567.891234", v.Formatted())
	assert.Equal(t, "-1,234,567.891234", (-v).Formatted())
	assert.Equal(t, "1,000", SUN(1_000_000_000).Formatted())
	assert.Equal(t, "0.000001", SUN(1).Formatted())
}

func TestMustTRX(t *testing.T) {
	assert.Equal(t, SUN(1_600_000), MustTRX("1.6"))
	assert.Panics(t, func() { MustTRX("abc") })
}

func TestParseSUN(t *testing.T) {
	got, err := ParseSUN("123")
	require.NoError(t, err)
	assert.Equal(t, SUN(123), got)
	got, err = ParseSUN("-5")
	require.NoError(t, err)
	assert.Equal(t, SUN(-5), got)

	_, err = ParseSUN("1.5")
	assert.True(t, HasCode(err, CodeAmountInvalid), "ParseSUN takes raw sun only")
	_, err = ParseSUN("1,000")
	assert.True(t, HasCode(err, CodeAmountInvalid))
	_, err = ParseSUN("+1")
	assert.True(t, HasCode(err, CodeAmountInvalid))
	_, err = ParseSUN("9223372036854775808")
	assert.True(t, HasCode(err, CodeAmountOverflow))
}

func TestSUNInt64(t *testing.T) {
	assert.Equal(t, int64(1_000_000), TRX(1).Int64())
	assert.Equal(t, int64(-1), SUN(-1).Int64())
}
