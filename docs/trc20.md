# 🪙 TRC20 Package Reference

The `trc20` package provides a comprehensive, user-friendly interface for interacting with TRC20 tokens on the TRON blockchain. It handles decimal precision, caches immutable properties, and provides all standard TRC20 operations.

## 📚 Learning Path

This document is part of the TronLib learning path:
1. [Quick Start Guide](quickstart.md) - Basic usage
2. [Architecture Overview](architecture.md) - Understanding the design
3. **TRC20 Package Reference** (this document) - Detailed TRC20 operations
4. [Other Package Documentation](../README.md#package-references) - Additional functionality
5. [API Reference](API_REFERENCE.md) - Complete function documentation

## 📋 Overview

The TRC20 package features:
- **Decimal Precision** - Automatic conversion between human-readable decimals and on-chain integers
- **Property Caching** - Immutable properties (name, symbol, decimals) cached for efficiency
- **Standard Operations** - All TRC20 standard methods (transfer, approve, allowance, etc.)
- **Type Safety** - Strong typing prevents common errors
- **Error Handling** - Comprehensive error types for different failure scenarios

## 🏗️ Core Components

### TRC20Manager

The `TRC20Manager` is the main interface for TRC20 operations:

```go
type TRC20Manager struct {
    client      lowlevel.ConnProvider
    address     *types.Address
    // Cached properties
    name        string
    symbol      string
    decimals    int
    // Internal state
}

// Create a new TRC20 manager.
//
// Prefer creating a manager via cli.TRC20Manager(addr) which returns an error
// if the contract cannot be queried. The deprecated cli.TRC20(addr) variant
// swallows that error and may return nil.
//
// NewManager is used by the client facade internally and when constructing a
// manager from a raw lowlevel.ConnProvider (e.g. in unit tests):
//
//	mgr, err := trc20.NewManager(connProvider, tokenAddr)
func NewManager(tronClient lowlevel.ConnProvider, contractAddress *types.Address) (*TRC20Manager, error)
```

## 🚀 Getting Started

### Basic Setup

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/shopspring/decimal"
    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/trc20"
    "github.com/kslamph/tronlib/pkg/types"
)

func main() {
    // Connect to TRON network
    cli, err := client.NewClient("grpc://grpc.trongrid.io:50051")
    if err != nil {
        log.Fatal(err)
    }
    defer cli.Close()

    // USDT contract address on mainnet
    usdtAddr, err := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
    if err != nil {
        log.Fatal(err)
    }

    // Create TRC20 manager
    // TRC20Manager() returns (*TRC20Manager, error); the deprecated TRC20() variant may return nil.
    trc20Mgr, err := cli.TRC20Manager(usdtAddr)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // The manager automatically fetches and caches token metadata
    name, _ := trc20Mgr.Name(ctx)
    symbol, _ := trc20Mgr.Symbol(ctx)
    decimals, _ := trc20Mgr.Decimals(ctx)

    fmt.Printf("Token: %s (%s) with %d decimals\n", name, symbol, decimals)
    // Output: Token: Tether USD (USDT) with 6 decimals
}
```

## 📊 Reading Token Information

### Token Metadata (Cached)

These properties are fetched once and cached for the lifetime of the manager:

```go
// Get token name (cached after first call)
name, err := trc20Mgr.Name(ctx)
if err != nil {
    log.Printf("Failed to get token name: %v", err)
}

// Get token symbol (cached after first call)
symbol, err := trc20Mgr.Symbol(ctx)
if err != nil {
    log.Printf("Failed to get token symbol: %v", err)
}

// Get token decimals (cached after first call)
decimals, err := trc20Mgr.Decimals(ctx)
if err != nil {
    log.Printf("Failed to get token decimals: %v", err)
}

fmt.Printf("Token: %s (%s), Decimals: %d\n", name, symbol, decimals)
```

### Total Supply

```go
// Get total supply (always fresh from network)
totalSupply, err := trc20Mgr.TotalSupply(ctx)
if err != nil {
    log.Printf("Failed to get total supply: %v", err)
} else {
    fmt.Printf("Total supply: %s %s\n", totalSupply.String(), symbol)
}
```

### Account Balances

```go
// Check balance for an address
holder, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

balance, err := trc20Mgr.BalanceOf(ctx, holder)
if err != nil {
    log.Printf("Failed to get balance: %v", err)
} else {
    fmt.Printf("Balance: %s %s\n", balance.String(), symbol)
}

