# 🪙 TRC20 Package Reference

The `trc20` package provides a comprehensive, user-friendly interface for interacting with TRC20 tokens on the TRON blockchain. It handles decimal precision, caches immutable properties, and provides all standard TRC20 operations.

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
    client      Client
    address     *types.Address
    // Cached properties
    name        string
    symbol      string
    decimals    int
    // Internal state
}

// Create a new TRC20 manager
func NewManager(client Client, tokenAddress *types.Address) (*TRC20Manager, error)
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
    trc20Mgr, err := trc20.NewManager(cli, usdtAddr)
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
// Returns: transaction ID string, transaction object, error
txid, tx, err := trc20Mgr.Transfer(ctx, from, to, amount)
if err != nil {
    log.Fatalf("Failed to build transfer: %v", err)
}

fmt.Printf("Transaction built: %s\n", txid)

// Note: At this point, the transaction is built but not signed or broadcast
// You need to sign and broadcast it using the client
```

### Transfer with Signing and Broadcasting

```go
// Complete transfer workflow
amount := decimal.NewFromFloat(25.75)

// Build transaction
_, tx, err := trc20Mgr.Transfer(ctx, from, to, amount)
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
result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
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

func PerformBatchTransfers(ctx context.Context, trc20Mgr *trc20.TRC20Manager, from *types.Address, transfers []TransferRequest) error {
    for i, transfer := range transfers {
        fmt.Printf("Processing transfer %d/%d to %s: %s\n", 
            i+1, len(transfers), transfer.To, transfer.Amount)

        _, tx, err := trc20Mgr.Transfer(ctx, from, transfer.To, transfer.Amount)
        if err != nil {
            return fmt.Errorf("failed to build transfer %d: %w", i, err)
        }

        // Sign and broadcast each transaction
        result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
        if err != nil {
            return fmt.Errorf("failed to broadcast transfer %d: %w", i, err)
        }

        fmt.Printf("  ✅ Success: %s\n", result.TxID)
    }

    return nil
}

// Usage
transfers := []TransferRequest{
    {to1, decimal.NewFromFloat(10.0)},
    {to2, decimal.NewFromFloat(20.0)},
    {to3, decimal.NewFromFloat(15.5)},
}

err := PerformBatchTransfers(ctx, trc20Mgr, from, transfers)
```

## 🔐 Approval Operations

### Basic Approval

```go
// Approve spender to transfer tokens on your behalf
owner, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
spender, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

// Approve specific amount
approveAmount := decimal.NewFromFloat(100.0)

_, tx, err := trc20Mgr.Approve(ctx, owner, spender, approveAmount)
if err != nil {
    log.Fatalf("Failed to build approval: %v", err)
}

// Sign and broadcast approval
result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
if err != nil {
    log.Fatalf("Approval failed: %v", err)
}

fmt.Printf("✅ Approval successful: %s\n", result.TxID)
```

### Unlimited Approval

```go
// Approve unlimited amount (common pattern for DEX interactions)
maxAmount := decimal.NewFromString("115792089237316195423570985008687907853269984665640564039457584007913129639935")

_, tx, err := trc20Mgr.Approve(ctx, owner, spender, maxAmount)
if err != nil {
    log.Fatalf("Failed to build unlimited approval: %v", err)
}

// Or use a convenience function if available
_, tx, err = trc20Mgr.ApproveUnlimited(ctx, owner, spender)
```

### Safe Approval Pattern

```go
// Safe approval pattern: set to 0 first, then to desired amount
// This prevents certain attack vectors

func SafeApprove(ctx context.Context, trc20Mgr *trc20.TRC20Manager, owner, spender *types.Address, amount decimal.Decimal) error {
    // First, check current allowance
    currentAllowance, err := trc20Mgr.Allowance(ctx, owner, spender)
    if err != nil {
        return fmt.Errorf("failed to check current allowance: %w", err)
    }

    // If there's an existing allowance and we're not setting to 0, reset first
    if !currentAllowance.IsZero() && !amount.IsZero() {
        fmt.Println("Resetting allowance to 0 first...")
        
        _, tx, err := trc20Mgr.Approve(ctx, owner, spender, decimal.Zero)
        if err != nil {
            return fmt.Errorf("failed to reset allowance: %w", err)
        }

        result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
        if err != nil {
            return fmt.Errorf("failed to broadcast reset: %w", err)
        }

        fmt.Printf("Reset successful: %s\n", result.TxID)
    }

    // Now set the desired allowance
    _, tx, err := trc20Mgr.Approve(ctx, owner, spender, amount)
    if err != nil {
        return fmt.Errorf("failed to set allowance: %w", err)
    }

    result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
    if err != nil {
        return fmt.Errorf("failed to broadcast approval: %w", err)
    }

    fmt.Printf("Approval successful: %s\n", result.TxID)
    return nil
}
```

## 🔄 TransferFrom Operations

```go
// Transfer tokens on behalf of another account (requires prior approval)
owner, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")  // Token owner
spender, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x") // You (authorized spender)
recipient, _ := types.NewAddress("TAuB7aNiJ2Sj5r3xrqoRH8UhZVNYBUHxdf") // Final recipient

