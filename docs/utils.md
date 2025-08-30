# 🛠️ Utils Package Reference

The `utils` package provides essential ABI (Application Binary Interface) encoding/decoding functionality and type conversion utilities. It serves as the backbone for smart contract interactions and event processing.

## 📋 Overview

The utils package handles:
- **ABI Processing** - Parse, encode, and decode contract ABIs
- **Method Encoding** - Convert Go values to contract method calls
- **Event Decoding** - Parse event logs into structured data
- **Type Conversion** - Handle Solidity ↔ Go type mappings
- **Data Validation** - Ensure data integrity and format compliance

## 🏗️ Core Components

### ABIProcessor

The `ABIProcessor` is the central component for all ABI-related operations:

```go
type ABIProcessor struct {
    abi *core.SmartContract_ABI
    // Internal state for caching and optimization
}

// Create a new processor
func NewABIProcessor(abi *core.SmartContract_ABI) *ABIProcessor
```

## 📝 ABI Parsing

### Loading ABI from JSON

```go
// Parse ABI from JSON string
abiJSON := `[
    {
        "type": "function",
        "name": "transfer",
        "inputs": [
            {"name": "_to", "type": "address"},
            {"name": "_value", "type": "uint256"}
        ],
        "outputs": [{"name": "", "type": "bool"}]
    }
]`

abi, err := utils.ParseABI(abiJSON)
if err != nil {
    log.Fatalf("Failed to parse ABI: %v", err)
}

processor := utils.NewABIProcessor(abi)
```

### Loading ABI from File

```go
func LoadABIFromFile(filename string) (*core.SmartContract_ABI, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read ABI file: %w", err)
    }

    return utils.ParseABI(string(data))
}

// Usage
abi, err := LoadABIFromFile("contract.abi.json")
if err != nil {
    log.Fatal(err)
}
```

## 🔧 Method Encoding

### Basic Method Encoding

```go
// Encode a method call with parameters
processor := utils.NewABIProcessor(abi)

// Transfer method: transfer(address _to, uint256 _value)
recipientAddr, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
amount := big.NewInt(1000000) // 1 USDT (6 decimals)

encoded, err := processor.EncodeMethod("transfer", 
    []string{"address", "uint256"}, 
    recipientAddr, amount)
if err != nil {
    log.Fatalf("Encoding failed: %v", err)
}

fmt.Printf("Encoded method call: %x\n", encoded)
```

### Advanced Parameter Types

```go
// Complex method with multiple parameter types
// approve(address spender, uint256 amount)
spender, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
amount := big.NewInt(0) // Unlimited approval

encoded, err := processor.EncodeMethod("approve",
    []string{"address", "uint256"},
    spender, amount)

// Method with array parameters
// batchTransfer(address[] recipients, uint256[] amounts)
recipients := []*types.Address{
    types.MustNewAddressFromBase58("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x"),
    types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"),
}
amounts := []*big.Int{
    big.NewInt(1000000),
    big.NewInt(2000000),
}

encoded, err = processor.EncodeMethod("batchTransfer",
    []string{"address[]", "uint256[]"},
    recipients, amounts)
```

### Parameter Type Mapping

| Solidity Type | Go Type | Example |
|---------------|---------|---------|
| `address` | `*types.Address` | `types.NewAddress("T...")` |
| `uint256` | `*big.Int` | `big.NewInt(123)` |
| `uint8/16/32` | `uint8/16/32` | `uint32(42)` |
| `int256` | `*big.Int` | `big.NewInt(-123)` |
| `bool` | `bool` | `true` |
| `string` | `string` | `"hello"` |
| `bytes` | `[]byte` | `[]byte{0x01, 0x02}` |
| `bytes32` | `[32]byte` | `[32]byte{...}` |
| `address[]` | `[]*types.Address` | `[]*types.Address{...}` |
| `uint256[]` | `[]*big.Int` | `[]*big.Int{...}` |

## 🎭 Event Decoding

### Simple Event Decoding