// Check multiple balances
addresses := []*types.Address{
    types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"),
    types.MustNewAddressFromBase58("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x"),
}

for _, addr := range addresses {
    balance, err := trc20Mgr.BalanceOf(ctx, addr)
    if err != nil {
        fmt.Printf("Error getting balance for %s: %v\n", addr, err)
        continue
    }
    fmt.Printf("%s: %s %s\n", addr, balance.String(), symbol)
}
```

### Allowances

```go
// Check allowance (how much spender can transfer from owner)
owner, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
spender, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

allowance, err := trc20Mgr.Allowance(ctx, owner, spender)
if err != nil {
    log.Printf("Failed to get allowance: %v", err)
} else {
    fmt.Printf("Allowance: %s %s\n", allowance.String(), symbol)
}
```

## 💸 Transfer Operations

### Direct Transfer

```go
// Transfer tokens directly from your account
from, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
to, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

// Transfer 10.5 tokens (automatically handles decimals)
amount := decimal.NewFromFloat(10.5)

// Build transfer transaction
// Transfer returns (txExt, err) — a TransactionExtention, not a txid string.
ext, err := trc20Mgr.Transfer(ctx, from, to, amount)
if err != nil {
    log.Fatalf("Failed to build transfer: %v", err)
}

// Sign and broadcast using the client
result, err := cli.SignAndBroadcast(ctx, ext, opts, signer)
if err != nil {
    log.Fatalf("Transfer failed: %v", err)
}

fmt.Printf("Transaction ID: %s\n", result.TxID)
```

### Transfer with Signing and Broadcasting

```go
// Complete transfer workflow
amount := decimal.NewFromFloat(25.75)

// Build transaction (returns a TransactionExtention)
ext, err := trc20Mgr.Transfer(ctx, from, to, amount)
if err != nil {
    log.Fatalf("Failed to build transfer: %v", err)
}

// Create signer
signer, err := signer.NewPrivateKeySigner("your-private-key")
if err != nil {
    log.Fatal(err)
}

// Configure broadcast options for TRC20 (higher energy needed)
opts := client.DefaultBroadcastOptions()
opts.FeeLimit = 50_000_000  // 50 TRX max fee for TRC20 operations
opts.WaitForReceipt = true
opts.WaitTimeout = 30 * time.Second

// Sign and broadcast
result, err := cli.SignAndBroadcast(ctx, ext, opts, signer)
if err != nil {
    log.Fatalf("Transfer failed: %v", err)
}

fmt.Printf("✅ Transfer successful!\n")
fmt.Printf("Transaction ID: %s\n", result.TxID)
fmt.Printf("Energy used: %d\n", result.EnergyUsage)
fmt.Printf("Success: %v\n", result.Success)
```

### Batch Transfers

```go
// Perform multiple transfers efficiently
type TransferRequest struct {
    To     *types.Address
    Amount decimal.Decimal
}

func PerformBatchTransfers(ctx context.Context, cli *client.Client, trc20Mgr *trc20.TRC20Manager, from *types.Address, transfers []TransferRequest, s signer.Signer) error {
    for i, transfer := range transfers {
        fmt.Printf("Processing transfer %d/%d to %s: %s\n",
            i+1, len(transfers), transfer.To, transfer.Amount)

        ext, err := trc20Mgr.Transfer(ctx, from, transfer.To, transfer.Amount)
        if err != nil {
            return fmt.Errorf("failed to build transfer %d: %w", i, err)
        }

        // Sign and broadcast each transaction
        opts := client.DefaultBroadcastOptions()
        opts.FeeLimit = 50_000_000
        opts.WaitForReceipt = true
        result, err := cli.SignAndBroadcast(ctx, ext, opts, s)
        if err != nil {
            return fmt.Errorf("failed to broadcast transfer %d: %w", i, err)
        }

        fmt.Printf("  ✅ Success: %s\n", result.TxID)
    }

    return nil
}

// Usage
transfers := []TransferRequest{
    {To: to1, Amount: decimal.NewFromFloat(10.0)},
    {To: to2, Amount: decimal.NewFromFloat(20.0)},
    {To: to3, Amount: decimal.NewFromFloat(15.5)},
}

err := PerformBatchTransfers(ctx, cli, trc20Mgr, from, transfers, signer)
```

## 🔐 Approval Operations

### Basic Approval

```go
// Approve spender to transfer tokens on your behalf
owner, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
spender, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

