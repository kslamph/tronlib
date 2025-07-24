// Package utils provides validation utilities for the TRON SDK
package utils

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/tronlib/pkg/types"
)

// Address validation

// IsValidTronAddress validates a TRON address in any format
func IsValidTronAddress(address string) bool {
	if address == "" {
		return false
	}
	
	// Try different formats
	if IsValidBase58Address(address) {
		return true
	}
	
	if IsValidHexAddress(address) {
		return true
	}
	
	return false
}

// IsValidBase58Address validates a TRON base58 address
func IsValidBase58Address(address string) bool {
	if len(address) != types.AddressBase58Length {
		return false
	}
	
	// Try to parse as base58 address
	_, err := types.NewAddressFromBase58(address)
	return err == nil
}

// IsValidHexAddress validates a TRON hex address
func IsValidHexAddress(address string) bool {
	// Check format
	if strings.HasPrefix(address, "0x") {
		if len(address) != types.AddressHexLength {
			return false
		}
	} else {
		if len(address) != types.AddressHexLength-2 {
			return false
		}
	}
	
	// Try to parse as hex address
	_, err := types.NewAddressFromHex(address)
	return err == nil
}

// ValidateAddress validates an address and returns a standardized Address object
func ValidateAddress(address string) (*types.Address, error) {
	if address == "" {
		return nil, errors.New("address cannot be empty")
	}
	
	// Try base58 first
	if addr, err := types.NewAddressFromBase58(address); err == nil {
		return addr, nil
	}
	
	// Try hex
	if addr, err := types.NewAddressFromHex(address); err == nil {
		return addr, nil
	}
	
	return nil, fmt.Errorf("invalid address format: %s", address)
}

// Amount validation

// IsValidAmount validates that an amount is positive and within reasonable bounds
func IsValidAmount(amount *big.Int) bool {
	if amount == nil {
		return false
	}
	
	// Must be positive
	if amount.Sign() <= 0 {
		return false
	}
	
	// Check for reasonable upper bound (less than total TRX supply)
	maxSupply := new(big.Int).Mul(big.NewInt(100_000_000_000), big.NewInt(types.SunPerTRX)) // 100B TRX
	return amount.Cmp(maxSupply) <= 0
}

// ValidateAmount validates an amount and returns error if invalid
func ValidateAmount(amount *big.Int, minAmount *big.Int) error {
	if amount == nil {
		return errors.New("amount cannot be nil")
	}
	
	if amount.Sign() <= 0 {
		return errors.New("amount must be positive")
	}
	
	if minAmount != nil && amount.Cmp(minAmount) < 0 {
		return fmt.Errorf("amount %s is less than minimum %s", amount.String(), minAmount.String())
	}
	
	// Check for reasonable upper bound
	maxSupply := new(big.Int).Mul(big.NewInt(100_000_000_000), big.NewInt(types.SunPerTRX))
	if amount.Cmp(maxSupply) > 0 {
		return fmt.Errorf("amount %s exceeds maximum supply", amount.String())
	}
	
	return nil
}

// Message verification

// VerifyMessageV2 verifies a message signature using TIP-191 format (v2)
func VerifyMessageV2(message string, signature string, expectedAddress string) (bool, error) {
	// Parse the signature
	if !strings.HasPrefix(signature, "0x") {
		return false, errors.New("signature must start with 0x")
	}
	
	sigBytes := common.FromHex(signature)
	if len(sigBytes) != 65 {
		return false, errors.New("signature must be 65 bytes")
	}
	
	// Adjust recovery ID (v) back to go-ethereum format
	// Tron uses 27/28, go-ethereum uses 0/1
	if sigBytes[64] < 27 {
		return false, errors.New("invalid recovery ID")
	}
	sigBytes[64] -= 27
	
	// Prepare the message data
	var data []byte
	if strings.HasPrefix(message, "0x") {
		data = common.FromHex(message)
	} else {
		data = []byte(message)
	}
	
	// Prefix the message (same as signing)
	messageLen := len(data)
	prefixedMessage := []byte(fmt.Sprintf("\x19TRON Signed Message:\n%d%s", messageLen, string(data)))
	
	// Hash the prefixed message
	hash := crypto.Keccak256Hash(prefixedMessage)
	
	// Recover the public key
	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}
	
	// Convert public key to TRON address
	ethAddr := crypto.PubkeyToAddress(*pubKey)
	tronBytes := append([]byte{0x41}, ethAddr.Bytes()...)
	
	recoveredAddr, err := types.NewAddressFromBytes(tronBytes)
	if err != nil {
		return false, fmt.Errorf("failed to create recovered address: %w", err)
	}
	
	// Validate the expected address
	expectedAddr, err := ValidateAddress(expectedAddress)
	if err != nil {
		return false, fmt.Errorf("invalid expected address: %w", err)
	}
	
	// Compare addresses
	return recoveredAddr.String() == expectedAddr.String(), nil
}

// ValidateTRXAmount validates a TRX amount (in SUN)
func ValidateTRXAmount(amount *big.Int) error {
	minAmount := big.NewInt(1) // 1 SUN minimum
	return ValidateAmount(amount, minAmount)
}

// ValidateFreezeAmount validates an amount for freezing (minimum 1 TRX)
func ValidateFreezeAmount(amount *big.Int) error {
	minAmount := big.NewInt(types.SunPerTRX) // 1 TRX minimum
	return ValidateAmount(amount, minAmount)
}

// Contract validation

// IsValidContractAddress validates a smart contract address
func IsValidContractAddress(address string) bool {
	addr, err := ValidateAddress(address)
	if err != nil {
		return false
	}
	
	// Contract addresses start with 0x41 and have specific patterns
	bytes := addr.Bytes()
	return len(bytes) == 21 && bytes[0] == 0x41
}