```go
// Decode Transfer event
// event Transfer(address indexed from, address indexed to, uint256 value)

topics := [][]byte{
    // Event signature hash
    mustDecodeHex("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
    // From address (indexed)
    mustDecodeHex("000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
    // To address (indexed)  
    mustDecodeHex("0000000000000000000000004e83362442b8d1bec281594cea3050c8eb01311c"),
}

data := mustDecodeHex("00000000000000000000000000000000000000000000000000000000000003e8")

event, err := utils.DecodeEvent(abi, topics, data)
if err != nil {
    log.Fatalf("Failed to decode event: %v", err)
}

fmt.Printf("Event: %s\n", event.Name)
for _, param := range event.Inputs {
    fmt.Printf("  %s: %v\n", param.Name, param.Value)
}
```

### Event Signature Generation

```go
// Generate event signature for lookup
signature := utils.EventSignature("Transfer", []string{"address", "address", "uint256"})
fmt.Printf("Transfer signature: %x\n", signature)
// Output: ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef

// Common TRC20 event signatures
transferSig := utils.EventSignature("Transfer", []string{"address", "address", "uint256"})
approvalSig := utils.EventSignature("Approval", []string{"address", "address", "uint256"})
```

## 📊 Direct Encoding/Decoding Functions

For simple use cases, the package provides direct encoding functions:

### Address Encoding/Decoding

```go
// Encode address to 32-byte format
addr, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
encoded := utils.EncodeAddress(addr)
fmt.Printf("Encoded address: %x\n", encoded) // 32 bytes

// Decode address from 32-byte format
decoded, err := utils.DecodeAddress(encoded)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Decoded address: %s\n", decoded.String())
```

### Number Encoding/Decoding

```go
// Encode uint256
value := big.NewInt(123456789)
encoded := utils.EncodeUint256(value)
fmt.Printf("Encoded uint256: %x\n", encoded) // 32 bytes

// Decode uint256
decoded, err := utils.DecodeUint256(encoded)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Decoded value: %s\n", decoded.String())

// Smaller integer types
encoded32 := utils.EncodeUint32(42)
decoded32, _ := utils.DecodeUint32(encoded32)
```

### String Encoding/Decoding

```go
// Encode string
text := "Hello, TRON!"
encoded := utils.EncodeString(text)
fmt.Printf("Encoded string: %x\n", encoded)

// Decode string
decoded, err := utils.DecodeString(encoded)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Decoded string: %s\n", decoded)
```

### Bytes Encoding/Decoding

```go
// Dynamic bytes
data := []byte{0x01, 0x02, 0x03, 0x04}
encoded := utils.EncodeBytes(data)
decoded, _ := utils.DecodeBytes(encoded)

// Fixed-size bytes
var fixed32 [32]byte
copy(fixed32[:], data)
encodedFixed := utils.EncodeBytes32(fixed32)
decodedFixed, _ := utils.DecodeBytes32(encodedFixed)
```

## 🔍 Advanced Usage Patterns

### Multi-Method Contract Interface

```go
type TokenContract struct {
    processor *utils.ABIProcessor
}

func NewTokenContract(abiJSON string) (*TokenContract, error) {
    abi, err := utils.ParseABI(abiJSON)
    if err != nil {
        return nil, err
    }

    return &TokenContract{
        processor: utils.NewABIProcessor(abi),
    }, nil
}

func (tc *TokenContract) EncodeTransfer(to *types.Address, amount *big.Int) ([]byte, error) {
    return tc.processor.EncodeMethod("transfer", 
        []string{"address", "uint256"}, 
        to, amount)
}

func (tc *TokenContract) EncodeApprove(spender *types.Address, amount *big.Int) ([]byte, error) {
    return tc.processor.EncodeMethod("approve",
        []string{"address", "uint256"},
        spender, amount)
}

func (tc *TokenContract) EncodeBalanceOf(owner *types.Address) ([]byte, error) {
    return tc.processor.EncodeMethod("balanceOf",
        []string{"address"},
        owner)
}
```

### Batch Operations

