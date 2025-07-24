// Package utils provides type conversion utilities for the TRON SDK
package utils

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

// TRX/SUN conversions

// TRXToSUN converts TRX amount to SUN (1 TRX = 1,000,000 SUN)
func TRXToSUN(trx float64) *big.Int {
	// Convert to SUN with precision
	sunFloat := trx * float64(types.SunPerTRX)
	sunBig := new(big.Float).SetFloat64(sunFloat)
	
	// Convert to integer
	sunInt, _ := sunBig.Int(nil)
	return sunInt
}

// SUNToTRX converts SUN amount to TRX as float64
func SUNToTRX(sun *big.Int) float64 {
	if sun == nil {
		return 0
	}
	
	sunFloat := new(big.Float).SetInt(sun)
	divisor := new(big.Float).SetInt64(types.SunPerTRX)
	result, _ := new(big.Float).Quo(sunFloat, divisor).Float64()
	
	return result
}

// SUNToTRXString converts SUN amount to TRX as formatted string
func SUNToTRXString(sun *big.Int, decimals int) string {
	if sun == nil {
		return "0"
	}
	
	return FormatBigInt(sun, 6) // TRX has 6 decimal places
}

// TRXStringToSUN converts TRX string to SUN big.Int
func TRXStringToSUN(trxStr string) (*big.Int, error) {
	trxFloat, err := strconv.ParseFloat(trxStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid TRX amount: %v", err)
	}
	
	return TRXToSUN(trxFloat), nil
}

// Address conversions

// AddressToBytes converts various address formats to bytes
func AddressToBytes(address string) ([]byte, error) {
	addr, err := ValidateAddress(address)
	if err != nil {
		return nil, err
	}
	
	return addr.Bytes(), nil
}

// BytesToAddress converts bytes to Address object
func BytesToAddress(data []byte) (*types.Address, error) {
	return types.NewAddressFromBytes(data)
}

// AddressToHex converts address to hex string
func AddressToHex(address string) (string, error) {
	addr, err := ValidateAddress(address)
	if err != nil {
		return "", err
	}
	
	return addr.Hex(), nil
}

// AddressToBase58 converts address to base58 string
func AddressToBase58(address string) (string, error) {
	addr, err := ValidateAddress(address)
	if err != nil {
		return "", err
	}
	
	return addr.Base58(), nil
}

// Time conversions

// UnixMillisToTime converts unix milliseconds to time.Time
func UnixMillisToTime(millis int64) time.Time {
	return time.Unix(millis/1000, (millis%1000)*1000000)
}

// TimeToUnixMillis converts time.Time to unix milliseconds
func TimeToUnixMillis(t time.Time) int64 {
	return t.UnixMilli()
}

// Now returns current time in unix milliseconds
func Now() int64 {
	return time.Now().UnixMilli()
}

// Resource type conversions

// ResourceTypeToCore converts ResourceType to core.ResourceCode
func ResourceTypeToCore(resourceType types.ResourceType) core.ResourceCode {
	switch resourceType {
	case types.ResourceBandwidth:
		return core.ResourceCode_BANDWIDTH
	case types.ResourceEnergy:
		return core.ResourceCode_ENERGY
	default:
		return core.ResourceCode_BANDWIDTH
	}
}

// CoreToResourceType converts core.ResourceCode to ResourceType
func CoreToResourceType(resourceCode core.ResourceCode) types.ResourceType {
	switch resourceCode {
	case core.ResourceCode_BANDWIDTH:
		return types.ResourceBandwidth
	case core.ResourceCode_ENERGY:
		return types.ResourceEnergy
	default:
		return types.ResourceBandwidth
	}
}

// Contract type conversions

// ContractTypeToString converts ContractType to string
func ContractTypeToString(contractType types.ContractType) string {
	return contractType.String()
}

// StringToContractType converts string to ContractType
func StringToContractType(str string) types.ContractType {
	switch str {
	case "TRC20":
		return types.ContractTypeTRC20
	case "TRC721":
		return types.ContractTypeTRC721
	case "TRC1155":
		return types.ContractTypeTRC1155
	case "CUSTOM":
		return types.ContractTypeCustom
	default:
		return types.ContractTypeUnknown
	}
}

// Number format conversions

// FormatSUN formats SUN amount with proper decimal places
func FormatSUN(amount *big.Int) string {
	if amount == nil {
		return "0"
	}
	
	// Format as TRX with 6 decimal places
	return FormatBigInt(amount, 6) + " TRX"
}

// FormatWei formats wei amount (for TRC20 tokens)
func FormatWei(amount *big.Int, decimals int, symbol string) string {
	if amount == nil {
		return "0"
	}
	
	formatted := FormatBigInt(amount, decimals)
	if symbol != "" {
		return formatted + " " + symbol
	}
	
	return formatted
}