// Approve specific amount
approveAmount := decimal.NewFromFloat(100.0)

ext, err := trc20Mgr.Approve(ctx, owner, spender, approveAmount)
if err != nil {
    log.Fatalf("Failed to build approval: %v", err)
}

// Sign and broadcast approval
result, err := cli.SignAndBroadcast(ctx, ext, opts, signer)
if err != nil {
    log.Fatalf("Approval failed: %v", err)
}

fmt.Printf("✅ Approval successful: %s\n", result.TxID)
```

### Unlimited Approval

```go
// Approve unlimited amount (common pattern for DEX interactions).
// There is no dedicated ApproveUnlimited helper — use the maximum uint256
// value directly:
maxAmount := decimal.NewFromString("115792089237316195423570985008687907853269984665640564039457584007913129639935")

ext, err := trc20Mgr.Approve(ctx, owner, spender, maxAmount)
if err != nil {
    log.Fatalf("Failed to build unlimited approval: %v", err)
}
```

### Safe Approval Pattern

```go
// Safe approval pattern: set to 0 first, then to desired amount
// This prevents certain attack vectors

func SafeApprove(ctx context.Context, cli *client.Client, trc20Mgr *trc20.TRC20Manager, owner, spender *types.Address, amount decimal.Decimal, s signer.Signer) error {
    // First, check current allowance
    currentAllowance, err := trc20Mgr.Allowance(ctx, owner, spender)
    if err != nil {
        return fmt.Errorf("failed to check current allowance: %w", err)
    }

    // If there's an existing allowance and we're not setting to 0, reset first
    if !currentAllowance.IsZero() && !amount.IsZero() {
        fmt.Println("Resetting allowance to 0 first...")

        ext, err := trc20Mgr.Approve(ctx, owner, spender, decimal.Zero)
        if err != nil {
            return fmt.Errorf("failed to reset allowance: %w", err)
        }

        opts := client.DefaultBroadcastOptions()
        result, err := cli.SignAndBroadcast(ctx, ext, opts, s)
        if err != nil {
            return fmt.Errorf("failed to broadcast reset: %w", err)
        }

        fmt.Printf("Reset successful: %s\n", result.TxID)
    }

    // Now set the desired allowance
    ext, err = trc20Mgr.Approve(ctx, owner, spender, amount)
    if err != nil {
        return fmt.Errorf("failed to set allowance: %w", err)
    }

    opts := client.DefaultBroadcastOptions()
    result, err := cli.SignAndBroadcast(ctx, ext, opts, s)
    if err != nil {
        return fmt.Errorf("failed to broadcast approval: %w", err)
    }

    fmt.Printf("Approval successful: %s\n", result.TxID)
    return nil
}
```

## 🔄 TransferFrom (approve + transferFrom pattern)

The TRC20 package does not expose a `TransferFrom` helper. To perform a
`transferFrom` on behalf of a token owner, use the `smartcontract` package
with the standard TRC20 ABI:

```go
// Requires a prior Approve from the token owner.
owner, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
spender, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
recipient, _ := types.NewAddress("TAuB7aNiJ2Sj5r3xrqoRH8UhZVNYBUHxdf")

transferAmount := decimal.NewFromFloat(50.0)

// Check allowance first
allowance, err := trc20Mgr.Allowance(ctx, owner, spender)
if err != nil {
    log.Fatalf("Failed to check allowance: %v", err)
}

if allowance.LessThan(transferAmount) {
    log.Fatalf("Insufficient allowance: have %s, need %s", allowance, transferAmount)
}

// Option A: Use smartcontract.Instance with the token ABI
// Option B: Use a helper contract that calls transferFrom on behalf of the spender
```

## 💱 Decimal Conversion Utilities

The TRC20 package uses `shopspring/decimal` for precise arithmetic and provides utilities for converting between human-readable decimals and on-chain integer values.

### Manual Conversion Functions

```go
// Convert human decimal to on-chain integer (wei)
humanAmount := decimal.NewFromFloat(12.34)
var decimals uint8 = 6 // USDT has 6 decimals

weiAmount, err := trc20.ToWei(humanAmount, decimals)
if err != nil {
    log.Fatalf("Conversion error: %v", err)
}
fmt.Printf("Human: %s, On-chain: %s\n", humanAmount, weiAmount)
// Output: Human: 12.34, On-chain: 12340000

