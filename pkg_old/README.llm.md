# tronlib/pkg Directory – API Reference (LLM-Friendly)

This document lists all exported functions, types, and methods in the `tronlib/pkg` directory, categorized by package. It also notes which functions are safe for integration testing (i.e., do not update on-chain state).

---

## pkg/client – Tron Node Client

### Main Types
- **Client**: Manages connection to a Tron node with connection pooling.
- **ClientConfig**: Configuration for the client.
- **ConnPool**: Manages a pool of gRPC connections.

### Key Functions & Methods

#### Client Construction & Connection
- `NewClient(config ClientConfig) (*Client, error)`
- `Close()`
- `GetConnection(ctx) (*grpc.ClientConn, error)`
- `ReturnConnection(conn *grpc.ClientConn)`
- `GetTimeout() time.Duration`

#### Account & Resource Queries (Safe for integration tests)
- `GetAccount(ctx, *types.Address) (*core.Account, error)`
- `GetAccountNet(ctx, *types.Address) (*api.AccountNetMessage, error)`
- `GetAccountResource(ctx, *types.Address) (*api.AccountResourceMessage, error)`
- `GetRewardInfo(ctx, address string) (int64, error)`
- `ListWitnesses(ctx) (*api.WitnessList, error)`

#### Node & Network Info (Safe)
- `GetNodeInfo(ctx) (*core.NodeInfo, error)`
- `ListNodes(ctx) (*api.NodeList, error)`
- `GetChainParameters(ctx) (*core.ChainParameters, error)`
- `GetBlockById(ctx, blockId []byte) (*core.Block, error)`
- `GetBlockByNum(ctx, blockNumber int64) (*api.BlockExtention, error)`
- `GetNowBlock(ctx) (*api.BlockExtention, error)`

#### Transaction & Contract (NOT safe for integration tests)
- `CreateTransferTransaction(ctx, from, to string, amount int64) (*api.TransactionExtention, error)`
- `CreateFreezeTransaction(ctx, ownerAddress string, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error)`
- `CreateUnfreezeTransaction(ctx, ownerAddress string, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error)`
- `CreateDelegateResourceTransaction(ctx, ownerAddress, delegateTo string, amount int64, resource core.ResourceCode, lock bool, blocksToLock ...int64) (*api.TransactionExtention, error)`
- `CreateUndelegateResourceTransaction(ctx, ownerAddress, reclaimFrom string, amount int64, resource core.ResourceCode) (*api.TransactionExtention, error)`
- `CreateWithdrawBalanceTransaction(ctx, ownerAddress string) (*api.TransactionExtention, error)`
- `CreateWithdrawExpireUnfreezeTransaction(ctx, ownerAddress string) (*api.TransactionExtention, error)`
- `DeployContract(ctx, owner *types.Address, bytecode []byte, abi string, name string, originEnergyLimit int64, consumeUserResourcePercent int64, constructorParams ...interface{}) (*api.TransactionExtention, error)`
- `CreateDeployContractTransaction(ctx, contract *core.CreateSmartContract) (*api.TransactionExtention, error)`
- `CreateTriggerSmartContractTransaction(ctx, ownerAddress, contractAddress []byte, data []byte, callValue int64) (*api.TransactionExtention, error)`
- `UpdateSetting(ctx, contract *core.UpdateSettingContract) (*api.TransactionExtention, error)`
- `UpdateEnergyLimit(ctx, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error)`
- `ClearContractABI(ctx, contract *core.ClearABIContract) (*api.TransactionExtention, error)`
- `AccountPermissionUpdate(ctx, contract *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error)`
- `UpdateAccount2(ctx, ownerAddress types.Address, accountName string) (*api.TransactionExtention, error)`

#### Voting (NOT safe)
- `VoteWitnessAccount2(ctx, contract *core.VoteWitnessContract) (*api.TransactionExtention, error)`