// ParseAmount parses amount string with decimals
func ParseAmount(amountStr string, decimals int) (*big.Int, error) {
	if amountStr == "" {
		return big.NewInt(0), nil
	}
	
	// Parse as big float first
	amountFloat, ok := new(big.Float).SetString(amountStr)
	if !ok {
		return nil, fmt.Errorf("invalid amount format: %s", amountStr)
	}
	
	// Multiply by 10^decimals
	multiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	result := new(big.Float).Mul(amountFloat, multiplier)
	
	// Convert to integer
	amount, _ := result.Int(nil)
	return amount, nil
}

// Percentage calculations

// CalculatePercentage calculates percentage of an amount
func CalculatePercentage(amount *big.Int, percentage float64) *big.Int {
	if amount == nil || percentage <= 0 {
		return big.NewInt(0)
	}
	
	// Convert to float, calculate, and convert back
	amountFloat := new(big.Float).SetInt(amount)
	percentFloat := new(big.Float).SetFloat64(percentage / 100.0)
	result := new(big.Float).Mul(amountFloat, percentFloat)
	
	resultInt, _ := result.Int(nil)
	return resultInt
}

// CalculatePercentageOf calculates what percentage one amount is of another
func CalculatePercentageOf(part, total *big.Int) float64 {
	if total == nil || total.Sign() == 0 || part == nil {
		return 0
	}
	
	partFloat := new(big.Float).SetInt(part)
	totalFloat := new(big.Float).SetInt(total)
	
	// (part / total) * 100
	ratio := new(big.Float).Quo(partFloat, totalFloat)
	percentage := new(big.Float).Mul(ratio, big.NewFloat(100))
	
	result, _ := percentage.Float64()
	return result
}

// Unit conversions

// BytesToMB converts bytes to megabytes
func BytesToMB(bytes int64) float64 {
	return float64(bytes) / (1024 * 1024)
}

// MBToBytes converts megabytes to bytes
func MBToBytes(mb float64) int64 {
	return int64(mb * 1024 * 1024)
}

// SecondsToBlocks estimates number of blocks for given seconds
func SecondsToBlocks(seconds int64) int64 {
	return seconds * 1000 / types.BlockTimeMS
}

// BlocksToSeconds estimates seconds for given number of blocks
func BlocksToSeconds(blocks int64) int64 {
	return blocks * types.BlockTimeMS / 1000
}

// Energy calculations

// EstimateEnergyForBytes estimates energy needed for given data size
func EstimateEnergyForBytes(dataSize int) int64 {
	// Basic estimation: 1 energy per byte + base cost
	baseCost := int64(21000) // Base transaction cost
	dataCost := int64(dataSize) * types.EnergyPerByte
	
	return baseCost + dataCost
}

// EstimateBandwidthForBytes estimates bandwidth needed for given data size
func EstimateBandwidthForBytes(dataSize int) int64 {
	// Basic estimation: 1 bandwidth per byte + base cost
	baseCost := int64(200) // Base transaction cost
	dataCost := int64(dataSize) * types.BandwidthPerByte
	
	return baseCost + dataCost
}

// Array conversions

// StringSliceToBytes converts string slice to byte slices
func StringSliceToBytes(strings []string) [][]byte {
	result := make([][]byte, len(strings))
	for i, str := range strings {
		result[i] = []byte(str)
	}
	return result
}

// ByteSlicesToStrings converts byte slices to string slice
func ByteSlicesToStrings(bytes [][]byte) []string {
	result := make([]string, len(bytes))
	for i, b := range bytes {
		result[i] = string(b)
	}
	return result
}

// AddressSliceToBytes converts address strings to byte slices
func AddressSliceToBytes(addresses []string) ([][]byte, error) {
	result := make([][]byte, len(addresses))
	for i, addr := range addresses {
		bytes, err := AddressToBytes(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address at index %d: %v", i, err)
		}
		result[i] = bytes
	}
	return result, nil
}

// Safe math operations

// SafeAdd safely adds two big integers
func SafeAdd(a, b *big.Int) *big.Int {
	if a == nil {
		a = big.NewInt(0)
	}
	if b == nil {
		b = big.NewInt(0)
	}
	
	return new(big.Int).Add(a, b)
}

// SafeSub safely subtracts two big integers (returns 0 if result would be negative)
func SafeSub(a, b *big.Int) *big.Int {
	if a == nil {
		a = big.NewInt(0)
	}
	if b == nil {
		b = big.NewInt(0)
	}
	
	result := new(big.Int).Sub(a, b)
	if result.Sign() < 0 {
		return big.NewInt(0)
	}
	
	return result
}

// SafeMul safely multiplies two big integers
func SafeMul(a, b *big.Int) *big.Int {
	if a == nil {
		a = big.NewInt(0)
	}
	if b == nil {
		b = big.NewInt(0)
	}
	
	return new(big.Int).Mul(a, b)
}

// SafeDiv safely divides two big integers (returns 0 if divisor is 0)
func SafeDiv(a, b *big.Int) *big.Int {
	if a == nil {
		a = big.NewInt(0)
	}
	if b == nil || b.Sign() == 0 {
		return big.NewInt(0)
	}
	
	return new(big.Int).Div(a, b)
}