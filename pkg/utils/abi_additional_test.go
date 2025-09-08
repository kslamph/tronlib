package utils

import (
	"math/big"
	"testing"
)

func TestHumanReadableNumber(t *testing.T) {
	tests := []struct {
		name     string
		number   any
		decimal  int64
		expected string
		hasError bool
	}{
		// String inputs - these represent raw values that need to be divided by 10^decimal
		{"string positive", "1234567890", 6, "1,234.567890", false},
		{"string negative", "-1234567890", 6, "-1,234.567890", false},
		{"string zero", "0", 6, "0.000000", false},
		{"string zero no decimal", "0", 0, "0", false},
		{"string small number", "123456", 2, "1,234.56", false},
		{"string large number", "999999999999999999", 18, "0.999999999999999999", false},
		{"string with decimal", "1234567", 4, "123.4567", false},
		{"string scientific", "1230000", 2, "12,300.00", false},
		{"empty string", "", 2, "", true},
		{"invalid string", "abc", 2, "", true},

		// *big.Int inputs
		{"bigint positive", new(big.Int).SetInt64(1234567890), 6, "1,234.567890", false},
		{"bigint negative", new(big.Int).SetInt64(-1234567890), 6, "-1,234.567890", false},
		{"bigint zero", new(big.Int).SetInt64(0), 6, "0.000000", false},
		{"bigint large", func() *big.Int { bi, _ := new(big.Int).SetString("123456789012345678901234567890", 10); return bi }(), 18, "123,456,789,012.345678901234567890", false},

		// *big.Float inputs
		{"bigfloat positive", new(big.Float).SetFloat64(1234567890), 2, "12,345,678.90", false},
		{"bigfloat negative", new(big.Float).SetFloat64(-1234567890), 2, "-12,345,678.90", false},
		{"bigfloat zero", new(big.Float).SetFloat64(0), 2, "0.00", false},

		// Integer types - these are treated as raw values
		{"int positive", int(1234567), 3, "1,234.567", false},
		{"int negative", int(-1234567), 3, "-1,234.567", false},
		{"int8", int8(123), 2, "1.23", false},
		{"int16", int16(12345), 3, "12.345", false},
		{"int32", int32(1234567), 3, "1,234.567", false},
		{"int64", int64(1234567890), 6, "1,234.567890", false},

		// Unsigned integer types
		{"uint", uint(1234567), 3, "1,234.567", false},
		{"uint8", uint8(123), 2, "1.23", false},
		{"uint16", uint16(12345), 3, "12.345", false},
		{"uint32", uint32(1234567), 3, "1,234.567", false},
		{"uint64", uint64(1234567890), 6, "1,234.567890", false},

		// Float types - these will be converted to raw values first
		{"float32", float32(1234567), 3, "1,234.567", false},
		{"float64", float64(1234567890), 2, "12,345,678.90", false},
		{"float64 negative", float64(-1234567890), 2, "-12,345,678.90", false},

		// Edge cases
		{"zero decimals", "1234567", 0, "1,234,567", false},
		{"single digit", "5", 2, "0.05", false},
		{"two digits", "12", 2, "0.12", false},
		{"three digits", "123", 2, "1.23", false},
		{"four digits", "1234", 2, "12.34", false},
		{"very large number", "1234567890123456789", 0, "1,234,567,890,123,456,789", false},
		{"TRC20 max allowance", "115792089237316195423570985008687907853269984665640564039457584007913129639935", 18, "115,792,089,237,316,195,423,570,985,008,687,907,853,269,984,665,640,564,039,457.584007913129639935", false},

		// Error cases
		{"nil input", nil, 2, "", true},
		{"negative decimals", "123", -1, "", true},
		{"decimal string not allowed", "123.45", 2, "", true},
		{"float with decimal not allowed", 123.45, 2, "", true},
		{"nil bigint", (*big.Int)(nil), 2, "", true},
		{"nil bigfloat", (*big.Float)(nil), 2, "", true},
		{"bigfloat with decimal not allowed", new(big.Float).SetFloat64(123.45), 2, "", true},
		{"unsupported type", []int{1, 2, 3}, 2, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HumanReadableNumber(tt.number, tt.decimal)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHumanReadableTokenAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   any
		decimals int64
		expected string
		hasError bool
	}{
		{"USDT amount", "1000000", 6, "1.000000", false},
		{"ETH amount", "1000000000000000000", 18, "1.000000000000000000", false},
		{"TRX amount", "1000000", 6, "1.000000", false},
		{"large token amount", "1234567890123456789", 18, "1.234567890123456789", false},
		{"zero amount", "0", 6, "0.000000", false},
		{"negative amount", "-1000000", 6, "-1.000000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HumanReadableTokenAmount(tt.amount, tt.decimals)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHumanReadableBalance(t *testing.T) {
	tests := []struct {
		name     string
		balance  any
		decimals int64
		expected string
		hasError bool
	}{
		{"account balance", "1234567890", 6, "1,234.567890", false},
		{"small balance", "123456", 6, "0.123456", false},
		{"zero balance", "0", 6, "0.000000", false},
		{"negative balance", "-1234567890", 6, "-1,234.567890", false},
		{"large balance", "999999999999999999", 18, "0.999999999999999999", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HumanReadableBalance(tt.balance, tt.decimals)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestToBigFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
		hasError bool
	}{
		{"string number", "123.456", "", true}, // Now expects error
		{"string integer", "123", "123", false},
		{"string negative", "-123.456", "", true}, // Now expects error
		{"string zero", "0", "0", false},
		{"string scientific", "1.23e6", "", true}, // Now expects error
		{"empty string", "", "", true},
		{"invalid string", "abc", "", true},
		{"bigint", new(big.Int).SetInt64(123), "123", false},
		{"bigfloat", new(big.Float).SetFloat64(123.456), "", true}, // Now expects error
		{"int", int(123), "123", false},
		{"int64", int64(-123), "-123", false},
		{"uint64", uint64(123), "123", false},
		{"float64", float64(123.456), "", true}, // Now expects error
		{"nil", nil, "", true},
		{"unsupported type", []int{1, 2, 3}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toBigFloat(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.String())
			}
		})
	}
}

func TestAddCommasSeparators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single digit", "5", "5"},
		{"two digits", "12", "12"},
		{"three digits", "123", "123"},
		{"four digits", "1234", "1,234"},
		{"five digits", "12345", "12,345"},
		{"six digits", "123456", "123,456"},
		{"seven digits", "1234567", "1,234,567"},
		{"eight digits", "12345678", "12,345,678"},
		{"nine digits", "123456789", "123,456,789"},
		{"ten digits", "1234567890", "1,234,567,890"},
		{"very large", "1234567890123456789", "1,234,567,890,123,456,789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addCommasSeparators(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single char", "a", "a"},
		{"two chars", "ab", "ba"},
		{"word", "hello", "olleh"},
		{"numbers", "12345", "54321"},
		{"mixed", "a1b2c3", "3c2b1a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reverse(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkHumanReadableNumber(b *testing.B) {
	testCases := []struct {
		name    string
		number  any
		decimal int64
	}{
		{"string", "1234567890123456789", 18},
		{"bigint", func() *big.Int { bi, _ := new(big.Int).SetString("1234567890123456789", 10); return bi }(), 18},
		{"int64", int64(1234567890), 6},
		{"float64", float64(1234567.89), 2},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = HumanReadableNumber(tc.number, tc.decimal)
			}
		})
	}
}

func BenchmarkAddCommasSeparators(b *testing.B) {
	testString := "1234567890123456789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = addCommasSeparators(testString)
	}
}
