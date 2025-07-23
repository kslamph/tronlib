# tronlib Quick Reference

## Account Activation & Balance
- Use `client.GetAccount(ctx, addr)` to get `*core.Account`.
  - If `account.GetCreateTime() == 0`, the address is inactive.
  - Use `account.GetBalance()` (in SUN, 1 TRX = 1_000_000 SUN).
- `client.GetAccountNet(ctx, addr)` and `client.GetAccountResource(ctx, addr)` for bandwidth/energy info.

## TRX Transfer
- Use `transaction.NewTransaction(client).SetOwner(ownerAccount.Address()).TransferTRX(ctx, receiverAddr, amountInSun)`.
- Sign and broadcast: `.Sign(ownerAccount).Broadcast().GetReceipt()`.
- Wait for confirmation: `client.WaitForTransactionInfo(ctx, receipt.TxID, retries)`.

## Address & Account
- Create address: `types.NewAddress(base58String)` / `types.NewAddressFromHex(hexAddr)` / `types.NewAddressFromBytes([]byte)`.
- Create account from private key: `types.NewAccountFromPrivateKey(hexKey)`.

## Node/Client
- Use Nile for testing: `grpc.nile.trongrid.io:50051`.
- `client.NewClient(client.ClientConfig)` to create a client.
- `client.Close()` to close connections.

## Transaction Utilities
- `transaction.NewTransaction(client)` - create a new transaction builder.
- `SetOwner`, `SetFeelimit`, `SetExpiration` - set transaction options.
- `Sign`, `Broadcast`, `GetReceipt`, `GetError` - transaction lifecycle.
- `Freeze`, `Unfreeze`, `Delegate`, `Reclaim`, `Withdraw`, `ClaimReward` - resource management.

## Smart Contract
- `client.NewContractFromAddress(ctx, address)` - fetch contract ABI from chain.
- `smartcontract.NewContract(abi, address)` - create contract from ABI.
- `TriggerSmartContract`, `TriggerConstantSmartContract` - call contract methods.
- `EstimateEnergy` - estimate energy for contract call.

## Event & Log Parsing
- `parser.ContractsSliceToMap(contracts)` - for event parsing.
- `parser.ParseTransactionInfoLog(transactionInfo, contractsMap)` - parse logs.

## Key Functions by Package

### pkg/client
- `NewClient(config)`
- `GetAccount(ctx, addr)`
- `GetAccountNet(ctx, addr)`
- `GetAccountResource(ctx, addr)`
- `GetTransactionInfoById(ctx, txId)`
- `WaitForTransactionInfo(ctx, txId, timeout)`
- `NewContractFromAddress(ctx, addr)`

### pkg/transaction
- `NewTransaction(client)`
- `SetOwner(owner)`
- `SetFeelimit(limit)`
- `SetExpiration(seconds)`
- `TransferTRX(ctx, to, amount)`
- `Freeze(ctx, amount, resource)`
- `Unfreeze(ctx, amount, resource)`
- `Delegate(ctx, to, amount, resource)`
- `Reclaim(ctx, to, amount, resource)`
- `Withdraw(ctx)`
- `ClaimReward(ctx)`
- `Sign(signer)`
- `Broadcast()`
- `GetReceipt()`
- `GetError()`

### pkg/types
- `NewAddress(base58Addr)`
- `NewAddressFromHex(hexAddr)`
- `NewAddressFromBytes(bytes)`
- `MustNewAddress(base58Addr)`
- `NewAccountFromPrivateKey(hexKey)`
- `NewContract(abi, address)`
- `EncodeInput(method, params...)`
- `DecodeEventLog(topics, data)`

### pkg/parser
- `ContractsSliceToMap(contracts)`
- `ParseTransactionInfoLog(transactionInfo, contractsMap)`

### pkg/smartcontract
- `NewTRC20Contract(address, client)`
- `TRC20Contract.BalanceOf(ctx, address)`
- `TRC20Contract.Transfer(ctx, from, to, amount)`
- `TRC20Contract.Approve(ctx, owner, spender, amount)`
- `TRC20Contract.TransferFrom(ctx, from, to, amount)`

## Example: Check Account
```go
client, _ := client.NewClient(client.ClientConfig{NodeAddress: "grpc.nile.trongrid.io:50051"})
addr, _ := types.NewAddress("T...address...")
account, _ := client.GetAccount(ctx, addr)
active := account.GetCreateTime() != 0
balanceTRX := float64(account.GetBalance()) / 1_000_000
```

## Example: Transfer TRX
```go
owner, _ := types.NewAccountFromPrivateKey("hexkey...")
receiver, _ := types.NewAddress("T...address...")
tx := transaction.NewTransaction(client).SetOwner(owner.Address()).TransferTRX(ctx, receiver, 1_000_000)
receipt := tx.Sign(owner).Broadcast().GetReceipt()
if receipt.Err != nil { /* handle error */ }
```

## Example: TRC20 Transfer
```go
contract, _ := smartcontract.NewTRC20Contract(contractAddr, client)
from, _ := types.NewAccountFromPrivateKey("hexkey...")
to, _ := types.NewAddress("T...address...")
receipt := contract.Transfer(ctx, from.Address().String(), to.String(), amount).
  Sign(from).
  Broadcast().
  GetReceipt()
if receipt.Err != nil { /* handle error */ }
```

## Example: Parse Events
```go
contractsMap := parser.ContractsSliceToMap([]*smartcontract.Contract{contract})
events := parser.ParseTransactionInfoLog(txInfo, contractsMap)
```

## See Also
- See `examples/` and `integration_test/` for more usage patterns. 