// ValidateContractData validates smart contract call data
func ValidateContractData(data []byte) error {
	if len(data) == 0 {
		return errors.New("contract data cannot be empty")
	}
	
	// Check minimum length for method signature
	if len(data) < 4 {
		return errors.New("contract data must be at least 4 bytes (method signature)")
	}
	
	// Check maximum reasonable size
	if len(data) > types.MaxContractSize {
		return fmt.Errorf("contract data size %d exceeds maximum %d", len(data), types.MaxContractSize)
	}
	
	return nil
}

// ValidateABI validates a contract ABI JSON string
func ValidateABI(abiJSON string) error {
	if abiJSON == "" {
		return errors.New("ABI cannot be empty")
	}
	
	// Basic JSON validation
	_, err := JSONToMap(abiJSON)
	if err != nil {
		return fmt.Errorf("invalid ABI JSON: %v", err)
	}
	
	// TODO: Add more specific ABI validation
	return nil
}

// Transaction validation

// ValidateTransactionOptions validates transaction options
func ValidateTransactionOptions(opts *types.TransactionOptions) error {
	if opts == nil {
		return errors.New("transaction options cannot be nil")
	}
	
	// Validate fee limit
	if opts.FeeLimit < 0 {
		return errors.New("fee limit cannot be negative")
	}
	
	// Validate call value
	if opts.CallValue < 0 {
		return errors.New("call value cannot be negative")
	}
	
	// Validate token values
	if opts.TokenValue < 0 {
		return errors.New("token value cannot be negative")
	}
	
	if opts.TokenID < 0 {
		return errors.New("token ID cannot be negative")
	}
	
	// Validate permission ID
	if opts.PermissionID < 0 {
		return errors.New("permission ID cannot be negative")
	}
	
	return nil
}

// String validation

// IsValidMethodName validates a smart contract method name
func IsValidMethodName(method string) bool {
	if method == "" {
		return false
	}
	
	// Method names should be valid identifiers
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, method)
	return matched
}

// ValidateMethodName validates a method name and returns error if invalid
func ValidateMethodName(method string) error {
	if method == "" {
		return errors.New("method name cannot be empty")
	}
	
	if !IsValidMethodName(method) {
		return fmt.Errorf("invalid method name format: %s", method)
	}
	
	return nil
}

// IsValidTokenSymbol validates a token symbol
func IsValidTokenSymbol(symbol string) bool {
	if len(symbol) == 0 || len(symbol) > 10 {
		return false
	}
	
	// Token symbols should be alphanumeric
	matched, _ := regexp.MatchString(`^[A-Z0-9]+$`, strings.ToUpper(symbol))
	return matched
}

// ValidateTokenSymbol validates a token symbol and returns error if invalid
func ValidateTokenSymbol(symbol string) error {
	if symbol == "" {
		return errors.New("token symbol cannot be empty")
	}
	
	if len(symbol) > 10 {
		return errors.New("token symbol cannot be longer than 10 characters")
	}
	
	if !IsValidTokenSymbol(symbol) {
		return fmt.Errorf("invalid token symbol format: %s", symbol)
	}
	
	return nil
}

// Network validation

// IsValidNodeURL validates a TRON node URL
func IsValidNodeURL(url string) bool {
	if url == "" {
		return false
	}
	
	// Basic format validation
	// Should be in format: host:port or grpc://host:port
	patterns := []string{
		`^[a-zA-Z0-9.-]+:\d+$`,                    // host:port
		`^grpc://[a-zA-Z0-9.-]+:\d+$`,            // grpc://host:port
		`^grpcs://[a-zA-Z0-9.-]+:\d+$`,           // grpcs://host:port
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, url); matched {
			return true
		}
	}
	
	return false
}

// ValidateNodeURL validates a node URL and returns error if invalid
func ValidateNodeURL(url string) error {
	if url == "" {
		return errors.New("node URL cannot be empty")
	}
	
	if !IsValidNodeURL(url) {
		return fmt.Errorf("invalid node URL format: %s", url)
	}
	
	return nil
}

// Numeric validation

// IsValidDecimals validates token decimals
func IsValidDecimals(decimals int) bool {
	return decimals >= 0 && decimals <= 18
}

// ValidateDecimals validates token decimals and returns error if invalid
func ValidateDecimals(decimals int) error {
	if decimals < 0 {
		return errors.New("decimals cannot be negative")
	}
	
	if decimals > 18 {
		return errors.New("decimals cannot be greater than 18")
	}
	
	return nil
}

// IsValidPermissionID validates a permission ID
func IsValidPermissionID(permissionID int32) bool {
	return permissionID >= 0 && permissionID <= 255
}

// ValidatePermissionID validates a permission ID and returns error if invalid
func ValidatePermissionID(permissionID int32) error {
	if !IsValidPermissionID(permissionID) {
		return fmt.Errorf("invalid permission ID: %d (must be 0-255)", permissionID)
	}
	
	return nil
}

// Batch validation

// ValidateAddresses validates multiple addresses
func ValidateAddresses(addresses []string) error {
	if len(addresses) == 0 {
		return errors.New("addresses list cannot be empty")
	}
	
	for i, addr := range addresses {
		if _, err := ValidateAddress(addr); err != nil {
			return fmt.Errorf("invalid address at index %d: %v", i, err)
		}
	}
	
	return nil
}

// ValidateAmounts validates multiple amounts
func ValidateAmounts(amounts []*big.Int, minAmount *big.Int) error {
	if len(amounts) == 0 {
		return errors.New("amounts list cannot be empty")
	}
	
	for i, amount := range amounts {
		if err := ValidateAmount(amount, minAmount); err != nil {
			return fmt.Errorf("invalid amount at index %d: %v", i, err)
		}
	}
	
	return nil
}