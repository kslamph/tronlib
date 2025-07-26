package trc20

import (
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

// toWei converts a decimal.Decimal amount to a raw *big.Int amount based on the token's decimals.
func toWei(amount decimal.Decimal, decimals uint8) (*big.Int, error) {
	if decimals > 18 { // Standard max for Ethereum/TRON tokens is 18
		return nil, fmt.Errorf("unsupported decimals value: %d", decimals)
	}
	// Shift the decimal point back to the right

	// // Shift the decimal point by 'decimals' places
	// shiftedAmount := amount.Mul(decimal.NewFromInt(1).Shift(int32(decimals)))

	// // Check if the shifted amount has any fractional part left
	// if shiftedAmount.Mod(decimal.NewFromInt(1)).Cmp(decimal.NewFromInt(0)) != 0 {
	// 	return nil, fmt.Errorf("amount has too many decimal places for token with %d decimals", decimals)
	// }

	// // Convert to *big.Int
	// result := new(big.Int)
	// _, ok := result.SetString(shiftedAmount.String(), 10) // Convert decimal string to big.Int
	// if !ok {
	// 	return nil, fmt.Errorf("failed to convert decimal to big.Int")
	// }
	return amount.Shift(int32(decimals)).BigInt(), nil
}

// fromWei converts a raw *big.Int amount to a decimal.Decimal amount based on the token's decimals.
func fromWei(rawAmount *big.Int, decimals uint8) (decimal.Decimal, error) {
	if rawAmount == nil {
		return decimal.Zero, nil
	}
	if decimals > 18 {
		return decimal.Zero, fmt.Errorf("unsupported decimals value: %d", decimals)
	}

	// Convert rawAmount to decimal.Decimal

	// Shift the decimal point back by 'decimals' places
	// result := amount.Div(decimal.NewFromInt(1).Shift(int32(decimals)))

	return decimal.NewFromBigInt(rawAmount, -int32(decimals)), nil
}
