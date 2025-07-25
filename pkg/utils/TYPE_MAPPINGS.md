# Smart Contract Type Mappings

This document provides a comprehensive reference for type conversions between Solidity types and Go types in the TronLib smart contract implementation. Understanding these mappings is crucial for proper type assertion, error handling, and data interpretation when working with smart contract data.

## Table of Contents
- [ABI Decoder Mappings](#abi-decoder-mappings)
- [Event Decoder Mappings](#event-decoder-mappings)
- [ABI Encoder Mappings](#abi-encoder-mappings)
- [Type Assertion Examples](#type-assertion-examples)
- [Error Handling Best Practices](#error-handling-best-practices)
- [Common Pitfalls](#common-pitfalls)

## ABI Decoder Mappings

Used when decoding smart contract call results via `Contract.DecodeResult()`.

| Solidity Type | Go Type | Notes |
|---------------|---------|-------|
| `uint256`, `uint128`, `uint64`, `uint32`, `uint16`, `uint8` | `string` | Converted via `bigInt.String()` to handle large numbers |
| `int256`, `int128`, `int64`, `int32`, `int16`, `int8` | `string` | Converted via `bigInt.String()` to handle large numbers |
| `address` | `string` | Converted from Ethereum hex to TRON Base58 format |
| `bytes`, `bytes32`, `bytes16`, `bytes8` | `string` | Hex encoded via `hex.EncodeToString()` |
| `string` | `string` | Unchanged |
| `bool` | `bool` | Unchanged |
| `type[]` | `[]interface{}` | Each element processed with base type mapping |

### Example Usage
```go
// Decoding a uint256 balance
result, err := contract.DecodeResult("balanceOf", data)
if err != nil {
    return err
}

// Type assert as string, not *big.Int
balance, ok := result.(string)
if !ok {
    return fmt.Errorf("expected string, got %T", result)
}

// Convert to big.Int for calculations
balanceBigInt, ok := new(big.Int).SetString(balance, 10)
if !ok {
    return fmt.Errorf("invalid number format: %s", balance)
}
```

## Event Decoder Mappings

Used when decoding event logs via `Contract.DecodeEventLog()`. All types are converted to strings for display purposes.

| Solidity Type | Go Type | String Format |
|---------------|---------|---------------|
| `address` | `string` | TRON Base58 format (e.g., "TRX9Uhjn...") |
| `uint256`, `uint128`, `uint64`, `uint32`, `uint16`, `uint8` | `string` | Decimal string (e.g., "1000000000") |
| `int256`, `int128`, `int64`, `int32`, `int16`, `int8` | `string` | Decimal string (e.g., "-1000000000") |
| `bytes`, `bytes32`, `bytes16`, `bytes8` | `string` | Hex string (e.g., "0x1234abcd") |
| `string` | `string` | Unchanged |
| `bool` | `string` | "true" or "false" |

### Example Usage
```go
// Decoding an event log
event, err := contract.DecodeEventLog(topics, data)
if err != nil {
    return err
}

for _, param := range event.Parameters {
    switch param.Type {
    case "uint256":
        // All event values are strings
        amount := param.Value // This is a string like "1000000000"
        
        // Convert to big.Int if needed for calculations
        amountBigInt, ok := new(big.Int).SetString(amount, 10)
        if !ok {
            return fmt.Errorf("invalid amount: %s", amount)
        }
        
    case "address":
        // Already in TRON Base58 format
        address := param.Value // This is a string like "TRX9Uhjn..."
        
    case "bool":
        // String representation of boolean
        isActive := param.Value == "true"
    }
}
```

## ABI Encoder Mappings

Used when encoding parameters for smart contract calls via `Contract.EncodeInput()`.

| Input Go Type | Solidity Type | Conversion |
|---------------|---------------|------------|
| `string` (TRON Base58) | `address` | Converted to Ethereum `common.Address` |
| `string` (Ethereum hex) | `address` | Parsed as hex to `common.Address` |
| `string` (number) | `uint256`, `int256`, etc. | Parsed to `*big.Int` |
| `*big.Int` | `uint256`, `int256`, etc. | Used directly |
| `int64`, `uint64`, etc. | `uint256`, `int256`, etc. | Converted to `*big.Int` |
| `bool` | `bool` | Used directly |
| `string` | `string` | Used directly |
| `[]byte` | `bytes`, `bytes32`, etc. | Used directly |
| `string` (hex) | `bytes`, `bytes32`, etc. | Decoded from hex |
| `[]interface{}` | `type[]` | Each element converted per base type |

### Example Usage
```go
// Encoding parameters for a contract call
data, err := contract.EncodeInput("transfer", 
    "TRX9Uhjn...",  // address (TRON Base58)
    "1000000000",   // uint256 amount (string)
)
if err != nil {
    return err
}

// Alternative with *big.Int
amount := new(big.Int).SetUint64(1000000000)
data, err := contract.EncodeInput("transfer", 
    "TRX9Uhjn...",  // address
    amount,         // *big.Int
)
```

## Type Assertion Examples

### Single Return Value
```go
// For methods returning one value
result, err := contract.DecodeResult("balanceOf", data)
if err != nil {
    return err
}

// uint256 returns as string
balance, ok := result.(string)
if !ok {
    return fmt.Errorf("expected string, got %T", result)
}

// bool returns as bool
isActive, ok := result.(bool)
if !ok {
    return fmt.Errorf("expected bool, got %T", result)
}
```

### Multiple Return Values
```go
// For methods returning multiple values
result, err := contract.DecodeResult("getInfo", data)
if err != nil {
    return err
}

// Multiple returns come as map[string]interface{}
resultMap, ok := result.(map[string]interface{})
if !ok {
    return fmt.Errorf("expected map, got %T", result)
}

// Access individual values
name, ok := resultMap["name"].(string)
if !ok {
    return fmt.Errorf("expected string for name")
}

balance, ok := resultMap["balance"].(string) // uint256 as string
if !ok {
    return fmt.Errorf("expected string for balance")
}
```

### Array Return Values
```go
// For methods returning arrays
result, err := contract.DecodeResult("getAddresses", data)
if err != nil {
    return err
}

// Arrays come as []interface{}
addresses, ok := result.([]interface{})
if !ok {
    return fmt.Errorf("expected slice, got %T", result)
}

// Process each element
for i, addr := range addresses {
    address, ok := addr.(string)
    if !ok {
        return fmt.Errorf("expected string at index %d, got %T", i, addr)
    }
    // address is now a TRON Base58 string
}
```

## Error Handling Best Practices

### 1. Always Check Type Assertions
```go
// ❌ Bad - can panic
balance := result.(string)

// ✅ Good - safe type assertion
balance, ok := result.(string)
if !ok {
    return fmt.Errorf("expected string, got %T", result)
}
```

### 2. Validate Number Strings
```go
// ❌ Bad - doesn't handle invalid numbers
amount, _ := new(big.Int).SetString(balance, 10)

// ✅ Good - validates number format
amount, ok := new(big.Int).SetString(balance, 10)
if !ok {
    return fmt.Errorf("invalid number format: %s", balance)
}
```

### 3. Handle Different Return Patterns
```go
func safeDecodeBalance(contract *Contract, data []byte) (*big.Int, error) {
    result, err := contract.DecodeResult("balanceOf", data)
    if err != nil {
        return nil, err
    }

    // Handle single return value
    if balance, ok := result.(string); ok {
        amount, ok := new(big.Int).SetString(balance, 10)
        if !ok {
            return nil, fmt.Errorf("invalid balance format: %s", balance)
        }
        return amount, nil
    }

    // Handle multiple return values
    if resultMap, ok := result.(map[string]interface{}); ok {
        if balance, ok := resultMap["balance"].(string); ok {
            amount, ok := new(big.Int).SetString(balance, 10)
            if !ok {
                return nil, fmt.Errorf("invalid balance format: %s", balance)
            }
            return amount, nil
        }
    }

    return nil, fmt.Errorf("unexpected result type: %T", result)
}
```

## Common Pitfalls

### 1. Expecting *big.Int Instead of string
```go
// ❌ Wrong - will panic
balance := result.(*big.Int)

// ✅ Correct - numeric types are strings
balance, ok := result.(string)
if !ok {
    return fmt.Errorf("expected string, got %T", result)
}
balanceBigInt, _ := new(big.Int).SetString(balance, 10)
```

### 2. Not Handling Multiple Return Values
```go
// ❌ Wrong - assumes single return value
name := result.(string)

// ✅ Correct - check if it's a map first
if resultMap, ok := result.(map[string]interface{}); ok {
    name = resultMap["name"].(string)
} else {
    name = result.(string)
}
```

### 3. Forgetting Address Format Conversion
```go
// ❌ Wrong - addresses are converted to TRON format
ethAddr := common.HexToAddress(result.(string))

// ✅ Correct - addresses are already in TRON Base58 format
tronAddr := result.(string) // Already "TRX9Uhjn..." format
```

### 4. Event vs Contract Result Confusion
```go
// ❌ Wrong - events return all values as strings
eventBool := eventParam.Value.(bool)

// ✅ Correct - event values are always strings
eventBool := eventParam.Value == "true"
```

### 5. Array Element Type Assumptions
```go
// ❌ Wrong - assumes specific element type
addresses := result.([]string)

// ✅ Correct - arrays are []interface{}
addressesSlice := result.([]interface{})
for _, addr := range addressesSlice {
    address := addr.(string)
    // Process address
}
```

## Summary

- **Numeric types** (uint/int variants) → `string` in decoder, events
- **Addresses** → `string` in TRON Base58 format
- **Bytes** → `string` in hex format
- **Booleans** → `bool` in decoder, `string` in events
- **Arrays** → `[]interface{}` with element type conversion
- **Multiple returns** → `map[string]interface{}`
- **Always use safe type assertions** with error checking
- **Validate number strings** before converting to big.Int
- **Remember event values are always strings**