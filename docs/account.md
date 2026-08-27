# Account Package

The `account` package provides high-level operations for querying TRON accounts and creating TRX transfer transactions. It wraps the low-level gRPC API with input validation and clear error returns, and is typically accessed through the client gateway (`cli.Account()`).

## Import

```go
import "github.com/kslamph/tronlib/pkg/account"
```

## Construction

```go
// Via client gateway (preferred)
cli, _ := client.NewClient("grpc://grpc.trongrid.io:50051")
defer cli.Close()
mgr := cli.Account()

// Direct (for testing with a custom ConnProvider)
mgr := account.NewManager(connProvider)
```

## Operations

### GetAccount

```go
acct, err := mgr.GetAccount(ctx, addr)
// acct.GetBalance() — TRX balance in SUN
// acct.GetAccountResource() — energy/bandwidth info
// acct.GetFrozenV2() — frozen balance list
```

Returns `*core.Account` with balance, resources, frozen amounts, and asset balances.

### GetBalance

```go
balanceSUN, err := mgr.GetBalance(ctx, addr)
// 1 TRX = 1_000_000 SUN
// Use utils.HumanReadableBalance(balanceSUN, 6) for TRX string
```

Convenience wrapper around `GetAccount` that returns just the balance in SUN.

### GetAccountResource

```go
res, err := mgr.GetAccountResource(ctx, addr)
// res.GetEnergyLimit(), res.GetEnergyUsed() — energy
// res.GetNetUsed(), res.GetFreeNetUsed() — bandwidth
```

Returns `*api.AccountResourceMessage` with energy and bandwidth totals/usage.

### GetAccountNet

```go
net, err := mgr.GetAccountNet(ctx, addr)
// net.GetNetUsed(), net.GetNetLimit() — total network usage
// net.GetFreeNetUsed(), net.GetFreeNetLimit() — free bandwidth
```

Returns `*api.AccountNetMessage` with bandwidth details.

### TransferTRX

```go
txExt, err := mgr.TransferTRX(ctx, from, to, 1_000_000) // 1 TRX in SUN
if err != nil {
    // types.ErrInvalidAddress, types.ErrInvalidAmount
    log.Fatal(err)
}
// Sign and broadcast:
result, err := cli.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), signer)
```

Creates an unsigned `*api.TransactionExtention`. Amount must be positive and in SUN.
Addresses must differ; both must be non-nil `*types.Address`.

## Error Handling

All errors wrap sentinel values from `pkg/types`:

| Sentinel | Meaning |
|---|---|
| `types.ErrInvalidAddress` | nil or empty address |
| `types.ErrInvalidAmount` | Amount ≤ 0 in SUN |
| `types.ErrInvalidParameter` | from == to (same address) |
| `types.ErrNetworkError` | gRPC call failed |

Use `errors.Is` to check:

```go
if errors.Is(err, types.ErrInvalidAmount) {
    // amount must be positive
}
```

## See Also

- [TRC20 Package](trc20.md) — ERC-20-style token transfers
- [TRC10 Package](trc10.md) — native TRON asset transfers
- [Resources Package](resources.md) — freeze/unfreeze/delegate
- [Signer Package](signer.md) — transaction signing
- [Client Package](client.md) — connection management and broadcasting
