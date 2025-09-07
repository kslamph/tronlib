// Package utils provides validation utilities for the TRON SDK
package utils

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/tronlib/pkg/types"
)

// IsValidAmount validates that an amount is positive and within reasonable bounds
func IsValidAmount(amount *big.Int) bool {
	if amount == nil {
		return false
	}

	// Must be positive
	if amount.Sign() <= 0 {
		return false
	}
	return true
}

// ValidateAmount validates an amount and returns error if invalid
func ValidateAmount(amount *big.Int) error {
	if amount == nil {
		return fmt.Errorf("%w: amount cannot be nil", types.ErrInvalidAmount)
	}
	if amount.Sign() <= 0 {
		return fmt.Errorf("%w: amount must be positive, got %s", types.ErrInvalidAmount, amount.String())
	}
	return nil
}

// VerifyMessageV2 verifies a message signature using TIP-191 format (v2)
func VerifyMessageV2(message string, signature string, expectedAddress string) (bool, error) {
	// Validate the expected address
	expectedAddr, err := types.NewAddress(expectedAddress)
	if err != nil {
		return false, fmt.Errorf("invalid expected address: %w", err)
	}

	// Parse the signature
	if !strings.HasPrefix(signature, "0x") {
		return false, fmt.Errorf("%w: signature must start with 0x, got %s", types.ErrInvalidParameter, signature)
	}

	sigBytes := common.FromHex(signature)
	if len(sigBytes) != 65 {
		return false, fmt.Errorf("%w: signature must be 65 bytes, got %d bytes", types.ErrInvalidParameter, len(sigBytes))
	}

	// Adjust recovery ID (v) back to go-ethereum format
	// Tron uses 27/28, go-ethereum uses 0/1
	if sigBytes[64] < 27 {
		return false, fmt.Errorf("%w: invalid recovery ID, must be 27 or 28, got %d", types.ErrInvalidParameter, sigBytes[64])
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

	// Compare addresses
	return recoveredAddr.String() == expectedAddr.String(), nil
}

// Contract validation

// ValidateContractData validates smart contract call data
func ValidateContractData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: contract data cannot be empty", types.ErrInvalidContract)
	}

	// Check minimum length for method signature
	if len(data) < 4 {
		return fmt.Errorf("%w: contract data must be at least 4 bytes (method signature), got %d bytes", types.ErrInvalidContract, len(data))
	}

	// Check maximum reasonable size
	if len(data) > types.MaxContractSize {
		return fmt.Errorf("%w: contract data size %d exceeds maximum %d bytes", types.ErrInvalidContract, len(data), types.MaxContractSize)
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
		return fmt.Errorf("%w: method name cannot be empty", types.ErrInvalidParameter)
	}

	if !IsValidMethodName(method) {
		return fmt.Errorf("%w: invalid method name format: %s (must be valid identifier)", types.ErrInvalidParameter, method)
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
		return fmt.Errorf("%w: token symbol cannot be empty", types.ErrInvalidParameter)
	}

	if len(symbol) > 10 {
		return fmt.Errorf("%w: token symbol cannot be longer than 10 characters, got %d characters", types.ErrInvalidParameter, len(symbol))
	}

	if !IsValidTokenSymbol(symbol) {
		return fmt.Errorf("%w: invalid token symbol format: %s (must be alphanumeric and uppercase)", types.ErrInvalidParameter, symbol)
	}

	return nil
}

// Network validation

// IsValidNodeURL validates a TRON node URL.
// Requires explicit scheme and host:port, e.g. grpc://host:port or grpcs://host:port
func IsValidNodeURL(url string) bool {
	if url == "" {
		return false
	}

	// Basic format validation
	// Should be in format: grpc://host:port or grpcs://host:port
	patterns := []string{
		`^grpc://[a-zA-Z0-9.-]+:\d+$`,  // grpc://host:port
		`^grpcs://[a-zA-Z0-9.-]+:\d+$`, // grpcs://host:port
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
		return fmt.Errorf("%w: node URL cannot be empty", types.ErrInvalidParameter)
	}

	if !IsValidNodeURL(url) {
		return fmt.Errorf("%w: invalid node URL format: %s (expected grpc://host:port or grpcs://host:port)", types.ErrInvalidParameter, url)
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
		return fmt.Errorf("%w: decimals cannot be negative, got %d", types.ErrInvalidParameter, decimals)
	}

	if decimals > 18 {
		return fmt.Errorf("%w: decimals cannot be greater than 18, got %d", types.ErrInvalidParameter, decimals)
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
		return fmt.Errorf("%w: invalid permission ID: %d (must be 0-255)", types.ErrInvalidParameter, permissionID)
	}

	return nil
}

// Contract name validation

// IsValidContractName validates a smart contract name
func IsValidContractName(name string) bool {
	if name == "" {
		return true // Empty names are allowed
	}

	// Contract names should only contain visible characters and spaces
	// Visible characters are printable characters excluding control characters
	for _, r := range name {
		if r < 32 || r == 127 { // Control characters
			return false
		}
	}

	return true
}

// ValidateContractName validates a contract name and returns error if invalid
func ValidateContractName(name string) error {
	if !IsValidContractName(name) {
		return fmt.Errorf("%w: invalid contract name: contains non-visible characters", types.ErrInvalidParameter)
	}

	return nil
}

// ValidateConsumeUserResourcePercent validates the consume user resource percentage
func ValidateConsumeUserResourcePercent(percent int64) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("%w: consume user resource percent must be between 0 and 100, got %d", types.ErrInvalidParameter, percent)
	}

	return nil
}