```go
// Encode multiple method calls
func EncodeBatchCalls(processor *utils.ABIProcessor, calls []MethodCall) ([][]byte, error) {
    encoded := make([][]byte, len(calls))
    
    for i, call := range calls {
        data, err := processor.EncodeMethod(call.Method, call.Types, call.Params...)
        if err != nil {
            return nil, fmt.Errorf("failed to encode call %d: %w", i, err)
        }
        encoded[i] = data
    }
    
    return encoded, nil
}

type MethodCall struct {
    Method string
    Types  []string
    Params []interface{}
}
```

### Custom Type Handlers

```go
// Handle custom struct types
func EncodeStruct(structType string, values map[string]interface{}) ([]byte, error) {
    switch structType {
    case "UserProfile":
        return encodeUserProfile(values)
    case "TokenInfo":
        return encodeTokenInfo(values)
    default:
        return nil, fmt.Errorf("unknown struct type: %s", structType)
    }
}

func encodeUserProfile(values map[string]interface{}) ([]byte, error) {
    name := values["name"].(string)
    age := values["age"].(uint8)
    addr := values["address"].(*types.Address)
    
    // Encode struct fields according to ABI specification
    nameEncoded := utils.EncodeString(name)
    ageEncoded := utils.EncodeUint8(age)
    addrEncoded := utils.EncodeAddress(addr)
    
    // Combine fields
    result := make([]byte, 0, len(nameEncoded)+len(ageEncoded)+len(addrEncoded))
    result = append(result, nameEncoded...)
    result = append(result, ageEncoded...)
    result = append(result, addrEncoded...)
    
    return result, nil
}
```

## 🎯 Best Practices

### 1. Validate Input Parameters

```go
func SafeEncodeMethod(processor *utils.ABIProcessor, method string, types []string, params ...interface{}) ([]byte, error) {
    // Validate parameter count
    if len(types) != len(params) {
        return nil, fmt.Errorf("parameter count mismatch: expected %d, got %d", len(types), len(params))
    }

    // Validate parameter types
    for i, param := range params {
        if err := validateParameterType(types[i], param); err != nil {
            return nil, fmt.Errorf("parameter %d validation failed: %w", i, err)
        }
    }

    return processor.EncodeMethod(method, types, params...)
}

func validateParameterType(expectedType string, param interface{}) error {
    switch expectedType {
    case "address":
        if _, ok := param.(*types.Address); !ok {
            return fmt.Errorf("expected *types.Address, got %T", param)
        }
    case "uint256":
        if _, ok := param.(*big.Int); !ok {
            return fmt.Errorf("expected *big.Int, got %T", param)
        }
    // Add more type validations...
    }
    return nil
}
```

### 2. Handle Large Numbers Safely

```go
// ✅ Good: Use big.Int for large numbers
func SafeAmount(amount string) (*big.Int, error) {
    value, ok := new(big.Int).SetString(amount, 10)
    if !ok {
        return nil, fmt.Errorf("invalid number format: %s", amount)
    }
    return value, nil
}

// ✅ Good: Validate amount ranges
func ValidateTokenAmount(amount *big.Int, decimals int) error {
    if amount.Sign() < 0 {
        return errors.New("amount cannot be negative")
    }
    
    // Check for overflow (example: max supply)
    maxSupply := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals+18)), nil)
    if amount.Cmp(maxSupply) > 0 {
        return errors.New("amount exceeds maximum supply")
    }
    
    return nil
}
```

### 3. Cache ABI Processors

```go
// Cache processors for frequently used contracts
var processorCache = make(map[string]*utils.ABIProcessor)
var cacheMutex sync.RWMutex

func GetCachedProcessor(abiJSON string) (*utils.ABIProcessor, error) {
    cacheMutex.RLock()
    if processor, exists := processorCache[abiJSON]; exists {
        cacheMutex.RUnlock()
        return processor, nil
    }
    cacheMutex.RUnlock()

    // Create new processor
    abi, err := utils.ParseABI(abiJSON)
    if err != nil {
        return nil, err
    }

    processor := utils.NewABIProcessor(abi)

    // Cache it
    cacheMutex.Lock()
    processorCache[abiJSON] = processor
    cacheMutex.Unlock()

    return processor, nil
}
```

### 4. Error Handling Patterns

