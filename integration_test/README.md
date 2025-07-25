# TronLib Integration Tests

This directory contains comprehensive integration tests for TronLib that validate API functionality against real TRON network data using gRPC endpoints.

## Overview

The integration tests are designed with a **gRPC-first approach** that:

- Uses only protobuf getter methods for field access
- Validates actual gRPC response structures 
- Gracefully handles differences between HTTP and gRPC endpoints
- Focuses on strongly-typed Go testing without complex data conversions
- Provides detailed logging for human verification and debugging

## Test Structure

### Mainnet Tests (`tests/mainnet_test.go`)

**Read-only tests against TRON mainnet via gRPC endpoint `127.0.0.1:50051`**

#### TestMainnetGetAccount
- **Purpose**: Validates `GetAccount` API responses for known mainnet addresses
- **Test Addresses**:
  - `TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g` - Account with assets and permissions
  - `TFNyPYvjWSMePXHTzf7TfWD7k61yfpugxc` - Another test account
- **Validation**: Uses `account.Get*()` methods to validate:
  - Basic account properties (address, balance, timestamps)
  - TRC10 assets via `GetAssetV2()`
  - Network settings via `GetNetWindowSize()`, `GetNetWindowOptimized()`
  - Account resources via `GetAccountResource()`
  - Permissions via `GetOwnerPermission()`, `GetActivePermission()`
  - Frozen resources via `GetFrozenV2()`

#### TestMainnetGetAccountResource  
- **Purpose**: Validates `GetAccountResource` API responses
- **Validation**: Uses `resource.Get*()` methods to validate:
  - Network bandwidth usage and limits
  - Energy usage and limits  
  - TRON Power (voting power) metrics
  - Storage usage and limits
  - Asset-specific network usage
  - Global network constants

#### TestMainnetGetAccountNet
- **Purpose**: Validates `GetAccountNet` API responses  
- **Validation**: Uses `net.Get*()` methods to validate:
  - Free network bandwidth allocation
  - Network usage and limits
  - Asset-specific network data
  - Global network constants

## Key Design Principles

### 1. gRPC-Only Field Access
```go
// ✅ Correct: Use getter methods
balance := account.GetBalance()
assets := account.GetAssetV2()

// ❌ Incorrect: Direct field access
balance := account.Balance  // May not exist in gRPC response
assets := account.AssetV2   // May have different structure
```

### 2. Graceful Field Handling
```go
// Handle optional fields gracefully
assetV2 := account.GetAssetV2()
if assetV2 != nil && len(assetV2) > 0 {
    t.Logf("Found %d TRC10 assets", len(assetV2))
    // Process assets...
} else {
    t.Logf("No TRC10 assets found")
}
```

### 3. Non-Negative Validation
```go
// Validate that numeric fields are non-negative
assert.GreaterOrEqual(t, balance, int64(0), "Balance should be non-negative")
```

### 4. Detailed Logging
```go
// Provide comprehensive logging for human verification
t.Logf("Account balance: %d SUN", balance)
t.Logf("Network bandwidth utilization: %.1f%%", utilization)
```

## Running Tests

### From IDE/Code Editor
```bash
# Run all integration tests
go test ./integration_test/tests/...

# Run specific test
go test -run TestMainnetGetAccount ./integration_test/tests/

# Run with verbose output
go test -v ./integration_test/tests/
```

### From Command Line
```bash
# Navigate to project root
cd /path/to/tronlib

# Run integration tests
go test -v ./integration_test/tests/

# Run with timeout
go test -timeout 60s -v ./integration_test/tests/
```

### CI/CD Integration
```yaml
# Example GitHub Actions step
- name: Run Integration Tests
  run: |
    go test -v -timeout 60s ./integration_test/tests/
  env:
    TRON_ENDPOINT: "127.0.0.1:50051"
```

## Test Configuration

### Network Endpoint
- **Default**: `127.0.0.1:50051` (local TRON node)
- **Timeout**: 30 seconds per test
- **Protocol**: gRPC

### Test Data Sources
- **Mainnet Data**: Real TRON mainnet accounts with known characteristics
- **Validation**: Against actual gRPC response structures
- **No Mocking**: Tests validate real network behavior

## Available Protobuf Getter Methods

