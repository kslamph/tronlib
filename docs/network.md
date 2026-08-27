# Network Package

The `network` package provides read-only queries for blocks, transactions, chain parameters, and node information. It is stateless and always accessed through the client gateway (`cli.Network()`).

## Import

```go
import "github.com/kslamph/tronlib/pkg/network"
```

## Construction

```go
// Via client gateway
cli, _ := client.NewClient("grpc://grpc.trongrid.io:50051")
defer cli.Close()
mgr := cli.Network()
```

## Operations

### GetNowBlock

```go
blk, err := mgr.GetNowBlock(ctx)
// blk.BlockHeader.RawData.Number — block number
// blk.BlockHeader.RawData.Timestamp — unix millis
// blk.Transactions — transaction list
```

Returns the latest `*api.BlockExtention`.

### GetBlockByNumber

```go
blk, err := mgr.GetBlockByNumber(ctx, 12345678)
```

Returns `*api.BlockExtention` for the given block number.

### GetBlocksByLimit

```go
blks, err := mgr.GetBlocksByLimit(ctx, 100, 109) // max 100 at once
```

Returns `*api.BlockListExtention` for the range `[start, end)`. End must be ≥ start and within 100 blocks.

### GetLatestBlocks

```go
blks, err := mgr.GetLatestBlocks(ctx, 10) // max 100
```

Returns the N most recent blocks as `*api.BlockListExtention`.

### GetTransactionById

```go
tx, err := mgr.GetTransactionById(ctx, "a1b2c3...64hexchars")
// tx.RawData — contract list, expiration, etc.
// tx.Signature — signer signatures
```

Transaction ID must be 64 hex characters; `0x` prefix is tolerated.

### GetTransactionInfoById

```go
info, err := mgr.GetTransactionInfoById(ctx, txIdHex)
// info.Receipt.Result — SUCCESS/FAILED
// info.EnergyUsageTotal — energy consumed
// info.Fee — total fee in SUN
// info.ContractResult — per-contract results
```

Returns `*core.TransactionInfo` with receipt, energy, and fee details.

### GetTransactionInfoByBlockNum

```go
txInfoList, err := mgr.GetTransactionInfoByBlockNum(ctx, 12345678)
// txInfoList.TransactionInfo — info records for all txs in the block
```

Returns `*api.TransactionInfoList` for every transaction in the block.

### GetChainParameters

```go
params, err := mgr.GetChainParameters()
```

Returns `*core.ChainParameters` with the current network configuration (energy limits, bandwidth fees, etc.).

### ListNodes

```go
nodes, err := mgr.ListNodes(ctx)
```

Returns `*api.NodeList` of active super-representative nodes.

## Error Handling

| Sentinel | Meaning |
|---|---|
| `types.ErrInvalidParameter` | Empty tx ID, invalid block number, or ID length ≠ 64 hex chars |
| `types.ErrNetworkError` | gRPC call failed |

Transaction IDs are validated: must be exactly 64 hex characters (32 bytes). A `0x` prefix is stripped automatically.

## See Also

- [Client Package](client.md) — connection setup and gateway methods
- [Account Package](account.md) — account queries
- [Resources Package](resources.md) — freeze/unfreeze/delegate
- [Types Package](types.md) — `*types.Address` and error sentinels
