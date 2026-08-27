# Voting Package

The `voting` package provides witness (Super Representative) management and voting operations on the TRON network. It handles vote casting, reward claims, witness creation, and brokerage configuration. All mutating methods return an unsigned `*api.TransactionExtention`.

## Import

```go
import "github.com/kslamph/tronlib/pkg/voting"
```

## Construction

```go
// Via client gateway
cli, _ := client.NewClient("grpc://grpc.trongrid.io:50051")
defer cli.Close()
mgr := cli.Voting()

// Direct (for testing)
mgr := voting.NewManager(connProvider)
```

## Vote Type

```go
type voting.Vote struct {
    WitnessAddress *types.Address
    VoteCount      int64
}
```

## Operations

### VoteWitnessAccount2

```go
votes := []voting.Vote{
    {WitnessAddress: witness1, VoteCount: 100},
    {WitnessAddress: witness2, VoteCount: 50},
}
txExt, err := mgr.VoteWitnessAccount2(ctx, owner, votes)
if err != nil {
    log.Fatal(err)
}
result, err := cli.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), signer)
```

Casts votes for one or more witnesses. Each vote count must be positive; the votes list must not be empty; all witness addresses must be non-nil.

### WithdrawBalance2

```go
txExt, err := mgr.WithdrawBalance2(ctx, owner)
```

Claims voting rewards accumulated from staking and voting.

### ListWitnesses

```go
witnesses, err := mgr.ListWitnesses(ctx)
// witnesses.Witnesses — list of current Super Representatives
// each witness: Url, VoteCount, TotalProduced, TotalMissed
```

Returns `*api.WitnessList` of all active witnesses.

### GetRewardInfo

```go
reward, err := mgr.GetRewardInfo(ctx, address)
// reward.Num — unclaimed reward amount in SUN
```

Returns `*api.NumberMessage` with the unclaimed reward balance.

### GetBrokerageInfo

```go
brokerage, err := mgr.GetBrokerageInfo(ctx, address)
// brokerage.Num — brokerage percentage (0-100)
```

Returns `*api.NumberMessage` with the witness's brokerage split.

### CreateWitness2

```go
txExt, err := mgr.CreateWitness2(ctx, owner, "https://example.com")
```

Registers the account as a Super Representative candidate.

### UpdateWitness2

```go
txExt, err := mgr.UpdateWitness2(ctx, owner, "https://new-url.com")
```

Updates the witness URL.

### UpdateBrokerage

```go
txExt, err := mgr.UpdateBrokerage(ctx, owner, 20) // 20% brokerage
```

Sets the brokerage percentage (0–100) — the fraction of rewards shared with voters.

## Error Handling

| Sentinel | Meaning |
|---|---|
| `types.ErrInvalidAddress` | nil owner or witness address |
| `types.ErrInvalidParameter` | Empty vote list, vote count ≤ 0, empty URL, brokerage outside 0–100 |

Use `errors.Is` to check:

```go
if errors.Is(err, types.ErrInvalidParameter) {
    // check vote counts and URL values
}
```

## See Also

- [Client Package](client.md) — connection setup, `SignAndBroadcast`
- [Account Package](account.md) — account queries and TRX transfers
- [Network Package](network.md) — block and transaction queries
- [Resources Package](resources.md) — staking for energy/bandwidth
- [Signer Package](signer.md) — transaction signing