### Account (`*core.Account`)
```go
// Basic properties
GetAccountName() []byte
GetAddress() []byte  
GetBalance() int64
GetType() AccountType

// Assets
GetAsset() map[string]int64          // Legacy TRC10
GetAssetV2() map[string]int64        // Current TRC10
GetAssetOptimized() bool

// Network resources
GetNetUsage() int64
GetNetWindowSize() int64
GetNetWindowOptimized() bool
GetFreeNetUsage() int64
GetFreeAssetNetUsage() map[string]int64
GetFreeAssetNetUsageV2() map[string]int64

// Energy and bandwidth
GetAccountResource() *Account_AccountResource
GetTronPower() *Account_Frozen
GetOldTronPower() int64

// Permissions
GetOwnerPermission() *Permission
GetWitnessPermission() *Permission  
GetActivePermission() []*Permission

// Frozen resources
GetFrozen() []*Account_Frozen
GetFrozenV2() []*Account_FreezeV2
GetUnfrozenV2() []*Account_UnFreezeV2

// Timestamps
GetCreateTime() int64
GetLatestOprationTime() int64
GetLatestConsumeTime() int64
GetLatestConsumeFreeTime() int64
GetLatestWithdrawTime() int64
GetLatestAssetOperationTime() map[string]int64
GetLatestAssetOperationTimeV2() map[string]int64

// Contract-related
GetCode() []byte
GetCodeHash() []byte
GetIsWitness() bool
GetIsCommittee() bool

// Delegation
GetAcquiredDelegatedFrozenBalanceForBandwidth() int64
GetDelegatedFrozenBalanceForBandwidth() int64
GetDelegatedFrozenV2BalanceForBandwidth() int64
GetAcquiredDelegatedFrozenV2BalanceForBandwidth() int64
```

### AccountResourceMessage (`*api.AccountResourceMessage`)
```go
// Network bandwidth
GetFreeNetUsed() int64
GetFreeNetLimit() int64
GetNetUsed() int64
GetNetLimit() int64
GetTotalNetLimit() int64
GetTotalNetWeight() int64

// Asset network usage
GetAssetNetUsed() map[string]int64
GetAssetNetLimit() map[string]int64

// TRON Power (voting power)
GetTronPowerUsed() int64
GetTronPowerLimit() int64
GetTotalTronPowerWeight() int64

// Energy
GetEnergyUsed() int64
GetEnergyLimit() int64
GetTotalEnergyLimit() int64
GetTotalEnergyWeight() int64

// Storage
GetStorageUsed() int64
GetStorageLimit() int64
```

### AccountNetMessage (`*api.AccountNetMessage`)
```go
// Network bandwidth
GetFreeNetUsed() int64
GetFreeNetLimit() int64
GetNetUsed() int64
GetNetLimit() int64
GetTotalNetLimit() int64
GetTotalNetWeight() int64

// Asset network usage
GetAssetNetUsed() map[string]int64
GetAssetNetLimit() map[string]int64
```

## Error Handling

### Network Connectivity
```go
// Tests will fail gracefully if gRPC endpoint is unavailable
require.NoError(t, err, "GetAccount should succeed")
```

### Missing Fields
```go
// Handle fields that may not exist in all responses
if accountResource := account.GetAccountResource(); accountResource != nil {
    // Process account resource data
} else {
    t.Logf("No account resource found")
}
```

### Data Validation
```go
// Validate data ranges and relationships
assert.GreaterOrEqual(t, netUsed, int64(0), "Net used should be non-negative")
if netLimit > 0 {
    utilization := float64(netUsed) / float64(netLimit) * 100
    t.Logf("Network utilization: %.1f%%", utilization)
}
```

## Extending Tests

### Adding New Test Cases
1. Add new test case to appropriate test function
2. Use only protobuf getter methods for field access
3. Include graceful handling for optional fields
4. Add comprehensive logging for verification
5. Validate non-negative constraints where appropriate

### Adding New APIs
1. Create new test function following naming pattern `TestMainnet<APIName>`
2. Use `setupTestManager()` for client initialization
3. Include timeout context with `context.WithTimeout()`
4. Follow gRPC-only validation patterns

## Best Practices

1. **Always use getter methods**: `account.GetBalance()` not `account.Balance`
2. **Handle nil responses**: Check if optional fields exist before processing
3. **Validate data ranges**: Ensure numeric values meet expected constraints
4. **Log comprehensively**: Include detailed output for human verification
5. **Test real data**: Use actual mainnet addresses with known characteristics
6. **Fail gracefully**: Provide clear error messages when validation fails

## Troubleshooting

### Test Failures
- **Connection errors**: Verify gRPC endpoint `127.0.0.1:50051` is accessible
- **Field access errors**: Ensure using getter methods, not direct field access
- **Data validation errors**: Check if validation expectations match current network state

### Performance Issues  
- **Slow tests**: Increase timeout values or optimize network connectivity
- **Rate limiting**: Add delays between test cases if needed

### Data Inconsistencies
- **Changing balances**: Use relative validation (non-negative) rather than exact values
- **Network state**: Account for dynamic network conditions in validation logic