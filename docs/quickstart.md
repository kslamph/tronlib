# 🚀 Quick Start Guide

This guide will get you up and running with TronLib in minutes. We'll cover the most common operations: TRX transfers and TRC20 token interactions. This is the first step in learning TronLib.

## 📚 Learning Path

1. **Quick Start Guide** (this document) - Basic usage
2. [Architecture Overview](architecture.md) - Understanding the design
3. [Package Documentation](../README.md#package-references) - Detailed API references
4. [API Reference](API_REFERENCE.md) - Complete function documentation
5. [Examples](../example/) - Real-world implementations

## 📋 Prerequisites

- Go 1.19 or later
- Access to a TRON node (we'll use public endpoints)
- A private key with some TRX for gas fees

## 🛠️ Installation

```bash
go mod init my-tron-app
go get github.com/kslamph/tronlib
```

## 🎯 Your First TRX Transfer

Let's start with the simplest operation: transferring TRX from one address to another.

### Step 1: Basic Setup

Create a new file `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/signer"
    "github.com/kslamph/tronlib/pkg/types"
    "github.com/kslamph/tronlib/pkg/utils"
)

func main() {
    // We'll add code here step by step
}
```

### Step 2: Connect to TRON Network

```go
func main() {
    // Connect to Nile testnet (use testnet for learning!)
    cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer cli.Close()

    fmt.Println("✅ Connected to TRON network")
}
```

### Step 3: Set Up Your Wallet

> ⚠️ **Security Note**: Never hardcode private keys in production! Use environment variables or secure key management.

```go
func main() {
    // ... connection code from Step 2

    // Create signer from private key (get test TRX from Nile faucet)
    privateKey := "your-private-key-here" // Replace with your test key
    signer, err := signer.NewPrivateKeySigner(privateKey)
    if err != nil {
        log.Fatalf("Invalid private key: %v", err)
    }

    from := signer.Address()
    fmt.Printf("Your address: %s\n", from)
}
```

### Step 4: Check Your Balance

```go
func main() {
    // ... previous code

    ctx := context.Background()
    
    // Check balance before transfer
    balance, err := cli.Account().GetBalance(ctx, from)
    if err != nil {
        log.Fatalf("Failed to get balance: %v", err)
    }
    
    // Convert SUN to TRX using utils package
    trxBalance, err := utils.HumanReadableBalance(balance, 6) // 6 decimal places for TRX
    if err != nil {
        log.Printf("Warning: Failed to format balance: %v", err)
        fmt.Printf("Current balance: %d SUN\n", balance)
    } else {
        fmt.Printf("Current balance: %s TRX\n", trxBalance)
    }
    
    if balance < 2_000_000 { // Need at least 2 TRX
        log.Fatal("❌ Insufficient balance. Get test TRX from Nile faucet!")
    }
}
```

### Step 5: Perform the Transfer

```go
func main() {
    // ... previous code

    // Define recipient address
    to, err := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
    if err != nil {
        log.Fatalf("Invalid recipient address: %v", err)
    }

    // Transfer 1 TRX (1,000,000 SUN)
    transferAmount := int64(1_000_000)
    
    // Convert SUN to TRX for display using utils package
    transferAmountTRX, convErr := utils.HumanReadableBalance(transferAmount, 6)
    if convErr != nil {
        log.Printf("Warning: Failed to format transfer amount: %v", convErr)
        fmt.Printf("Transferring %d SUN to %s...\n", transferAmount, to)
    } else {
        fmt.Printf("Transferring %s TRX to %s...\n", transferAmountTRX, to)
    }

    // Build the transaction
    tx, err := cli.Account().TransferTRX(ctx, from, to, transferAmount)
    if err != nil {
        log.Fatalf("Failed to build transaction: %v", err)
    }

    // Configure broadcast options
    opts := client.DefaultBroadcastOptions()
    opts.WaitForReceipt = true                    // Wait for confirmation
    opts.WaitTimeout = 30 * time.Second          // Timeout after 30s
    opts.FeeLimit = 10_000_000                   // Max 10 TRX fee

    // Sign and broadcast
    result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
    if err != nil {
        log.Fatalf("Transaction failed: %v", err)
    }

    // Show results
    fmt.Printf("🎉 Transaction successful!\n")
    fmt.Printf("   Transaction ID: %s\n", result.TxID)
    fmt.Printf("   Energy used: %d\n", result.EnergyUsage)
    fmt.Printf("   Success: %v\n", result.Success)
}
```

### Complete Example

<details>
<summary>Click to see the complete TRX transfer example</summary>

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/signer"
    "github.com/kslamph/tronlib/pkg/types"
    "github.com/kslamph/tronlib/pkg/utils"
)

func main() {
    // Connect to Nile testnet
    cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer cli.Close()

    // Create signer
    privateKey := "your-private-key-here"
    signer, err := signer.NewPrivateKeySigner(privateKey)
    if err != nil {
        log.Fatalf("Invalid private key: %v", err)
    }

    from := signer.Address()
    ctx := context.Background()

    // Check balance
    balance, err := cli.Account().GetBalance(ctx, from)
    if err != nil {
        log.Fatalf("Failed to get balance: %v", err)
    }

    // Convert SUN to TRX using utils package
    trxBalance, err := utils.HumanReadableBalance(balance, 6) // 6 decimal places for TRX
    if err != nil {
        log.Printf("Warning: Failed to format balance: %v", err)
        fmt.Printf("Balance: %d SUN\n", balance)
    } else {
        fmt.Printf("Balance: %s TRX\n", trxBalance)
    }

    // Transfer setup
    to, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
    transferAmount := int64(1_000_000) // 1 TRX
    
    // Convert SUN to TRX for display using utils package
    transferAmountTRX, convErr := utils.HumanReadableBalance(transferAmount, 6)
    if convErr != nil {
        log.Printf("Warning: Failed to format transfer amount: %v", convErr)
        fmt.Printf("Transferring %d SUN to %s...\n", transferAmount, to)
    } else {
        fmt.Printf("Transferring %s TRX to %s...\n", transferAmountTRX, to)
    }

    // Build and send transaction
    tx, err := cli.Account().TransferTRX(ctx, from, to, transferAmount)
    if err != nil {
        log.Fatalf("Failed to build transaction: %v", err)
    }

    opts := client.DefaultBroadcastOptions()
    opts.WaitForReceipt = true
    opts.WaitTimeout = 30 * time.Second

    result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
    if err != nil {
        log.Fatalf("Transaction failed: %v", err)
    }

    fmt.Printf("✅ Success! TxID: %s\n", result.TxID)
}
```
</details>

## 🪙 TRC20 Token Transfer

Now let's learn how to transfer TRC20 tokens. We'll use USDT as an example.

### Step 1: TRC20 Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/shopspring/decimal"
    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/signer"
    "github.com/kslamph/tronlib/pkg/trc20"
    "github.com/kslamph/tronlib/pkg/types"
)

func main() {
    // Connect to network
    cli, err := client.NewClient("grpc://grpc.trongrid.io:50051") // Mainnet for real USDT
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer cli.Close()

    // Set up signer
    signer, err := signer.NewPrivateKeySigner("your-private-key")
    if err != nil {
        log.Fatalf("Invalid private key: %v", err)
    }

    from := signer.Address()
    ctx := context.Background()
}
```

### Step 2: Create TRC20 Manager

```go
func main() {
    // ... previous setup code

    // USDT contract address on mainnet
    usdtAddr, err := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
    if err != nil {
        log.Fatalf("Invalid token address: %v", err)
    }

    // Create TRC20 manager
    trc20Mgr := cli.TRC20(usdtAddr)
    if trc20Mgr == nil {
        log.Fatal("Failed to create TRC20 manager")
    }

    // The manager automatically fetches and caches token metadata
    name, _ := trc20Mgr.Name(ctx)
    symbol, _ := trc20Mgr.Symbol(ctx)
    decimals, _ := trc20Mgr.Decimals(ctx)

    fmt.Printf("Token: %s (%s) with %d decimals\n", name, symbol, decimals)
}
```

### Step 3: Check Token Balance

```go
func main() {
    // ... previous code

    // Check TRC20 balance
    balance, err := trc20Mgr.BalanceOf(ctx, from)
    if err != nil {
        log.Fatalf("Failed to get token balance: %v", err)
    }

    fmt.Printf("USDT Balance: %s\n", balance.String())

    // Check if we have enough tokens
    minAmount := decimal.NewFromFloat(1.0) // Need at least 1 USDT
    if balance.LessThan(minAmount) {
        log.Fatal("❌ Insufficient USDT balance")
    }
}
```

### Step 4: Perform Token Transfer

```go
func main() {
    // ... previous code

    // Define transfer details
    recipient, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
    amount := decimal.NewFromFloat(10.5) // 10.5 USDT

    fmt.Printf("Transferring %s USDT to %s...\n", amount.String(), recipient)

    // Build TRC20 transfer transaction
    // Note: This returns both transaction ID (for immediate use) and transaction object
    _, tx, err := trc20Mgr.Transfer(ctx, from, recipient, amount)
    if err != nil {
        log.Fatalf("Failed to build transfer: %v", err)
    }

    // Configure options for TRC20 (higher fee limit needed)
    opts := client.DefaultBroadcastOptions()
    opts.FeeLimit = 50_000_000                   // 50 TRX max (TRC20 needs more energy)
    opts.WaitForReceipt = true
    opts.WaitTimeout = 30 * time.Second

    // Sign and broadcast
    result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
    if err != nil {
        log.Fatalf("Transaction failed: %v", err)
    }

    fmt.Printf("🎉 TRC20 transfer successful!\n")
    fmt.Printf("   Transaction ID: %s\n", result.TxID)
    fmt.Printf("   Energy used: %d\n", result.EnergyUsage)
}
```

### Complete TRC20 Example

<details>
<summary>Click to see the complete TRC20 transfer example</summary>

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/shopspring/decimal"
    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/signer"
    "github.com/kslamph/tronlib/pkg/trc20"
    "github.com/kslamph/tronlib/pkg/types"
)

