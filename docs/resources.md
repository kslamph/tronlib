# Resources Package

The `resources` package manages TRX staking operations (freeze, unfreeze, delegate, undelegate) for bandwidth and energy. All mutating methods return an unsigned `*api.TransactionExtention` that must be signed and broadcast separately.

## Import

```go
import "github.com/kslamph/tronlib/pkg/resources"
```

## Construction

```go
// Via client gateway
cli, _ := client.NewClient("grpc://grpc.trongrid.io:50051")
defer cli.Close()
mgr := cli.Resources()

// Direct (for testing)
mgr := resources.NewManager(connProvider)
```

## Resource Types

```go
const (
    resources.ResourceTypeBandwidth = 0
    resources.ResourceTypeEnergy    = 1
)
```

## Operations

### FreezeBalanceV2

```go
txExt, err := mgr.FreezeBalanceV2(ctx, owner, 100_000_000, resources.ResourceTypeEnergy)
// 100 TRX → energy
if err != nil {
    log.Fatal(err)
}
result, err := cli.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), signer)
```

Freezes TRX to obtain energy or bandwidth.

### UnfreezeBalanceV2

```go
txExt, err := mgr.UnfreezeBalanceV2(ctx, owner, 50_000_000, resources.ResourceTypeBandwidth)
```

Unfreezes previously frozen TRX. The unfrozen amount becomes withdrawable after the freeze period expires.

### DelegateResource

```go
txExt, err := mgr.DelegateResource(ctx, owner, receiver, 80_000_000, resources.ResourceTypeEnergy, true)
// lock=true prevents the receiver from undelegating early
```

Delegates staked energy or bandwidth to another address.

### UnDelegateResource

```go
txExt, err := mgr.UnDelegateResource(ctx, owner, receiver, 40_000_000, resources.ResourceTypeEnergy)
```

Removes a delegation to the specified receiver.

### WithdrawExpireUnfreeze

```go
txExt, err := mgr.WithdrawExpireUnfreeze(ctx, owner)
```

Withdraws TRX that finished the unfreeze cooling-off period.

### CancelAllUnfreezeV2

```go
txExt, err := mgr.CancelAllUnfreezeV2(ctx, owner)
```

Cancels all pending unfreeze requests (the TRX stays frozen).

### GetDelegatedResourceV2

```go
del, err := mgr.GetDelegatedResourceV2(ctx, fromAddr, toAddr)
// del.DelegatedResourceV2 — list of delegated amounts per resource type
```

Returns `*api.DelegatedResourceList` showing what `from` delegated to `to`.

### GetCanDelegatedMaxSize

```go
resp, err := mgr.GetCanDelegatedMaxSize(ctx, owner, delegateType)
// delegateType: 0 = bandwidth, 1 = energy
// resp.MaxSize — maximum delegatable amount
```

Returns the maximum amount of the given resource type you can still delegate.

### GetAvailableUnfreezeCount

```go
resp, err := mgr.GetAvailableUnfreezeCount(ctx, owner)
```

Returns `*api.GetAvailableUnfreezeCountResponseMessage` — the number of unfreeze operations you can still initiate this cycle.

## Error Handling

| Sentinel | Meaning |
|---|---|
| `types.ErrInvalidAddress` | nil owner, receiver, or from/to address |
| `types.ErrInvalidAmount` | Amount ≤ 0 |
| `types.ErrInvalidParameter` | owner == receiver (same address for delegate) |

## See Also

- [Client Package](client.md) — connection setup, `SignAndBroadcast`
- [Account Package](account.md) — balance queries
- [Signer Package](signer.md) — transaction signing
- [Network Package](network.md) — block/transaction queries