#### Contract Calls (Safe if read-only)
- `NewContractFromAddress(ctx, address *types.Address) (*smartcontract.Contract, error)`
- `TriggerConstantSmartContract(ctx, contract *smartcontract.Contract, ownerAddress *types.Address, data []byte) ([][]byte, error)`
- `EstimateEnergy(ctx, contract *smartcontract.Contract, ownerAddress *types.Address, data []byte) (int64, error)`
- `GetContractInfo(ctx, address []byte) (*core.SmartContractDataWrapper, error)`

#### Transaction Info (Safe)
- `GetTransactionById(ctx, txId string) (*core.Transaction, error)`
- `GetTransactionInfoById(ctx, txId string) (*core.TransactionInfo, error)`
- `GetTransactionInfoByBlockNum(ctx, blockNumber int64) (*api.TransactionInfoList, error)`
- `WaitForTransactionInfo(ctx, txId string) (*core.TransactionInfo, error)`

#### Delegation (Read-only queries are safe)
- `GetDelegatedResourceV2(ctx, req *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)`
- `GetDelegatedResourceAccountIndexV2(ctx, address []byte) (*core.DelegatedResourceAccountIndex, error)`
- `GetCanDelegatedMaxSize(ctx, req *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error)`
- `GetCanWithdrawUnfreezeAmount(ctx, req *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error)`

---

## pkg/smartcontract – Smart Contract Utilities

- `DecodeABI(abi string) (*core.SmartContract_ABI, error)`
- `NewContract(abi string, address string) (*Contract, error)`
- `NewContractFromABI(abi *core.SmartContract_ABI, address string) (*Contract, error)`

**Contract methods (Safe for integration tests):**
- `DecodeEventLog(topics, data)`
- `DecodeEventSignature(signature)`
- `DecodeInputData(data)`
- `DecodeResult(method, result)`
- `EncodeInput(method, params...)`

---

## pkg/types – Core Types

- **Account**: `NewAccountFromPrivateKey`, `NewAccountFromHDWallet`, `Address()`, `PublicKey()`, `Sign()`, `MultiSign()`, `SignMessageV2()`, `PrivateKeyHex()`
- **Address**: `NewAddress`, `NewAddressFromBytes`, `NewAddressFromHex`, `MustNewAddress`, `MustNewAddressFromBytes`, `MustNewAddressFromHex`, `Bytes()`, `Hex()`, `String()`
- **KMSAccount**: `NewKMSAccount`, `Address()`, `PublicKey()`, `Sign()`, `MultiSign()`, `SignMessageV2()`
- **Signer** (interface): `Address()`, `PublicKey()`, `Sign()`, `SignWithPermissionID()`, `SignMessageV2()`
- **KMSClientInterface** (interface): `SignDigest()`, `GetPublicKey()`

All of these are safe for integration/unit tests except for `Sign`/`MultiSign` if they are used to submit on-chain transactions.

---

## pkg/helper – Helper Functions (All safe for integration tests)

- `ContractsSliceToMap(contracts)`
- `GetTxid(tx)`
- `SunToTrx(amount)`
- `TrxToSun(amount)`
- `SunToTrxString(sun)`
- `SunToTrxStringCommas(sun)`
- `ParseTransactionInfoLog(transactionInfo, contracts)`

---

## pkg/crypto – Cryptographic Utilities (Safe)

- `VerifyMessageV2(address, message, hexSignature) (bool, error)`

---

## pkg/trc20 – TRC20 Contract Helpers

- **TRC20Contract**: `NewTRC20Contract(client, address)`
- Methods: `Allowance`, `Approve`, `BalanceOf`, `Decimals`, `Name`, `Symbol`, `TotalSupply`, `Transfer`, `TransferFrom`

Only read-only methods (`Allowance`, `BalanceOf`, `Decimals`, `Name`, `Symbol`, `TotalSupply`) are safe for integration tests.

---

# Integration Test Coverage Guidance

- **Safe for integration tests:** All read-only queries, helpers, cryptographic checks, and contract ABI utilities.
- **NOT safe:** Any function that creates, updates, or broadcasts on-chain transactions (see above for specifics). 