func main() {
    // Connect and setup
    cli, err := client.NewClient("grpc://grpc.trongrid.io:50051")
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer cli.Close()

    signer, err := signer.NewPrivateKeySigner("your-private-key")
    if err != nil {
        log.Fatalf("Invalid private key: %v", err)
    }

    from := signer.Address()
    ctx := context.Background()

    // Create TRC20 manager for USDT
    usdtAddr, _ := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
    trc20Mgr, err := trc20.NewManager(cli, usdtAddr)
    if err != nil {
        log.Fatalf("Failed to create TRC20 manager: %v", err)
    }

    // Check balance
    balance, err := trc20Mgr.BalanceOf(ctx, from)
    if err != nil {
        log.Fatalf("Failed to get balance: %v", err)
    }

    fmt.Printf("USDT Balance: %s\n", balance.String())

    // Transfer
    recipient, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
    amount := decimal.NewFromFloat(10.5)

    _, tx, err := trc20Mgr.Transfer(ctx, from, recipient, amount)
    if err != nil {
        log.Fatalf("Failed to build transfer: %v", err)
    }

    opts := client.DefaultBroadcastOptions()
    opts.FeeLimit = 50_000_000
    opts.WaitForReceipt = true

    result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
    if err != nil {
        log.Fatalf("Transaction failed: %v", err)
    }

    fmt.Printf("✅ Success! TxID: %s\n", result.TxID)
}
```
</details>

## 🔍 Transaction Simulation

Before spending real TRX on fees, you can simulate transactions to predict their outcome:

```go
// Build transaction as usual
tx, err := cli.Account().TransferTRX(ctx, from, to, amount)
if err != nil {
    log.Fatal(err)
}