// Convert on-chain integer back to human decimal
backToHuman, err := trc20.FromWei(weiAmount, decimals)
if err != nil {
    log.Fatalf("Conversion error: %v", err)
}
fmt.Printf("Round-trip: %s\n", backToHuman)
// Output: Round-trip: 12.34
```

### Working with Different Token Decimals

```go
// Different tokens have different decimal places
tokens := map[string]struct {
    address  string
    decimals uint8
}{
    "USDT": {"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", 6},
    "USDC": {"TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8", 6},
    "WTRX": {"TNUC9Qb1rRpS5CbWLmNMxXBjyFoydXjWFR", 6},
    "JST":  {"TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9", 18},
}

func ConvertAmount(tokenSymbol string, humanAmount decimal.Decimal) (*big.Int, error) {
    token, exists := tokens[tokenSymbol]
    if !exists {
        return nil, fmt.Errorf("unknown token: %s", tokenSymbol)
    }

    return trc20.ToWei(humanAmount, token.decimals)
}

// Usage
amount := decimal.NewFromFloat(100.5)
jstWei, _ := ConvertAmount("JST", amount)   // 18 decimals
usdtWei, _ := ConvertAmount("USDT", amount) // 6 decimals

fmt.Printf("JST wei: %s\n", jstWei)   // 100500000000000000000
fmt.Printf("USDT wei: %s\n", usdtWei) // 10050000
```

### Precision Handling

```go
// Handle precision carefully with financial amounts
func SafeDecimalFromString(s string) (decimal.Decimal, error) {
    d, err := decimal.NewFromString(s)
    if err != nil {
        return decimal.Zero, fmt.Errorf("invalid decimal: %s", s)
    }

    // Limit precision to avoid issues
    if d.Exponent() < -18 {
        return decimal.Zero, fmt.Errorf("precision too high: %s", s)
    }

    return d, nil
}

// Example: User input handling
userInput := "123.456789123456789123" // Very high precision
amount, err := SafeDecimalFromString(userInput)
if err != nil {
    log.Printf("Invalid amount: %v", err)
} else {
    // Truncate to token's precision
    tokenDecimals := 6
    truncated := amount.Truncate(int32(tokenDecimals))
    fmt.Printf("Original: %s, Truncated: %s\n", amount, truncated)
}
```

## 🎯 Advanced Patterns

### Multi-Token Manager

For constructing TRC20Managers from a live connection, prefer
`cli.TRC20Manager(addr)` which handles errors. The example below uses
`NewManager` with a raw `ConnProvider` for illustration.

```go
type MultiTokenManager struct {
    cp       lowlevel.ConnProvider
    managers map[string]*trc20.TRC20Manager
}

func NewMultiTokenManager(cp lowlevel.ConnProvider) *MultiTokenManager {
    return &MultiTokenManager{
        cp:       cp,
        managers: make(map[string]*trc20.TRC20Manager),
    }
}

func (m *MultiTokenManager) GetManager(tokenAddress string) (*trc20.TRC20Manager, error) {
    if mgr, exists := m.managers[tokenAddress]; exists {
        return mgr, nil
    }

    addr, err := types.NewAddress(tokenAddress)
    if err != nil {
        return nil, err
    }

    mgr, err := trc20.NewManager(m.cp, addr)
    if err != nil {
        return nil, err
    }

    m.managers[tokenAddress] = mgr
    return mgr, nil
}

func (m *MultiTokenManager) GetBalance(ctx context.Context, tokenAddress, holderAddress string) (decimal.Decimal, error) {
    mgr, err := m.GetManager(tokenAddress)
    if err != nil {
        return decimal.Zero, err
    }

    holder, err := types.NewAddress(holderAddress)
    if err != nil {
        return decimal.Zero, err
    }

    return mgr.BalanceOf(ctx, holder)
}
```

### Portfolio Tracker

For production code use `cli.TRC20Manager(addr)` — the example below uses
`NewManager` with a `ConnProvider` to illustrate the pattern.

```go
type TokenBalance struct {
    Symbol   string
    Address  string
    Balance  decimal.Decimal
    Decimals int
}

