# High-Level Package Design Guidelines

## Overview
This document provides design guidelines for implementing high-level packages in the TRON SDK. These guidelines ensure consistency, simplicity, and maintainability across all high-level packages.

## Core Principles

### 1. One gRPC Call Per Function
- Each high-level function should map to exactly ONE lowlevel gRPC call
- Do NOT combine multiple gRPC calls in a single high-level function
- Keep functions focused and atomic

### 2. Amount Representation
- All amounts in TRON are representable in int64 (SUN units)
- Do NOT use *big.Int for amounts - use int64 directly
- TRON protocol uses int64 for all amount fields

### 3. Function Responsibilities
High-level functions should:
- ✅ Validate inputs using utils package
- ✅ Encode/decode data properly
- ✅ Prepare and construct gRPC call parameters
- ✅ Call exactly ONE lowlevel gRPC function
- ✅ Return the direct result from lowlevel call
- ❌ Combine multiple gRPC calls
- ❌ Implement complex business workflows
- ❌ Create composite data structures from multiple calls

### 4. Workflow Separation
- Transaction signing workflows are handled separately
- Broadcasting workflows are handled separately  
- Cost estimation workflows are handled separately
- High-level packages focus on individual operations only

### 5. Return Types
- Return the direct gRPC response types from lowlevel calls
- Do NOT create custom composite structs that combine multiple gRPC responses
- Keep return types simple and aligned with TRON protocol

## Implementation Pattern

```go
// ✅ CORRECT: One gRPC call, proper validation, direct return
func (m *Manager) GetAccount(ctx context.Context, address string) (*core.Account, error) {
    // 1. Validate inputs
    addr, err := utils.ValidateAddress(address)
    if err != nil {
        return nil, fmt.Errorf("invalid address: %w", err)
    }
    
    // 2. Prepare gRPC parameters
    req := &core.Account{
        Address: addr.Bytes(),
    }
    
    // 3. Call exactly ONE lowlevel function
    return lowlevel.GetAccount(m.client, ctx, req)
}

// ❌ INCORRECT: Combines multiple gRPC calls
func (m *Manager) GetAccountInfo(ctx context.Context, address string) (*AccountInfo, error) {
    account, _ := lowlevel.GetAccount(...)           // Multiple
    accountNet, _ := lowlevel.GetAccountNet(...)     // gRPC calls
    accountResource, _ := lowlevel.GetAccountResource(...) // in one function
    // ... combine results
}
```

## Package Structure

Each high-level package should follow this structure:

```
pkg/[package_name]/
├── manager.go      # Main functionality with Manager struct
├── manager_test.go # Comprehensive tests
└── [optional additional files for complex packages]
```

## Manager Pattern

- Each package provides a `Manager` struct that wraps the client
- Manager methods are the public API for the package
- Manager constructor: `NewManager(client *client.Client) *Manager`

## Error Handling

- Use utils package for input validation
- Return descriptive errors with context
- Wrap lowlevel errors with operation context
- Follow Go error handling best practices

## Testing Guidelines

- Test input validation thoroughly
- Test with invalid inputs to verify error handling
- Network errors are expected in unit tests (no real TRON node)
- Focus on testing validation logic and parameter preparation

## Package Implementation Order

As specified, implement packages in this order:
1. ✅ account - Account operations (GetAccount, GetBalance, TransferTRX)
2. 🔄 network - Network and node information
3. ⏳ resources - Energy and bandwidth management  
4. ⏳ voting - Voting and witness operations
5. ⏳ smartcontract - Smart contract operations
6. ⏳ trc20 - TRC20 token operations

## Examples by Package

### Account Package
- `GetAccount(address)` → `lowlevel.GetAccount()`
- `GetAccountNet(address)` → `lowlevel.GetAccountNet()`  
- `GetAccountResource(address)` → `lowlevel.GetAccountResource()`
- `TransferTRX(from, to, amount)` → `lowlevel.CreateTransaction2()`

### Network Package (Future)
- `GetNodeInfo()` → `lowlevel.GetNodeInfo()`
- `GetChainParameters()` → `lowlevel.GetChainParameters()`
- `GetBlockByNumber(number)` → `lowlevel.GetBlockByNum2()`

### Resources Package (Future)
- `FreezeBalance(address, amount, resource)` → `lowlevel.FreezeBalance2()`
- `UnfreezeBalance(address, resource)` → `lowlevel.UnfreezeBalance2()`

## Anti-Patterns to Avoid

1. **Multi-call functions**: Functions that make multiple gRPC calls
2. **Composite structs**: Custom structs combining multiple gRPC responses
3. **Workflow functions**: Functions handling signing/broadcasting workflows
4. **BigInt usage**: Using *big.Int for amounts (use int64)
5. **Complex business logic**: High-level packages should be thin wrappers

## Notes for Future Implementation

- Always refer to this document when implementing new high-level packages
- Each package should be independently usable
- Keep the API surface small and focused
- Maintain consistency across all packages
- Update this document if new patterns emerge