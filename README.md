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

    // Replace with your gRPC node address (mainnet/testnet/full node)
    node := "your.tron.node:port"
    cfg := client.DefaultClientConfig(node)
    cli, err := client.NewClient(cfg)
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

    cli, err := client.NewClient(client.DefaultClientConfig(node))
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

    cli, _ := client.NewClient(client.DefaultClientConfig(node))
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
    _, _ = cli.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), pk)

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

    cli, _ := client.NewClient(client.DefaultClientConfig(node))
    defer cli.Close()

    token, _ := types.NewAddress("Ttokenxxxxxxxxxxxxxxxxxxxxxxxxxxx")
    holder, _ := types.NewAddress("Tholderxxxxxxxxxxxxxxxxxxxxxxxxx")
    recipient, _ := types.NewAddress("Trecipientxxxxxxxxxxxxxxxxxxxxx")
    spender, _ := types.NewAddress("Tspenderxxxxxxxxxxxxxxxxxxxxxxxx")

    erc20, err := trc20.NewManager(cli, token)
    if err != nil { log.Fatal(err) }

    // Reads
    name, _ := erc20.Name(ctx)
    symbol, _ := erc20.Symbol(ctx)
    decimals, _ := erc20.Decimals(ctx)
    bal, _ := erc20.BalanceOf(ctx, holder)
    allowance, _ := erc20.Allowance(ctx, holder, spender)
    _ = name; _ = symbol; _ = bal; _ = allowance

    // Write (build tx then sign & broadcast)
    amount := decimal.NewFromFloat(12.34)
    txid, txExt, err := erc20.Transfer(ctx, holder, recipient, amount)
    if err != nil { log.Fatal(err) }
    _ = txid
    pk, _ := signer.NewPrivateKeySigner("0x<hex-privkey>")
    _, _ = cli.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), pk)
}
```

### Transaction broadcast

`DefaultBroadcastOptions()` controls signing and broadcast behavior.
- `FeeLimit` (SUN)
- `WaitForReceipt` (bool)
- `WaitTimeout` (seconds, int64)
- `PollInterval` (`time.Duration`)
- `PermissionID` (int32)

```go
opts := client.DefaultBroadcastOptions()
opts.FeeLimit = 100_000_000
res, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
if err != nil { /* network/broadcast error */ }
if !res.Success { /* TRON return code/message in res */ }
// When WaitForReceipt=true and receipt arrives:
_ = res.ContractReceipt
_ = res.ContractResult
```

### Simulation (constant execution)

Simulate execution/energy of a transaction before sending:

```go
ext, err := cli.Simulate(ctx, txExt /* or *core.Transaction */)
if err != nil { /* validation or RPC error */ }
energyUsed := ext.GetEnergyUsed()
```

### Testing (contributors)

- Tests use hermetic bufconn gRPC servers; keep test contexts short and deterministic.
- Most managers are thin over the core client; favor unit tests at the manager boundary with fakes.

