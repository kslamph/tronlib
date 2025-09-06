# 🏷️ Types Package Reference

The `types` package provides fundamental data types, constants, and utilities that form the foundation of TronLib. Understanding these types is essential for effective use of the library.

## 📚 Learning Path

This document is part of the TronLib learning path:
1. [Quick Start Guide](quickstart.md) - Basic usage
2. [Architecture Overview](architecture.md) - Understanding the design
3. **Types Package Reference** (this document) - Fundamental data types
4. [Other Package Documentation](../README.md#package-references) - Additional functionality
5. [API Reference](API_REFERENCE.md) - Complete function documentation

## 📋 Overview

The types package handles:
- **Address Management** - Multi-format TRON address handling
- **Transaction Wrappers** - Enhanced transaction structures
- **Constants** - Blockchain constants and conversion factors
- **Error Types** - Standardized error handling
- **Validation** - Input validation utilities

## 🏠 Address Type

The `Address` type is the cornerstone of TRON operations, supporting multiple address formats while maintaining type safety.

### Address Formats

TRON addresses can be represented in several formats:

| Format | Description | Example |
|--------|-------------|---------|
| **Base58** | Human-readable format | `TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH` |
| **TRON Hex** | 41-prefixed hex (21 bytes) | `41a614f803b6fd780986a42c78ec9c7f77e6ded13c` |
| **EVM Hex** | 0x-prefixed hex (20 bytes) | `0xa614f803b6fd780986a42c78ec9c7f77e6ded13c` |
| **Raw Bytes** | Binary representation | `[]byte{0x41, 0xa6, 0x14, ...}` |

### Creating Addresses

#### From Base58 String
```go
// Most common format
addr, err := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
if err != nil {
    return fmt.Errorf("invalid address: %w", err)
}

// With validation (panics on error - use only with trusted input)
addr := types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
```

#### From Hex String
```go
// TRON format (41-prefixed)
addr, err := types.NewAddressFromHex("41a614f803b6fd780986a42c78ec9c7f77e6ded13c")

// EVM format (0x-prefixed, 20 bytes)
addr, err := types.NewAddressFromHexEVM("0xa614f803b6fd780986a42c78ec9c7f77e6ded13c")
```

#### From Raw Bytes
```go
// TRON format (21 bytes, 0x41 prefix)
tronBytes := []byte{0x41, 0xa6, 0x14, /* ... */}
addr, err := types.NewAddressFromBytes(tronBytes)

// EVM format (20 bytes, no prefix)
evmBytes := []byte{0xa6, 0x14, /* ... */}
addr, err := types.NewAddressFromBytesEVM(evmBytes)
```

### Address Conversion

Once you have an `Address`, convert between formats easily:

```go
addr, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

// Get different representations
base58 := addr.String()                    // "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"
base58Alt := addr.Base58()                 // Same as String()
tronHex := addr.Hex()                      // "41a614f803b6fd780986a42c78ec9c7f77e6ded13c"
evmHex := addr.HexEVM()                    // "0xa614f803b6fd780986a42c78ec9c7f77e6ded13c"
tronBytes := addr.Bytes()                  // []byte{0x41, 0xa6, ...} (21 bytes)
evmBytes := addr.BytesEVM()                // []byte{0xa6, 0x14, ...} (20 bytes)
```

### Address Validation

The `Address` type provides built-in validation:

```go
// Valid TRON address
addr, err := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
if err != nil {
    // Handle invalid address
    fmt.Printf("Invalid address: %v", err)
}

// Check if address is zero address
if addr.IsZero() {
    fmt.Println("This is the zero address")
}

// Compare addresses
addr1, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
addr2, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

if addr1.Equal(addr2) {
    fmt.Println("Addresses are the same")
}
```

### Common Address Patterns

#### Safe Address Creation
```go
func createAddress(input string) (*types.Address, error) {
    addr, err := types.NewAddress(input)
    if err != nil {
        return nil, fmt.Errorf("failed to create address from %q: %w", input, err)
    }
    return addr, nil
}
```

#### Address from User Input
```go
func handleUserAddress(input string) error {
    // Trim whitespace and validate
    input = strings.TrimSpace(input)
    if input == "" {
        return errors.New("address cannot be empty")
    }

    addr, err := types.NewAddress(input)
    if err != nil {
        return fmt.Errorf("invalid TRON address format: %w", err)
    }

    // Use addr for operations...
    return nil
}
```

#### Working with Contract Addresses
```go
// Contract addresses are just regular addresses
contractAddr, err := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
if err != nil {
    return err
}

// You can use them with smart contract operations
contract, err := smartcontract.NewInstance(client, contractAddr, abi)
```

## 📊 Constants

The types package defines important blockchain constants:

### Currency Constants
```go
// SUN is the smallest unit of TRX
const SUN_PER_TRX = 1_000_000

// For exact integer-based conversions:
trxAmountTRX := int64(5)
sunAmount := trxAmountTRX * types.SUN_PER_TRX // 5,000,000 SUN

sunAmount := int64(10_000_000)
// For display purposes, use utils.HumanReadableBalance instead of direct division
```

**Note on Currency Conversion:**
TronLib intentionally avoids showing examples with `float64` arithmetic for currency conversions due to precision concerns. For financial applications, it's recommended to:

1. Use integer arithmetic for exact calculations (work in SUN)
2. Use `utils.HumanReadableBalance()` for displaying TRX amounts
3. Use `decimal.Decimal` for user input that requires fractional precision

### Address Constants
```go
// Size constants
const ADDRESS_SIZE = 21         // TRON address size in bytes
const ADDRESS_SIZE_EVM = 20     // EVM address size in bytes

// Validate address size
if len(addressBytes) != types.ADDRESS_SIZE {
    return errors.New("invalid address size")
}
```

### Network Constants
```go
// Network identifiers (used internally)
const MAINNET_NETWORK_ID = 1
const NILE_TESTNET_NETWORK_ID = 201910292
```

## ⚠️ Error Types

The types package defines standard error types used throughout TronLib:

### Address Errors
```go
var (
    ErrInvalidAddress     = errors.New("invalid address format")
    ErrInvalidAddressSize = errors.New("invalid address size")
    ErrInvalidChecksum    = errors.New("invalid address checksum")
)

// Usage example
addr, err := types.NewAddress("invalid-address")
if errors.Is(err, types.ErrInvalidAddress) {
    fmt.Println("The address format is invalid")
}
```

### Transaction Errors
```go
var (
    ErrInvalidTransaction = errors.New("invalid transaction")
    ErrInvalidSignature   = errors.New("invalid signature")
    ErrTransactionExpired = errors.New("transaction expired")
)
```

### Validation Errors
```go
var (
    ErrInvalidAmount = errors.New("invalid amount")
    ErrInvalidInput  = errors.New("invalid input")
)
```

## 🔧 Transaction Types

The types package includes enhanced transaction structures:

### Transaction Extension
```go
// TransactionExtension wraps core.Transaction with additional metadata
type TransactionExtension struct {
    Transaction *core.Transaction
    // Additional fields for tracking and processing
}
```

### Transaction Helpers
```go
// Check transaction status
func (tx *TransactionExtension) IsExpired() bool {
    // Implementation checks expiration time
}

// Get transaction size
func (tx *TransactionExtension) Size() int {
    // Returns serialized size
}
```

## 🛠️ Utility Functions

### Address Utilities

#### Batch Address Creation
```go
func CreateAddresses(inputs []string) ([]*types.Address, error) {
    addresses := make([]*types.Address, 0, len(inputs))
    
    for i, input := range inputs {
        addr, err := types.NewAddress(input)
        if err != nil {
            return nil, fmt.Errorf("invalid address at index %d (%q): %w", i, input, err)
        }
        addresses = append(addresses, addr)
    }
    
    return addresses, nil
}
```

#### Address Formatting
```go
func FormatAddress(addr *types.Address, format string) string {
    switch format {
    case "base58":
        return addr.String()
    case "hex":
        return addr.Hex()
    case "evm":
        return addr.HexEVM()
    default:
        return addr.String() // Default to base58
    }
}
```

### Amount Conversion Utilities

#### TRX/SUN Conversion

TronLib does not provide direct `TRXToSUN` and `SUNToTRX` conversion functions that use `float64` for TRX amounts. This is an intentional design decision to prevent precision issues that commonly occur with floating-point arithmetic in financial applications.

**Why Direct Conversion Functions Are Not Provided:**

1. **Floating-Point Precision Issues**: Using `float64` for financial calculations can lead to rounding errors and precision loss. For example, `0.1 + 0.2` in floating-point arithmetic does not equal exactly `0.3`.

2. **Financial Accuracy Requirements**: Blockchain applications require exact precision for monetary calculations. Even small rounding errors can lead to significant discrepancies in financial transactions.

3. **Best Practice Approach**: TronLib follows financial industry best practices by using integer-based arithmetic (SUN) for internal calculations and providing utility functions that handle conversions with proper precision.

**Recommended Approaches for TRX/SUN Conversion:**

1. **For Display Purposes**: Use `utils.HumanReadableBalance()` which provides properly formatted numbers with comma separators:
   ```go
   import "github.com/kslamph/tronlib/pkg/utils"
   
   balanceInSUN := int64(12500000)
   trxBalance, err := utils.HumanReadableBalance(balanceInSUN, 6) // "12.500000"
   if err != nil {
       // handle error
   }
   fmt.Printf("Balance: %s TRX\n", trxBalance)
   ```

2. **For TRC20 Token Operations**: Use the built-in decimal conversion in the TRC20 package:
   ```go
   import (
       "github.com/shopspring/decimal"
       "github.com/kslamph/tronlib/pkg/trc20"
   )
   
   // Convert human-readable amount to on-chain integer value
   humanAmount := decimal.NewFromFloat(12.5)
   weiAmount, err := trc20.ToWei(humanAmount, 6) // 6 decimals for USDT
   if err != nil {
       // handle error
   }
   ```

3. **For Manual Integer-Based Conversion**: When you need to convert between TRX and SUN, use integer arithmetic:
   ```go
   // TRX to SUN (for exact values)
   trxAmountSUN := trxAmountTRX * types.SUN_PER_TRX // where trxAmountTRX is an integer
   
   // SUN to TRX display (using utils package)
   trxAmountString, err := utils.HumanReadableBalance(sunAmount, 6)
   ```

By avoiding direct float64-based conversions, TronLib ensures that all financial calculations maintain the precision required for blockchain applications.

#### Safe Amount Handling
```go
func ValidateAmount(amount int64) error {
    if amount < 0 {
        return fmt.Errorf("amount cannot be negative: %d", amount)
    }
    if amount == 0 {
        return fmt.Errorf("amount cannot be zero")
    }
    return nil
}

func ClampAmount(amount, min, max int64) int64 {
    if amount < min {
        return min
    }
    if amount > max {
        return max
    }
    return amount
}
```

## 🎯 Best Practices

### 1. Always Validate Addresses
```go
// ✅ Good: Validate before use
func processTransfer(fromStr, toStr string, amount int64) error {
    from, err := types.NewAddress(fromStr)
    if err != nil {
        return fmt.Errorf("invalid from address: %w", err)
    }
    
    to, err := types.NewAddress(toStr)
    if err != nil {
        return fmt.Errorf("invalid to address: %w", err)
    }
    
    // Proceed with validated addresses
    return performTransfer(from, to, amount)
}

// ❌ Bad: Using strings directly
func badTransfer(from, to string, amount int64) {
    // Risk of invalid addresses causing runtime errors
}
```

### 2. Use Typed Functions
```go
// ✅ Good: Type-safe function signatures
func GetBalance(addr *types.Address) (int64, error) {
    // Implementation
}

// ❌ Bad: String parameters
func GetBalance(addr string) (int64, error) {
    // Caller might pass invalid address
}
```

### 3. Handle Conversion Errors
```go
// ✅ Good: Handle all conversion errors
func convertAddresses(inputs []string) ([]*types.Address, error) {
    var addresses []*types.Address
    
    for i, input := range inputs {
        addr, err := types.NewAddress(input)
        if err != nil {
            return nil, fmt.Errorf("address %d invalid: %w", i, err)
        }
        addresses = append(addresses, addr)
    }
    
    return addresses, nil
}
```

### 4. Use Constants for Clarity
```go
// ✅ Good: Use named constants
const (
    MinTransferAmount = 1                    // 1 SUN minimum
    MaxFeeLimit       = 1000 * types.SUN_PER_TRX // 1000 TRX
)

if amount < MinTransferAmount {
    return errors.New("amount too small")
}

// ❌ Bad: Magic numbers
if amount < 1 {
    return errors.New("amount too small")
}
```

## 🔍 Advanced Usage

### Custom Address Types
```go
// Create specialized address types for different purposes
type TokenAddress struct {
    *types.Address
    Symbol   string
    Decimals int
}

func NewTokenAddress(addr string, symbol string, decimals int) (*TokenAddress, error) {
    a, err := types.NewAddress(addr)
    if err != nil {
        return nil, err
    }
    
    return &TokenAddress{
        Address:  a,
        Symbol:   symbol,
        Decimals: decimals,
    }, nil
}
```

### Address Pools
```go
// Manage multiple addresses efficiently
type AddressPool struct {
    addresses []*types.Address
    current   int
}

func NewAddressPool(addrs []string) (*AddressPool, error) {
    pool := &AddressPool{}
    
    for _, addr := range addrs {
        a, err := types.NewAddress(addr)
        if err != nil {
            return nil, err
        }
        pool.addresses = append(pool.addresses, a)
    }
    
    return pool, nil
}

func (p *AddressPool) Next() *types.Address {
    addr := p.addresses[p.current]
    p.current = (p.current + 1) % len(p.addresses)
    return addr
}
```

## 🧪 Testing with Types

### Test Helpers
```go
// Helper functions for testing
func TestAddresses() []*types.Address {
    addrs := []string{
        "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
        "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x",
        "TAuB7aNiJ2Sj5r3xrqoRH8UhZVNYBUHxdf",
    }
    
    result := make([]*types.Address, len(addrs))
    for i, addr := range addrs {
        result[i] = types.MustNewAddressFromBase58(addr) // Safe for tests
    }
    
    return result
}

func RandomAddress() *types.Address {
    // Generate random address for testing
    bytes := make([]byte, 21)
    bytes[0] = 0x41 // TRON prefix
    rand.Read(bytes[1:])
    
    // This creates an address-like structure but without privatekey
    addr, _ := types.NewAddressFromBytes(bytes)
    return addr
}
```

### Assertion Helpers
```go
func AssertAddressEqual(t *testing.T, expected, actual *types.Address) {
    if !expected.Equal(actual) {
        t.Errorf("addresses not equal: expected %s, got %s", 
            expected.String(), actual.String())
    }
}

func AssertAddressFormat(t *testing.T, addr *types.Address, expectedBase58 string) {
    if addr.String() != expectedBase58 {
        t.Errorf("address format mismatch: expected %s, got %s", 
            expectedBase58, addr.String())
    }
}
```

The types package forms the foundation of TronLib, providing type safety and clarity for all blockchain operations. Master these types and patterns to write robust TRON applications! 🚀