// Simulate first
simResult, err := cli.Simulate(ctx, tx)
if err != nil {
    log.Fatalf("Simulation failed: %v", err)
}

fmt.Printf("Simulation Results:\n")
fmt.Printf("  Would succeed: %v\n", simResult.Success)
fmt.Printf("  Energy needed: %d\n", simResult.EnergyUsage)
fmt.Printf("  Message: %s\n", simResult.Message)

// Only broadcast if simulation succeeds
if simResult.Success {
    result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✅ Transaction sent: %s\n", result.TxID)
} else {
    fmt.Printf("❌ Transaction would fail: %s\n", simResult.Message)
}
```

## 📊 Reading Transaction Events

After a successful transaction, you can decode the events that were emitted:

```go
// Assuming 'result' is from a successful SignAndBroadcast
if len(result.Logs) > 0 {
    fmt.Printf("Transaction emitted %d events:\n", len(result.Logs))
    
    for i, log := range result.Logs {
        // Get contract address that emitted the event
        contractAddr := types.MustNewAddressFromBytes(log.GetAddress())
        
        // Decode the event
        event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
        if err != nil {
            fmt.Printf("  [%d] Failed to decode event: %v\n", i, err)
            continue
        }
        
        fmt.Printf("  [%d] %s emitted %s:\n", i, contractAddr, event.EventName)
        for _, param := range event.Parameters {
            fmt.Printf("      %s: %v\n", param.Name, param.Value)
        }
    }
} else {
    fmt.Println("No events emitted")
}
```

## 🎯 Common Patterns

### 1. Error Handling

```go
// Always check for specific error types
balance, err := cli.Accounts().GetBalance(ctx, address)
if err != nil {
    // Log the error with context
    log.Printf("Failed to get balance for %s: %v", address, err)
    return fmt.Errorf("balance query failed: %w", err)
}
```

### 2. Context with Timeout

```go
// Always use timeouts for network operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
```

### 3. Fee Limit Guidelines

```go
// Recommended fee limits for different operations
var opts = client.DefaultBroadcastOptions()

