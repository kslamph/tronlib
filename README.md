## tronlib

Go library for interacting with the TRON blockchain. It exposes:
- A core gRPC `client` with connection pooling and timeouts
- High-level managers: `account`, `resources`, `network`, `smartcontract`, `trc10`, `trc20`

### Install

```bash
go get github.com/kslamph/tronlib
```

### Quickstart

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

Notes on context:
- Pass `context.Context` to every call. Prefer short per-operation timeouts.
- If your context has no deadline, the client applies its own default timeout.

### Accounts and Resources

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

### Smart Contracts

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
    if !res.Success {
        log.Printf("transaction failed: code=%v msg=%s", res.Code, res.Message)
    } else {
        log.Printf("transaction succeeded: energyUsed=%d netUsage=%d", res.EnergyUsage, res.NetUsage)
    }

    // Constant (read-only) call
    out, err := c.TriggerConstantContract(ctx, owner, "getValue")
    if err != nil { log.Fatal(err) }
    _ = out // decoded single return, e.g., *big.Int
}
```

### TRC20

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
    spender, _ := types.NewAddress("Tspenderxxxxxxxxxxxxxxxxxxxxxxxx")

    trc20Contract, err := trc20.NewManager(cli, token)
    if err != nil { log.Fatal(err) }

    // Reads
    name, _ := trc20Contract.Name(ctx)
    symbol, _ := trc20Contract.Symbol(ctx)
    decimals, _ := trc20Contract.Decimals(ctx)
    bal, _ := trc20Contract.BalanceOf(ctx, holder)
    allowance, _ := trc20Contract.Allowance(ctx, holder, spender)
    _ = name; _ = symbol; _ = bal; _ = allowance

    // Write (build tx then sign & broadcast)
    amount := decimal.NewFromFloat(12.34)
    txid, txExt, err := trc20Contract.Transfer(ctx, holder, recipient, amount)
    if err != nil { log.Fatal(err) }
    log.Printf("built txid=%s", txid)
    pk, _ := signer.NewPrivateKeySigner("0x<hex-privkey>")
    opts := client.DefaultBroadcastOptions()
    opts.WaitForReceipt = true
    res, err := cli.SignAndBroadcast(ctx, txExt, opts, pk)
    if err != nil { log.Fatalf("broadcast error: %v", err) }
    log.Printf("txid=%s", res.TxID) // equals txid above
    if !res.Success {
        log.Printf("transfer failed: %s", res.Message)
    } else {
        log.Printf("transfer ok: energyUsed=%d", res.EnergyUsage)
    }
}
```

### Transaction broadcast

`DefaultBroadcastOptions()` controls signing and broadcast behavior.
- `FeeLimit` (SUN)
- `PermissionID` (int32)
- `WaitForReceipt` (bool)
- `WaitTimeout` (`time.Duration`)
- `PollInterval` (`time.Duration`)

Key points:
- **`res.TxID` is always set** (even if you don't wait for receipt).
- **If `WaitForReceipt=false`**, `res.Success` only means the node accepted the transaction for processing (not that it executed successfully on-chain).
- **If `WaitForReceipt=true`**, and a receipt arrives in time, `res.Success` reflects the final on-chain execution result. Resource usage fields are populated.

```go
opts := client.DefaultBroadcastOptions()
opts.FeeLimit = 100_000_000
opts.WaitForReceipt = true // get execution result
res, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
if err != nil {
    // network or broadcast error
    log.Fatal(err)
}
log.Printf("txid=%s", res.TxID)
if !res.Success {
    log.Printf("failed: code=%v msg=%s", res.Code, res.Message)
} else {
    // Available when receipt is fetched
    log.Printf("ok: energyUsed=%d netUsage=%d", res.EnergyUsage, res.NetUsage)
    // If the contract returns data, it is in ConstantReturn
    _ = res.ConstantReturn
    _ = res.Logs
}
```

### Simulation (constant execution)

Predict execution result and estimate energy before sending any transaction. You can pass either `*api.TransactionExtention` or `*core.Transaction`.

```go
sim, err := cli.Simulate(ctx, txExt /* or *core.Transaction */)
if err != nil {
    log.Fatal(err)
}

// Would the execution succeed?
if !sim.Success {
    log.Printf("would fail: code=%v msg=%s", sim.Code, sim.Message)
}

// Energy estimation is generally reliable
log.Printf("estimated energyUsed=%d", sim.EnergyUsage)

// If the method returns values, they are in ConstantReturn
_ = sim.ConstantReturn
```

Notes:
- Simulation does not require signatures. **Bandwidth (`netUsage`) depends on signatures and payload**; without full signatures, any bandwidth estimation is incomplete/inaccurate.
- For accurate bandwidth, sign the transaction as it will be sent, then broadcast with `WaitForReceipt=true` to observe actual `NetUsage` in the receipt.

### Testing (contributors)

- Tests use hermetic bufconn gRPC servers; keep test contexts short and deterministic.
- Most managers are thin over the core client; favor unit tests at the manager boundary with fakes.

