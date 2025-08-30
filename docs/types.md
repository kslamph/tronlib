# 🏷️ Types Package Reference

The `types` package provides fundamental data types, constants, and utilities that form the foundation of TronLib. Understanding these types is essential for effective use of the library.

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

// Convert between TRX and SUN
trxAmount := 5.5
sunAmount := int64(trxAmount * types.SUN_PER_TRX) // 5,500,000 SUN

sunAmount := int64(10_000_000)
trxAmount := float64(sunAmount) / types.SUN_PER_TRX // 10.0 TRX
```

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
```go
// Helper functions for amount conversion
func TRXToSUN(trx float64) int64 {
    return int64(trx * types.SUN_PER_TRX)
}

func SUNToTRX(sun int64) float64 {
    return float64(sun) / types.SUN_PER_TRX
}

// Example usage
userInput := 12.5 // TRX
sunAmount := TRXToSUN(userInput) // 12,500,000 SUN

receiptAmount := int64(5_000_000) // SUN from blockchain
trxAmount := SUNToTRX(receiptAmount) // 5.0 TRX for display
```

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