func GetPortfolio(ctx context.Context, cp lowlevel.ConnProvider, holderAddr *types.Address, tokens []string) ([]TokenBalance, error) {
    var portfolio []TokenBalance

    for _, tokenAddr := range tokens {
        addr, err := types.NewAddress(tokenAddr)
        if err != nil {
            continue // Skip invalid addresses
        }

        mgr, err := trc20.NewManager(cp, addr)
        if err != nil {
            continue // Skip if can't create manager
        }

        // Get token info
        symbol, _ := mgr.Symbol(ctx)
        decimals, _ := mgr.Decimals(ctx)
        balance, err := mgr.BalanceOf(ctx, holderAddr)
        if err != nil {
            continue // Skip if can't get balance
        }

        if !balance.IsZero() {
            portfolio = append(portfolio, TokenBalance{
                Symbol:   symbol,
                Address:  tokenAddr,
                Balance:  balance,
                Decimals: decimals,
            })
        }
    }

    return portfolio, nil
}
```

## 🚨 Error Handling

### Common Error Sentinels

Package-level error sentinels live in `pkg/types`, not in `pkg/trc20`:

```go
// types.ErrInvalidAddress   — invalid address format or value
// types.ErrInvalidAmount    — invalid amount value
// types.ErrInvalidContract  — invalid contract address/ABI
// types.ErrInvalidParameter — invalid parameter value
// types.ErrInvalidTransaction — invalid transaction format
```

### Usage with error checking

```go
balance, err := trc20Mgr.BalanceOf(ctx, holder)
if err != nil {
    if errors.Is(err, types.ErrInvalidAddress) {
        log.Println("The token contract address is invalid")
    } else {
        log.Printf("Unexpected error: %v", err)
    }
    return
}
```

### Robust Error Handling Pattern

```go
func SafeTransfer(ctx context.Context, cli *client.Client, mgr *trc20.TRC20Manager, from, to *types.Address, amount decimal.Decimal, s signer.Signer) error {
    // Validate amount
    if amount.IsNegative() {
        return fmt.Errorf("amount cannot be negative: %s", amount)
    }
    if amount.IsZero() {
        return fmt.Errorf("amount cannot be zero")
    }

    // Check balance first
    balance, err := mgr.BalanceOf(ctx, from)
    if err != nil {
        return fmt.Errorf("failed to check balance: %w", err)
    }

    if balance.LessThan(amount) {
        return fmt.Errorf("insufficient balance: have %s, need %s", balance, amount)
    }

    // Build transaction
    ext, err := mgr.Transfer(ctx, from, to, amount)
    if err != nil {
        return fmt.Errorf("failed to build transfer transaction: %w", err)
    }

    // Sign and broadcast
    opts := client.DefaultBroadcastOptions()
    opts.FeeLimit = 50_000_000
    opts.WaitForReceipt = true
    result, err := cli.SignAndBroadcast(ctx, ext, opts, s)
    if err != nil {
        return fmt.Errorf("failed to broadcast transfer: %w", err)
    }

    fmt.Printf("Transfer successful: %s\n", result.TxID)
    return nil
}
```

## 🧪 Testing

### Mock Testing

```go
// Test with mock token contract
func TestTRC20Transfer(t *testing.T) {
    // Setup test addresses
    from := types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
    to := types.MustNewAddressFromBase58("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

    // Mock ConnProvider (implement GetConnection/ReturnConnection/GetTimeout)
    mockCP := &MockConnProvider{}
    tokenAddr := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")

    mgr, err := trc20.NewManager(mockCP, tokenAddr)
    require.NoError(t, err)

    // Test transfer
    amount := decimal.NewFromFloat(10.5)
    ext, err := mgr.Transfer(context.Background(), from, to, amount)
    require.NoError(t, err)
    require.NotNil(t, ext)
}
```

### Integration Testing

```go
// Test against real testnet (requires the integration build tag; see TESTING_GUIDE.md)
func TestRealTRC20Operations(t *testing.T) {
    cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
    require.NoError(t, err)
    defer cli.Close()

    // Use a test token on Nile testnet
    tokenAddr := types.MustNewAddressFromBase58("test-token-address-here")
    mgr, err := trc20.NewManager(cli, tokenAddr)
    require.NoError(t, err)

    ctx := context.Background()

    // Test reading operations
    name, err := mgr.Name(ctx)
    require.NoError(t, err)
    require.NotEmpty(t, name)

    symbol, err := mgr.Symbol(ctx)
    require.NoError(t, err)
    require.NotEmpty(t, symbol)

    decimals, err := mgr.Decimals(ctx)
    require.NoError(t, err)
    require.GreaterOrEqual(t, decimals, 0)
}
```

The TRC20 package makes token operations simple and safe. Use these patterns to build robust token-based applications on TRON! 🚀
