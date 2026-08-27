# Client Package

The `client` package manages a gRPC connection pool to a single TRON node and provides the gateway to all domain managers. It handles timeouts, reconnection, and offers `Simulate` (dry-run) and `SignAndBroadcast` (sign + send + optionally wait for receipt).

## Import

```go
import "github.com/kslamph/tronlib/pkg/client"
```

## Construction

```go
cli, err := client.NewClient(
    "grpc://grpc.trongrid.io:50051",  // or grpcs:// for TLS
    client.WithTimeout(30*time.Second),
    client.WithPool(2, 10),            // init, max connections
)
if err != nil {
    log.Fatal(err)
}
defer cli.Close()
```

`endpoint` must include a scheme: `grpc://` or `grpcs://`, followed by `host:port`.

### Options

| Option | Default | Purpose |
|---|---|---|
| `WithTimeout(d)` | 30s | Default RPC timeout when context has no deadline |
| `WithPool(init, max)` | 1, 5 | Connection pool size bounds |

## Gateway Methods

Each method returns a domain-specific manager:

```go
mgr := cli.Account()           // *account.AccountManager
mgr := cli.Network()           // *network.NetworkManager
mgr := cli.Resources()         // *resources.ResourcesManager
mgr := cli.TRC10()             // *trc10.TRC10Manager
mgr := cli.Voting()            // *voting.VotingManager
mgr := cli.SmartContract()     // *smartcontract.Manager
inst, err := cli.ContractInstance(addr, abi)
mgr, err := cli.TRC20Manager(addr)  // preferred
mgr := cli.TRC20(addr)              // deprecated — returns nil on error
```

## Simulate

Dry-run a transaction without sending it:

```go
result, err := cli.Simulate(ctx, txExt) // accepts *api.TransactionExtention or *core.Transaction
if err != nil {
    // handle error
}
if !result.Success {
    fmt.Println("transaction would fail:", result.Message)
}
fmt.Printf("Energy: %d\n", result.EnergyUsage)
```

## SignAndBroadcast

Sign, broadcast, and optionally wait for receipt:

```go
result, err := cli.SignAndBroadcast(
    ctx,
    txExt,                         // *api.TransactionExtention or *core.Transaction
    client.DefaultBroadcastOptions(),
    signer,                        // one or more signer.Signer
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("TxID: %s  Success: %v  Energy: %d\n", result.TxID, result.Success, result.EnergyUsage)
```

`DefaultBroadcastOptions()` returns: FeeLimit 150M SUN, WaitForReceipt true, WaitTimeout 15s, PollInterval 3s.

### BroadcastOptions

```go
type BroadcastOptions struct {
    FeeLimit       int64
    PermissionID   int32
    WaitForReceipt bool
    WaitTimeout    time.Duration
    PollInterval   time.Duration
}
```

## Client Lifecycle

| Method | Purpose |
|---|---|
| `GetConnection(ctx)` | Borrow a connection from the pool |
| `ReturnConnection(conn)` | Return a connection to the pool |
| `Close()` | Close all pool connections (idempotent) |
| `IsConnected()` | Returns false after `Close()` |
| `GetNodeAddress()` | Returns the configured endpoint string |
| `GetTimeout()` | Returns the configured timeout |

## Error Handling

| Sentinel | Meaning |
|---|---|
| `ErrConnectionFailed` | Pool is nil or unavailable |
| `ErrClientClosed` | Client has been closed |
| `ErrContextCancelled` | Context was already done |

## See Also

- [Account Package](account.md) — account queries and TRX transfers
- [Signer Package](signer.md) — `PrivateKeySigner`, `HDWalletSigner`
- [TRC20 Package](trc20.md) — token operations
- [Smart Contract Package](smartcontract.md) — ABI loading and contract calls
