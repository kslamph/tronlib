# 🔗 TronLib - Go SDK for TRON Blockchain

[![Go Reference](https://pkg.go.dev/badge/github.com/kslamph/tronlib.svg)](https://pkg.go.dev/github.com/kslamph/tronlib)
[![Go Report Card](https://goreportcard.com/badge/github.com/kslamph/tronlib)](https://goreportcard.com/report/github.com/kslamph/tronlib)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![codecov](https://codecov.io/gh/kslamph/tronlib/graph/badge.svg?token=QIN77Y7S2T)](https://codecov.io/gh/kslamph/tronlib)

A comprehensive, production-ready Go SDK for interacting with the TRON blockchain. TronLib provides high-level abstractions for common operations while maintaining flexibility for advanced use cases.

## ✨ Features

- 🚀 **Simple & Intuitive** - High-level APIs that make blockchain interaction straightforward
- 🔐 **Secure** - Built-in support for private keys and HD wallets
- 💰 **TRC20 Ready** - First-class support for TRC20 tokens with decimal conversion
- 📦 **Smart Contracts** - Deploy and interact with smart contracts effortlessly
- 🎯 **Event Decoding** - Decode transaction logs with built-in TRC20 event support
- ⚡ **Performance** - Connection pooling and efficient gRPC communication
- 🔍 **Simulation** - Test transactions before broadcasting to the network
- 📊 **Resource Management** - Handle bandwidth and energy efficiently
- 🛡️ **Shielded TRC-20** - Low-level wrappers plus a documented walkthrough of mint / scan / transfer / burn ([docs](docs/shielded.md))

## 🏁 Quick Start

### Installation

```bash
go get github.com/kslamph/tronlib
```

### Simple TRX Transfer

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/signer"
    "github.com/kslamph/tronlib/pkg/types"
)

func main() {
    // Connect to TRON node
    cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
    if err != nil {
        log.Fatal(err)
    }
    defer cli.Close()

    // Create signer from private key
    signer, err := signer.NewPrivateKeySigner("your-private-key-hex")
    if err != nil {
        log.Fatal(err)
    }

    // Define addresses
    from := signer.Address()
    to, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

    // Transfer 1 TRX (1,000,000 SUN)
    tx, err := cli.Account().TransferTRX(context.Background(), from, to, 1_000_000)
    if err != nil {
        log.Fatal(err)
    }

    // Sign and broadcast
    result, err := cli.SignAndBroadcast(context.Background(), tx, client.DefaultBroadcastOptions(), signer)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Transaction ID: %s\n", result.TxID)
    fmt.Printf("Success: %v\n", result.Success)
}
```

### TRC20 Token Transfer

```go
package main

import (
    "context"
    "log"

    "github.com/shopspring/decimal"
    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/signer"
    "github.com/kslamph/tronlib/pkg/trc20"
    "github.com/kslamph/tronlib/pkg/types"
)

func main() {
    cli, _ := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
    defer cli.Close()

    signer, _ := signer.NewPrivateKeySigner("your-private-key-hex")
    
    // USDT contract address on mainnet
    token, _ := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
    to, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

    // Create TRC20 manager
    trc20Mgr, err := cli.TRC20Manager(token)
    if err != nil {
        log.Fatal(err)
    }

    // Transfer 10 USDT
    amount := decimal.NewFromInt(10)
    _, tx, err := trc20Mgr.Transfer(context.Background(), signer.Address(), to, amount)
    if err != nil {
        log.Fatal(err)
    }

    // Sign and broadcast
    result, err := cli.SignAndBroadcast(context.Background(), tx, client.DefaultBroadcastOptions(), signer)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("TRC20 transfer completed: %s", result.TxID)
}
```

## 📚 Documentation

To get the most out of TronLib, follow this learning path:

### 🚀 Getting Started
- **[Quick Start Guide](docs/quickstart.md)** - Step-by-step guide to basic operations

### 🏗️ Core Concepts
- **[Architecture Overview](docs/architecture.md)** - Understanding the library structure and design patterns

### 📦 Package References
Detailed documentation for each package:
- **[Types](docs/types.md)** - Address handling and fundamental types
- **[TRC20](docs/trc20.md)** - TRC20 token operations and decimal handling
- **[Smart Contracts](docs/smartcontract.md)** - Contract deployment and interaction
- **[Event Decoder](docs/eventdecoder.md)** - Transaction log decoding
- **[Utils](docs/utils.md)** - ABI encoding/decoding utilities
- **[Shielded TRC-20](docs/shielded.md)** - Sapling key hierarchy, note scanning, mint / transfer / burn, and the trust model they require

### 💡 Advanced Topics
- **[API Reference](docs/API_REFERENCE.md)** - Complete API documentation
- **[Shielded TRC-20 Walkthrough](example/shielded)** - Runnable five-mode tour of the shielded flow
- **[Complete Examples](example/)** - Real-world usage examples
- **[Integration Tests](integration_test/)** - Comprehensive test suite

## 🏗️ Project Structure

```
tronlib/
├── 📁 pkg/                    # Core library packages
│   ├── 📁 client/            # gRPC client and connection management
│   ├── 📁 types/             # Fundamental types (Address, constants)
│   ├── 📁 signer/            # Private key and HD wallet management
│   ├── 📁 account/           # Account operations (balance, TRX transfers)
│   ├── 📁 trc20/             # TRC20 token operations
│   ├── 📁 smartcontract/     # Smart contract deployment and interaction
│   ├── 📁 eventdecoder/      # Event log decoding
│   ├── 📁 utils/             # ABI encoding/decoding utilities
│   ├── 📁 resources/         # Resource management (bandwidth, energy)
│   ├── 📁 voting/            # Voting operations
│   └── 📁 network/           # Network operations
├── 📁 example/               # Usage examples
├── 📁 cmd/                   # Command-line tools
├── 📁 integration_test/      # Integration tests
└── 📁 docs/                  # Documentation
```

## 🚀 Advanced Usage

### Signing Operations

TronLib provides flexible options for signing transactions and messages.

#### Manual Transaction Signing and Broadcasting

If you prefer to sign transactions manually before broadcasting, or need to integrate with external signers (e.g., hardware wallets, cloud KMS), you can use the `signer.SignTx` function.

```go
// Example for manual signing and broadcasting:
// 1. Create the transaction
// tx, err := cli.Account().TransferTRX(...)
// if err != nil {
//     log.Fatal(err)
// }
//
// 2. Sign the transaction using SignTx
// err = signer.SignTx(yourSigner, tx) // 'yourSigner' is an instance of signer.Signer
// if err != nil {
//     log.Fatal(err)
// }
//
// 3. Broadcast the signed transaction
// result, err := cli.Broadcast(context.Background(), tx)
// if err != nil {
//     log.Fatal(err)
// }
// fmt.Printf("Transaction ID: %s\n", result.TxID)
```

#### Message Signing (TIP-191 v2)

To sign arbitrary messages following the TRON Improvement Proposal 191 (TIP-191) v2 format, use the standalone `signer.SignMessageV2` function.

```go
// Sign an arbitrary message using the standalone SignMessageV2 function
// privateKey := "your-private-key-hex"
// pkSigner, err := signer.NewPrivateKeySigner(privateKey)
// if err != nil {
//     log.Fatal(err)
// }
//
// message := "Hello TronLib!"
// signature, err := signer.SignMessageV2(pkSigner, message)
// if err != nil {
//     log.Fatal(err)
// }
// fmt.Printf("Signed Message: %s\n", signature)
```

### Transaction Simulation

Test transactions before broadcasting:

```go
// Simulate before sending
simResult, err := cli.Simulate(context.Background(), tx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Energy needed: %d\n", simResult.EnergyUsage)
fmt.Printf("Would succeed: %v\n", simResult.Success)

// Only broadcast if simulation succeeds
if simResult.Success {
    result, err := cli.SignAndBroadcast(context.Background(), tx, opts, signer)
    // ...
}
```

### Smart Contract Interaction

```go
// Create contract instance
contract, err := smartcontract.NewInstance(cli, contractAddr, abiJSON)
if err != nil {
    log.Fatal(err)
}

// Call contract method
tx, err := contract.Invoke(ctx, signer.Address(), 0, "setValue", uint64(42))
if err != nil {
    log.Fatal(err)
}

// Sign and broadcast
result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
```

### Event Decoding

```go
// Decode transaction logs
for _, log := range result.Logs {
    event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
    if err != nil {
        continue
    }
    
    fmt.Printf("Event: %s\n", event.EventName)
    for _, param := range event.Parameters {
        fmt.Printf("  %s: %v\n", param.Name, param.Value)
    }
}
```

## 🔧 Configuration

### Client Options

```go
cli, err := client.NewClient("grpc://grpc.trongrid.io:50051",
    client.WithTimeout(30*time.Second),     // Default timeout
    client.WithPool(5, 10),                 // Connection pool: 5 initial, 10 max
)
```

### Broadcast Options

```go
opts := client.DefaultBroadcastOptions()
opts.FeeLimit = 100_000_000                // Set fee limit in SUN
opts.WaitForReceipt = true                 // Wait for transaction receipt
opts.WaitTimeout = 20 * time.Second       // Timeout for receipt
opts.PollInterval = 500 * time.Millisecond // Polling interval
```

## 🌐 Network Support

- **Mainnet**: `grpc://grpc.trongrid.io:50051`
- **Nile Testnet**: `grpc://grpc.nile.trongrid.io:50051`
- **Local Node**: `grpc://127.0.0.1:50051`

> 💡 **Tip**: Use testnet for development and testing. Get test TRX from the [Nile Testnet Faucet](https://nile.tronscan.org/#/tools/system).

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch
3. Add tests for your changes
4. Ensure all tests pass
5. Submit a pull request

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built on the foundation of TRON's gRPC API
- Uses Google Protocol Buffers for efficient communication

---

**Made with ❤️ for the TRON community**

For questions, issues, or feature requests, please open an issue on GitHub.