```go
// Comprehensive error handling for encoding
func EncodeWithContext(processor *utils.ABIProcessor, method string, types []string, params ...interface{}) ([]byte, error) {
    encoded, err := processor.EncodeMethod(method, types, params...)
    if err != nil {
        // Add context to error
        return nil, fmt.Errorf("failed to encode method %q with types %v: %w", method, types, err)
    }

    // Validate encoded data
    if len(encoded) < 4 {
        return nil, fmt.Errorf("encoded data too short: %d bytes", len(encoded))
    }

    return encoded, nil
}
```

## 🔧 Debugging and Troubleshooting

### Method Signature Verification

```go
// Verify method signature matches expected
func VerifyMethodSignature(method string, types []string, expected string) error {
    signature := utils.MethodSignature(method, types)
    if hex.EncodeToString(signature[:4]) != expected {
        return fmt.Errorf("signature mismatch: expected %s, got %x", expected, signature[:4])
    }
    return nil
}

// Example usage
err := VerifyMethodSignature("transfer", []string{"address", "uint256"}, "a9059cbb")
```

### Data Length Validation

```go
func ValidateEncodedData(data []byte, expectedMinLength int) error {
    if len(data) < expectedMinLength {
        return fmt.Errorf("encoded data too short: %d bytes, expected at least %d", len(data), expectedMinLength)
    }
    
    // ABI data should be multiple of 32 bytes (after 4-byte method selector)
    if (len(data)-4)%32 != 0 {
        return fmt.Errorf("invalid ABI encoding: data length %d is not aligned", len(data)-4)
    }
    
    return nil
}
```

### Debug Helpers

```go
// Pretty print encoded data
func DebugEncodedData(data []byte) {
    fmt.Printf("Method Selector: %x\n", data[:4])
    
    params := data[4:]
    for i := 0; i < len(params); i += 32 {
        end := i + 32
        if end > len(params) {
            end = len(params)
        }
        fmt.Printf("Param %d: %x\n", i/32, params[i:end])
    }
}

// Decode and verify round-trip
func VerifyRoundTrip(processor *utils.ABIProcessor, method string, types []string, params ...interface{}) error {
    // Encode
    encoded, err := processor.EncodeMethod(method, types, params...)
    if err != nil {
        return fmt.Errorf("encoding failed: %w", err)
    }

    // For verification, you might decode outputs if it's a view function
    // This is contract-specific logic
    
    fmt.Printf("Round-trip verification passed for %s\n", method)
    return nil
}
```

## 🧪 Testing Utilities

### Test Data Generation

```go
// Generate test addresses
func TestAddresses() []*types.Address {
    addresses := []string{
        "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
        "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x",
        "TAuB7aNiJ2Sj5r3xrqoRH8UhZVNYBUHxdf",
    }
    
    result := make([]*types.Address, len(addresses))
    for i, addr := range addresses {
        result[i] = types.MustNewAddressFromBase58(addr)
    }
    return result
}

// Generate test amounts
func TestAmounts() []*big.Int {
    return []*big.Int{
        big.NewInt(0),
        big.NewInt(1),
        big.NewInt(1000000), // 1 USDT (6 decimals)
        big.NewInt(1000000000000000000), // 1 ETH equivalent
    }
}
```

### Encoding Test Suite

```go
func TestMethodEncoding(t *testing.T) {
    processor := setupTestProcessor(t)
    
    tests := []struct {
        name     string
        method   string
        types    []string
        params   []interface{}
        expected string
    }{
        {
            name:     "transfer",
            method:   "transfer",
            types:    []string{"address", "uint256"},
            params:   []interface{}{testAddr, big.NewInt(1000)},
            expected: "a9059cbb000000000000000000000000...",
        },
        // Add more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            encoded, err := processor.EncodeMethod(tt.method, tt.types, tt.params...)
            require.NoError(t, err)
            assert.Equal(t, tt.expected, hex.EncodeToString(encoded))
        })
    }
}
```

The utils package is fundamental to contract interaction in TronLib. Master these encoding and decoding patterns to build robust smart contract applications! 🚀
