# TRC10 Package

The `trc10` package manages native TRON assets (TRC10 tokens). TRC10 tokens are built into the protocol — no smart contract required — and are identified by name or ID. All mutating methods return an unsigned `*api.TransactionExtention`.

## Import

```go
import "github.com/kslamph/tronlib/pkg/trc10"
```

## Construction

```go
// Via client gateway
cli, _ := client.NewClient("grpc://grpc.trongrid.io:50051")
defer cli.Close()
mgr := cli.TRC10()

// Direct (for testing)
mgr := trc10.NewManager(connProvider)
```

## Operations

### TransferAsset2

```go
txExt, err := mgr.TransferAsset2(ctx, from, to, "1000001", 500)
if err != nil {
    log.Fatal(err)
}
result, err := cli.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), signer)
```

Transfers a TRC10 asset. `assetName` is the asset's string name or ID (e.g. `"1000001"`). Amount must be positive; from and to must differ.

### CreateAssetIssue2

```go
txExt, err := mgr.CreateAssetIssue2(ctx, owner,
    "MyToken",         // name
    "MTK",             // abbreviation
    1_000_000_000,     // totalSupply (in smallest unit)
    1,                 // trxNum — TRX per asset unit (ICO pricing)
    1,                 // icoNum
    startTime,         // unix seconds
    endTime,           // unix seconds
    "description",
    "https://example.com",
    0,                 // freeAssetNetLimit (0 = unlimited)
    0,                 // publicFreeAssetNetLimit
    []trc10.FrozenSupply{{FrozenAmount: 100_000, FrozenDays: 30}},
)
```

Creates a new TRC10 token. Validates: name/abbreviation non-empty, totalSupply/trxNum/icoNum > 0, startTime < endTime, each `FrozenSupply` amount and days > 0.

### GetAssetIssueByAccount

```go
list, err := mgr.GetAssetIssueByAccount(ctx, addr)
// list.AssetIssue — list of TRC10 assets issued by this account
```

Returns `*api.AssetIssueList` of all assets issued by the address.

### GetAssetIssueByName

```go
asset, err := mgr.GetAssetIssueByName(ctx, "MyToken")
// asset.Name, asset.TotalSupply, asset.Precision
```

Returns `*core.AssetIssueContract` for the named asset.

### GetAssetIssueById

```go
asset, err := mgr.GetAssetIssueById(ctx, []byte("1000001"))
```

Returns `*core.AssetIssueContract` for the asset with the given numeric ID (as bytes).

### GetPaginatedAssetIssueList

```go
list, err := mgr.GetPaginatedAssetIssueList(ctx, 0, 50) // offset, limit (max 100)
```

Returns a page of all TRC10 assets on the network.

### UpdateAsset2

```go
txExt, err := mgr.UpdateAsset2(ctx, owner, "new description", "https://new.url", 100, 200)
```

Updates an asset's description, URL, and network limits. Owner must be the issuer.

## Error Handling

TRC10 methods return plain `fmt.Errorf` errors (not `pkg/types` sentinels) for input validation:

| Condition | Error |
|---|---|
| Empty `assetName` | `"asset name cannot be empty"` |
| Amount ≤ 0 | `"amount must be positive"` |
| Nil address | `"invalid owner address: nil"` |
| Same from/to | `"owner and to addresses cannot be the same"` |
| offset < 0 | `"offset cannot be negative"` |
| limit > 100 | `"limit cannot exceed 100"` |

gRPC-level errors propagate as-is from the node.

## See Also

- [TRC20 Package](trc20.md) — smart-contract-based token operations
- [Account Package](account.md) — balance and account queries
- [Client Package](client.md) — connection setup and `SignAndBroadcast`
- [Signer Package](signer.md) — transaction signing
