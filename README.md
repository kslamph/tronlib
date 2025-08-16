# tronlib 🚀

[![GoDoc](https://godoc.org/github.com/kslamph/tronlib?status.svg)](https://godoc.org/github.com/kslamph/tronlib)
[![Go Report Card](https://goreportcard.com/badge/github.com/kslamph/tronlib)](https://goreportcard.com/report/github.com/kslamph/tronlib)

Go library for interacting with the TRON blockchain. It provides:

- A core gRPC `client` with connection pooling and timeouts
- High-level managers: `account`, `resources`, `network`, `smartcontract`, `trc10`, `trc20`

## 📦 Installation

```bash
go get github.com/kslamph/tronlib
```

## 🚀 Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/kslamph/tronlib/pkg/client"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Replace with your TRON node URL (grpc://host:port or grpcs://host:port)
    node := "grpc://your.tron.node:port"
    cli, err := client.NewClient(node)
    if err != nil {
        log.Fatalf("new client: %v", err)
    }
    defer cli.Close()

    // Use cli with high-level managers below
}
```

### 🕐 Context Usage

- Pass `context.Context` to every call. Prefer short per-operation timeouts.
- If your context has no deadline, the client applies its own default timeout.

## 📚 Documentation

- [Architecture Overview](docs/architecture.md) - High-level view of package structure and data flow
- [GoDoc Summary](docs/godoc_summary.md) - Key entry points and examples from package documentation
- [Event Decoding Guide](docs/event_decoding.md) - How to decode logs from receipts or simulations

## 🔄 Workflow Diagram

```
┌──────────────────────────────┐
│        High-level APIs       │
│ account / resources / network│
│ smartcontract / trc10 / trc20│
└───────────────┬──────────────┘
                │ build txs / make queries
                ▼
        ┌───────────────┐
        │ client.Client │
        │ RPC / simulate│
        │ sign+broadcast│
        └───────┬───────┘
                │
                ▼
        ┌───────────────┐
        │   TRON Node   │
        │ (gRPC endpoint)│
        └───────────────┘
```

- Transaction read: managers → client → node → data
- Contract read (constant): smartcontract/trc20 → client (constant trigger) → node → ABI-decode return values
- Transaction write: managers build tx → client.SignAndBroadcast (uses signer) → node → receipt → eventdecoder.DecodeLogs
- Simulation: managers or client.Simulate → node → inspect result → optionally decode logs with eventdecoder

### 🔧 Key Components

- **types.Address** 📍 - Unified address representation supporting multiple formats
- **signer** 🔐 - Private key and HD wallet management
- **broadcaster** 📡 - Broadcasting with receipt waiting (in `client`)
- **ABI Processor** 🧬 - Encoding/decoding of ABI data
- **Event Decoder** 📊 - Log decoding with built-in events
- **High-level Managers** 🛠️ - Simplified interfaces for common operations

### 🎯 Usage Patterns

**High-level usage** (recommended for most applications):

```go
// Using account manager for TRX transfer
am := account.NewManager(cli)
txExt, err := am.TransferTRX(ctx, from, to, 1_000_000, nil)
```

**Low-level usage** (for advanced customization):

```go
// Direct client usage for custom transactions
tx := &core.Transaction{ /* ... */ }
signedTx, err := cli.SignTransaction(ctx, tx, signer)
```

## 💡 Examples

### 👤 Accounts and Resources

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/kslamph/tronlib/pkg/account"
    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/resources"
    "github.com/kslamph/tronlib/pkg/signer"
    "github.com/kslamph/tronlib/pkg/types"
)

func exampleAccounts(node string) {
    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()

    cli, err := client.NewClient(node)
    if err != nil { log.Fatal(err) }
    defer cli.Close()

    am := account.NewManager(cli)
    rm := resources.NewManager(cli)

    from, _ := types.NewAddress("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1")
    to, _ := types.NewAddress("Tyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy2")

    // Get account balance
    bal, err := am.GetBalance(ctx, from)
    if err != nil { log.Fatal(err) }
    _ = bal

    // Build a TRX transfer (unsigned)
    txExt, err := am.TransferTRX(ctx, from, to, 1_000_000, nil) // 1 TRX = 1_000_000 SUN
    if err != nil { log.Fatal(err) }

    // Sign & broadcast
    pk, _ := signer.NewPrivateKeySigner("0x<hex-privkey>")
    res, err := cli.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), pk)
    if err != nil { log.Fatal(err) }
    if !res.Success { log.Printf("broadcast failed: %s", res.Message) }

    // Freeze/Unfreeze resources
    _, _ = rm.FreezeBalanceV2(ctx, from, 1_000_000, resources.ResourceTypeEnergy)
    _, _ = rm.UnfreezeBalanceV2(ctx, from, 1_000_000, resources.ResourceTypeEnergy)
}
```

### 📜 Smart Contracts

Deploy and interact using `smartcontract.Manager` or a typed `smartcontract.Contract`.

```go
package main

import (
    "context"
    "encoding/hex"
    "log"
    "time"

    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/signer"
    "github.com/kslamph/tronlib/pkg/smartcontract"
    "github.com/kslamph/tronlib/pkg/types"
)

func exampleSmartContract(node string) {
    ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
    defer cancel()

    cli, _ := client.NewClient(node)
    defer cli.Close()

    mgr := smartcontract.NewManager(cli)
    owner, _ := types.NewAddress("Townerxxxxxxxxxxxxxxxxxxxxxxxxxx")

    abiJSON := `{"entrys":[{"type":"constructor","inputs":[{"name":"_owner","type":"address"}]},{"type":"function","name":"setValue","inputs":[{"name":"v","type":"uint256"}]},{"type":"function","name":"getValue","inputs":[],"outputs":[{"name":"","type":"uint256"}],"constant":true}]}`
    bytecode, _ := hex.DecodeString("60806040...deadbeef")

    // Deploy
    txExt, err := mgr.DeployContract(ctx, owner, "MyContract", abiJSON, bytecode, 0, 100, 30000, owner.Bytes())
    if err != nil { log.Fatal(err) }
    pk, _ := signer.NewPrivateKeySigner("0x<hex-privkey>")
    _, _ = cli.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), pk)

    // Interact via Contract
    contractAddr, _ := types.NewAddress("Tcontractxxxxxxxxxxxxxxxxxxxxxxxx")
    c, err := smartcontract.NewContract(cli, contractAddr, abiJSON)
    if err != nil { log.Fatal(err) }

    // State-changing call (build tx only)
    txExt, err = c.TriggerSmartContract(ctx, owner, 0, "setValue", uint64(42))
    if err != nil { log.Fatal(err) }

    // Sign & broadcast and wait for execution result
    opts := client.DefaultBroadcastOptions()
    opts.WaitForReceipt = true
    res, err := cli.SignAndBroadcast(ctx, txExt, opts, pk)
    if err != nil { log.Fatalf("broadcast error: %v", err) }
    log.Printf("txid=%s", res.TxID) // always available
}
```

### 💰 TRC20

Read and write helpers plus exact unit conversion using `shopspring/decimal`.

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/shopspring/decimal"

    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/signer"
    "github.com/kslamph/tronlib/pkg/trc20"
    "github.com/kslamph/tronlib/pkg/types"
)

func exampleTRC20(node string) {
    ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
    defer cancel()

    cli, _ := client.NewClient(node)
    defer cli.Close()

    token, _ := types.NewAddress("Ttokenxxxxxxxxxxxxxxxxxxxxxxxxxxx")
    holder, _ := types.NewAddress("Tholderxxxxxxxxxxxxxxxxxxxxxxxxx")
    recipient, _ := types.NewAddress("Trecipientxxxxxxxxxxxxxxxxxxxxx")

    trc20Contract, err := trc20.NewManager(cli, token)
    if err != nil { log.Fatal(err) }

    // Reads
    _, _ = trc20Contract.Name(ctx)
    _, _ = trc20Contract.Symbol(ctx)
    _, _ = trc20Contract.Decimals(ctx)
    _, _ = trc20Contract.BalanceOf(ctx, holder)

    // Write (build tx then sign & broadcast)
    amount := decimal.NewFromFloat(12.34)
    txid, txExt, err := trc20Contract.Transfer(ctx, holder, recipient, amount)
    if err != nil { log.Fatal(err) }
    log.Printf("built txid=%s", txid)
    pk, _ := signer.NewPrivateKeySigner("0x<hex-privkey>")
    opts := client.DefaultBroadcastOptions()
    opts.WaitForReceipt = true
    _, _ = cli.SignAndBroadcast(ctx, txExt, opts, pk)
}
```

## 📊 Decoding Events

See [Event Decoding Guide](docs/event_decoding.md) for details and examples.

## 🤝 Contributing

Contributions are welcome! Please read our contributing guidelines before submitting pull requests.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
