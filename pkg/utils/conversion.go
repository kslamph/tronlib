package utils

import (
	"fmt"
	"math/big"
	"strings"
)

// HumanReadableNumber converts a number to human-readable format with comma separators
// and decimal places. Handles negative numbers and various input types.
//
// Parameters:
//   - number: Input number (raw value, e.g., from smart contract) - must be a whole number
//   - decimal: Number of decimal places the raw number should be divided by (e.g., 6 for USDT, 18 for ETH)
//
// Returns:
//   - string: Formatted number with comma separators (e.g., "1,234,567.890000")
//   - error: Error if conversion fails
//
// Examples:
//
//	HumanReadableNumber("1234567890", 6) -> "1,234.567890", nil (divides by 10^6)
//	HumanReadableNumber(-1234567890, 2) -> "-12,345,678.90", nil (divides by 10^2)
//	HumanReadableNumber("0", 0) -> "0", nil (no division)
func HumanReadableNumber(number any, decimal int64) (string, error) {
	if number == nil {
		return "", fmt.Errorf("number cannot be nil")
	}

	if decimal < 0 {
		return "", fmt.Errorf("decimal places cannot be negative: %d", decimal)
	}

	// Convert input to *big.Int for precise arithmetic
	bigInt, err := toBigInt(number)
	if err != nil {
		return "", fmt.Errorf("failed to convert number: %v", err)
	}

	// Handle zero case
	if bigInt.Sign() == 0 {
		if decimal == 0 {
			return "0", nil
		}
		return "0." + strings.Repeat("0", int(decimal)), nil
	}

	// Check if number is negative
	isNegative := bigInt.Sign() < 0
	if isNegative {
		bigInt = new(big.Int).Abs(bigInt)
	}

	var integerPart, decimalPart string

	// Apply decimal scaling (divide by 10^decimal)
	if decimal > 0 {
		// Create divisor: 10^decimal
		base := big.NewInt(10)
		exponent := big.NewInt(decimal)
		divisor := new(big.Int).Exp(base, exponent, nil)

		// Perform division to get quotient and remainder
		quotient := new(big.Int)
		remainder := new(big.Int)
		quotient.DivMod(bigInt, divisor, remainder)

		// Integer part
		integerPart = quotient.String()

		// Decimal part - pad with leading zeros if necessary
		decimalPart = remainder.String()
		// Pad with leading zeros to match decimal places
		if len(decimalPart) < int(decimal) {
			decimalPart = strings.Repeat("0", int(decimal)-len(decimalPart)) + decimalPart
		}
	} else {
		// No decimal places
		integerPart = bigInt.String()
	}

	// Add comma separators to integer part
	integerPart = addCommasSeparators(integerPart)

	// Reconstruct the number
	result := integerPart
	if decimal > 0 {
		result += "." + decimalPart
	}

	// Add negative sign if needed
	if isNegative {
		result = "-" + result
	}

	return result, nil
}

// toBigInt converts various numeric types to *big.Int
// Only accepts whole numbers (integers) to maintain precision
func toBigInt(number any) (*big.Int, error) {
	switch v := number.(type) {
	case string:
		// Handle empty string
		if strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("empty string is not a valid number")
		}

		// Check if string contains decimal point (not allowed for whole numbers)
		if strings.Contains(v, ".") {
			return nil, fmt.Errorf("decimal numbers not allowed, input must be a whole number: %s", v)
		}

		// Try to parse as big.Int
		bigInt, ok := new(big.Int).SetString(v, 10)
		if !ok {
			return nil, fmt.Errorf("invalid number string: %s", v)
		}
		return bigInt, nil

	case *big.Int:
		if v == nil {
			return nil, fmt.Errorf("*big.Int cannot be nil")
		}
		return new(big.Int).Set(v), nil

	case *big.Float:
		if v == nil {
			return nil, fmt.Errorf("*big.Float cannot be nil")
		}
		// Check if it's a whole number
		if !v.IsInt() {
			return nil, fmt.Errorf("big.Float must represent a whole number, got: %s", v.String())
		}
		bigInt, _ := v.Int(nil)
		return bigInt, nil

	// Signed integers
	case int:
		return big.NewInt(int64(v)), nil
	case int8:
		return big.NewInt(int64(v)), nil
	case int16:
		return big.NewInt(int64(v)), nil
	case int32:
		return big.NewInt(int64(v)), nil
	case int64:
		return big.NewInt(v), nil

	// Unsigned integers
	case uint:
		return new(big.Int).SetUint64(uint64(v)), nil
	case uint8:
		return new(big.Int).SetUint64(uint64(v)), nil
	case uint16:
		return new(big.Int).SetUint64(uint64(v)), nil
	case uint32:
		return new(big.Int).SetUint64(uint64(v)), nil
	case uint64:
		return new(big.Int).SetUint64(v), nil

	// Floating point - only accept if they represent whole numbers
	case float32:
		if v != float32(int64(v)) {
			return nil, fmt.Errorf("float32 must represent a whole number, got: %f", v)
		}
		return big.NewInt(int64(v)), nil
	case float64:
		if v != float64(int64(v)) {
			return nil, fmt.Errorf("float64 must represent a whole number, got: %f", v)
		}
		return big.NewInt(int64(v)), nil

	default:
		return nil, fmt.Errorf("unsupported number type: %T", number)
	}
}

// addCommasSeparators adds comma separators to an integer string
func addCommasSeparators(s string) string {
	if len(s) <= 3 {
		return s
	}

	// Handle the case where we need to add commas
	var result strings.Builder

	// Process from right to left
	for i, digit := range reverse(s) {
		if i > 0 && i%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}

	return reverse(result.String())
}

// reverse reverses a string
func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// HumanReadableTokenAmount is a convenience function for token amounts
// It handles the common case of converting token amounts with known decimals
//
// Parameters:
//   - amount: Token amount (raw value from smart contract) - must be a whole number
//   - decimals: Token decimals (e.g., 18 for most ERC20 tokens, 6 for USDT)
//
// Returns:
//   - string: Human-readable token amount
//   - error: Error if conversion fails
//
// Examples:
//
//	HumanReadableTokenAmount("1000000000000000000", 18) -> "1.000000000000000000", nil
//	HumanReadableTokenAmount("1000000", 6) -> "1.000000", nil
func HumanReadableTokenAmount(amount any, decimals int64) (string, error) {
	return HumanReadableNumber(amount, decimals)
}

// HumanReadableBalance is a convenience function for displaying balances
// It formats numbers with comma separators and a reasonable number of decimal places
//
// Parameters:
//   - balance: Balance amount - must be a whole number
//   - decimals: Number of decimal places to show (default recommendation: 6)
//
// Returns:
//   - string: Human-readable balance
//   - error: Error if conversion fails
//
// Examples:
//
//	HumanReadableBalance("1234567890123456789", 6) -> "1,234,567,890.123457", nil
//	HumanReadableBalance(-1000000, 2) -> "-10.00", nil
func HumanReadableBalance(balance any, decimals int64) (string, error) {
	return HumanReadableNumber(balance, decimals)
}

// toBigFloat converts various numeric types to *big.Float (kept for backward compatibility)
// Deprecated: Use toBigInt for precise arithmetic with whole numbers
func toBigFloat(number any) (*big.Float, error) {
	bigInt, err := toBigInt(number)
	if err != nil {
		return nil, err
	}
	return new(big.Float).SetInt(bigInt), nil
}