// TRX transfers: 1-5 TRX usually sufficient
opts.FeeLimit = 5_000_000

// TRC20 transfers: 10-50 TRX depending on contract complexity
opts.FeeLimit = 30_000_000

// Smart contract calls: Varies widely, simulate first
simResult, _ := cli.Simulate(ctx, tx)
opts.FeeLimit = simResult.EnergyUsage * 2 // Add safety margin
```

## 🔗 Next Steps

Now that you can transfer TRX and TRC20 tokens, explore these topics:

1. **[Smart Contracts](smartcontract.md)** - Deploy and interact with custom contracts
2. **[Event Decoding](eventdecoder.md)** - Decode complex transaction events
3. **[Architecture](architecture.md)** - Understand the library's design
4. **[Types](types.md)** - Master address handling and type conversions

## 🆘 Troubleshooting

### Common Issues

**"Insufficient balance" errors:**
- Check both TRX balance (for fees) and token balance
- Use testnet for learning: get free TRX from [Nile Faucet](https://nile.tronscan.org/#/tools/system)

**"Transaction failed" with energy errors:**
- Increase `opts.FeeLimit`
- Consider freezing TRX for energy if making many transactions

**Connection timeouts:**
- Use different node endpoints if one is slow
- Increase `client.WithTimeout()` value

**Address format errors:**
- TRON addresses start with 'T' and are 34 characters long
- Use `types.NewAddress()` for validation

### Getting Help

- Check the [example directory](../example/) for working code
- Review [integration tests](../integration_test/) for patterns
- Open an issue on GitHub for bugs or questions

Happy building on TRON! 🚀
