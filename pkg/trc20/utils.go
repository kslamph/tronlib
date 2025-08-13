package trc20

import (
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

// ToWei converts a user-facing decimal amount to on-chain integer units using
// the provided decimals. Decimals must be ≤ 18. If the amount has more
// fractional places than allowed, it returns an error (no truncation).
func ToWei(amount decimal.Decimal, decimals uint8) (*big.Int, error) {
	if decimals > 18 {
		return nil, fmt.Errorf("unsupported decimals value: %d", decimals)
	}
	// Validate scale: number of fractional digits must be ≤ decimals
	if amount.Exponent() < 0 {
		scale := -amount.Exponent()
		if int32(scale) > int32(decimals) {
			return nil, fmt.Errorf("amount has too many decimal places for token with %d decimals", decimals)
		}
	}
	// Exact scaling using Shift, safe because we validated scale above.
	return amount.Shift(int32(decimals)).BigInt(), nil
}

// FromWei converts a raw on-chain integer value to a user-facing decimal using
// the provided decimals. Decimals must be ≤ 18.
func FromWei(value *big.Int, decimals uint8) (decimal.Decimal, error) {
	if value == nil {
		return decimal.Zero, nil
	}
	if decimals > 18 {
		return decimal.Zero, fmt.Errorf("unsupported decimals value: %d", decimals)
	}
	return decimal.NewFromBigInt(value, -int32(decimals)), nil
}

// toWei is the internal helper kept for backward-compatibility.
// It now defers to ToWei for validation and conversion.
func toWei(amount decimal.Decimal, decimals uint8) (*big.Int, error) {
	return ToWei(amount, decimals)
}

// fromWei is the internal helper kept for backward-compatibility.
// It now defers to FromWei for validation and conversion.
func fromWei(rawAmount *big.Int, decimals uint8) (decimal.Decimal, error) {
	return FromWei(rawAmount, decimals)
}
