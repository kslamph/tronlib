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
//   - number: Input number (raw value, e.g., from smart contract)
//   - decimal: Number of decimal places the raw number should be divided by (e.g., 6 for USDT, 18 for ETH)
//
// Returns:
//   - string: Formatted number with comma separators (e.g., "1,234,567.890000")
//   - error: Error if conversion fails
//
// Examples:
//   HumanReadableNumber("1234567890", 6) -> "1,234.567890", nil (divides by 10^6)
//   HumanReadableNumber(-1234567890, 2) -> "-12,345,678.90", nil (divides by 10^2)
//   HumanReadableNumber("0", 0) -> "0", nil (no division)
func HumanReadableNumber(number any, decimal int64) (string, error) {
	if number == nil {
		return "", fmt.Errorf("number cannot be nil")
	}

	if decimal < 0 {
		return "", fmt.Errorf("decimal places cannot be negative: %d", decimal)
	}

	// Convert input to *big.Float for consistent handling
	bigFloat, err := toBigFloat(number)
	if err != nil {
		return "", fmt.Errorf("failed to convert number: %v", err)
	}

	// Handle zero case
	if bigFloat.Sign() == 0 {
		if decimal == 0 {
			return "0", nil
		}
		return "0." + strings.Repeat("0", int(decimal)), nil
	}

	// Check if number is negative
	isNegative := bigFloat.Sign() < 0
	if isNegative {
		bigFloat = new(big.Float).Abs(bigFloat)
	}

	// Apply decimal scaling (divide by 10^decimal)
	if decimal > 0 {
		divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(decimal), nil))
		bigFloat = new(big.Float).Quo(bigFloat, divisor)
	}

	// Format with specified decimal places
	formatted := bigFloat.Text('f', int(decimal))

	// Split into integer and decimal parts
	parts := strings.Split(formatted, ".")
	integerPart := parts[0]
	var decimalPart string
	if len(parts) > 1 {
		decimalPart = parts[1]
	}

	// Add comma separators to integer part
	integerPart = addCommasSeparators(integerPart)

	// Reconstruct the number
	result := integerPart
	if decimal > 0 {
		// Ensure decimal part has correct length
		if len(decimalPart) < int(decimal) {
			decimalPart = decimalPart + strings.Repeat("0", int(decimal)-len(decimalPart))
		}
		result += "." + decimalPart
	}

	// Add negative sign if needed
	if isNegative {
		result = "-" + result
	}

	return result, nil
}

// toBigFloat converts various numeric types to *big.Float
func toBigFloat(number any) (*big.Float, error) {
	switch v := number.(type) {
	case string:
		// Handle empty string
		if strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("empty string is not a valid number")
		}
		
		// Try to parse as big.Float
		bigFloat, _, err := big.ParseFloat(v, 10, 256, big.ToNearestEven)
		if err != nil {
			return nil, fmt.Errorf("invalid number string: %s", v)
		}
		return bigFloat, nil

	case *big.Int:
		if v == nil {
			return nil, fmt.Errorf("*big.Int cannot be nil")
		}
		return new(big.Float).SetInt(v), nil

	case *big.Float:
		if v == nil {
			return nil, fmt.Errorf("*big.Float cannot be nil")
		}
		return new(big.Float).Set(v), nil

	// Signed integers
	case int:
		return new(big.Float).SetInt64(int64(v)), nil
	case int8:
		return new(big.Float).SetInt64(int64(v)), nil
	case int16:
		return new(big.Float).SetInt64(int64(v)), nil
	case int32:
		return new(big.Float).SetInt64(int64(v)), nil
	case int64:
		return new(big.Float).SetInt64(v), nil

	// Unsigned integers
	case uint:
		return new(big.Float).SetUint64(uint64(v)), nil
	case uint8:
		return new(big.Float).SetUint64(uint64(v)), nil
	case uint16:
		return new(big.Float).SetUint64(uint64(v)), nil
	case uint32:
		return new(big.Float).SetUint64(uint64(v)), nil
	case uint64:
		return new(big.Float).SetUint64(v), nil

	// Floating point
	case float32:
		return new(big.Float).SetFloat64(float64(v)), nil
	case float64:
		return new(big.Float).SetFloat64(v), nil

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
//   - amount: Token amount (raw value from smart contract)
//   - decimals: Token decimals (e.g., 18 for most ERC20 tokens, 6 for USDT)
//
// Returns:
//   - string: Human-readable token amount
//   - error: Error if conversion fails
//
// Examples:
//   HumanReadableTokenAmount("1000000000000000000", 18) -> "1.000000000000000000", nil
//   HumanReadableTokenAmount("1000000", 6) -> "1.000000", nil
func HumanReadableTokenAmount(amount any, decimals int64) (string, error) {
	return HumanReadableNumber(amount, decimals)
}

// HumanReadableBalance is a convenience function for displaying balances
// It formats numbers with comma separators and a reasonable number of decimal places
//
// Parameters:
//   - balance: Balance amount
//   - decimals: Number of decimal places to show (default recommendation: 6)
//
// Returns:
//   - string: Human-readable balance
//   - error: Error if conversion fails
//
// Examples:
//   HumanReadableBalance("1234567890123456789", 6) -> "1,234,567,890.123457", nil
//   HumanReadableBalance(-1000000, 2) -> "-10.00", nil
func HumanReadableBalance(balance any, decimals int64) (string, error) {
	return HumanReadableNumber(balance, decimals)
}