// Check allowance first
allowance, err := trc20Mgr.Allowance(ctx, owner, spender)
if err != nil {
    log.Fatalf("Failed to check allowance: %v", err)
}

transferAmount := decimal.NewFromFloat(50.0)

if allowance.LessThan(transferAmount) {
    log.Fatalf("Insufficient allowance: have %s, need %s", allowance, transferAmount)
}

// Perform transferFrom
_, tx, err := trc20Mgr.TransferFrom(ctx, spender, owner, recipient, transferAmount)
if err != nil {
    log.Fatalf("Failed to build transferFrom: %v", err)
}

// Note: Transaction is signed by the spender, not the owner
result, err := cli.SignAndBroadcast(ctx, tx, opts, spenderSigner)
if err != nil {
    log.Fatalf("TransferFrom failed: %v", err)
}

fmt.Printf("✅ TransferFrom successful: %s\n", result.TxID)
```

## 💱 Decimal Conversion Utilities

The TRC20 package uses `shopspring/decimal` for precise arithmetic and provides utilities for converting between human-readable decimals and on-chain integer values.

### Manual Conversion Functions

```go
// Convert human decimal to on-chain integer (wei)
humanAmount := decimal.NewFromFloat(12.34)
decimals := 6 // USDT has 6 decimals

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
    decimals int
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
fmt.Printf("USDT wei: %s\n", usdtWei) // 100500000
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

```go
type MultiTokenManager struct {
    client   *client.Client
    managers map[string]*trc20.TRC20Manager
}

func NewMultiTokenManager(cli *client.Client) *MultiTokenManager {
    return &MultiTokenManager{
        client:   cli,
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

    mgr, err := trc20.NewManager(m.client, addr)
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

```go
type TokenBalance struct {
    Symbol   string
    Address  string
    Balance  decimal.Decimal
    Decimals int
}

func GetPortfolio(ctx context.Context, cli *client.Client, holderAddr *types.Address, tokens []string) ([]TokenBalance, error) {
    var portfolio []TokenBalance

    for _, tokenAddr := range tokens {
        addr, err := types.NewAddress(tokenAddr)
        if err != nil {
            continue // Skip invalid addresses
        }

        mgr, err := trc20.NewManager(cli, addr)
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

// Usage
tokens := []string{
    "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", // USDT
    "TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8", // USDC
    "TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9", // JST
}

holder, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
portfolio, err := GetPortfolio(ctx, cli, holder, tokens)
if err != nil {
    log.Fatal(err)
}

for _, token := range portfolio {
    fmt.Printf("%s: %s\n", token.Symbol, token.Balance)
}
```

## 🚨 Error Handling

### Common Error Types

```go
// The package defines specific error types
var (
    ErrInvalidTokenAddress   = errors.New("invalid token address")
    ErrInvalidAmount        = errors.New("invalid amount")
    ErrInsufficientBalance  = errors.New("insufficient balance")
    ErrInsufficientAllowance = errors.New("insufficient allowance")
)

// Usage with error checking
balance, err := trc20Mgr.BalanceOf(ctx, holder)
if err != nil {
    if errors.Is(err, trc20.ErrInvalidTokenAddress) {
        log.Println("The token contract address is invalid")
    } else {
        log.Printf("Unexpected error: %v", err)
    }
    return
}
```

### Robust Error Handling Pattern

```go
func SafeTransfer(ctx context.Context, mgr *trc20.TRC20Manager, from, to *types.Address, amount decimal.Decimal) error {
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
    _, tx, err := mgr.Transfer(ctx, from, to, amount)
    if err != nil {
        return fmt.Errorf("failed to build transfer transaction: %w", err)
    }

    // At this point, you would sign and broadcast
    fmt.Printf("Transfer ready: %s %s from %s to %s\n", amount, symbol, from, to)
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
    
    // Mock client (implement your mock)
    mockClient := &MockClient{}
    tokenAddr := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
    
    mgr, err := trc20.NewManager(mockClient, tokenAddr)
    require.NoError(t, err)
    
    // Test transfer
    amount := decimal.NewFromFloat(10.5)
    _, tx, err := mgr.Transfer(context.Background(), from, to, amount)
    require.NoError(t, err)
    require.NotNil(t, tx)
}
```

### Integration Testing

```go
// Test against real testnet
func TestRealTRC20Operations(t *testing.T) {
    // Skip if not in integration test mode
    if !*integrationTest {
        t.Skip("Skipping integration test")
    }

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
