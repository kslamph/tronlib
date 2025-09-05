# TronLib SDK API Reference

This document provides comprehensive documentation for the TronLib SDK, covering all public functions, methods, types, and constants. The SDK is organized into several packages, each with specific responsibilities.

## Table of Contents

1. [client package](#client-package)
2. [account package](#account-package)
3. [trc20 package](#trc20-package)
4. [smartcontract package](#smartcontract-package)
5. [types package](#types-package)
6. [signer package](#signer-package)
7. [eventdecoder package](#eventdecoder-package)
8. [utils package](#utils-package)

---

## client package

Package client provides connection management and RPC helpers for interacting with TRON full nodes over gRPC.

### Types

#### Client

```go
type Client struct {
    // contains filtered or unexported fields
}
```

Client manages connection to a single Tron node with connection pooling. The Client maintains a pool of gRPC connections to improve performance for concurrent operations. It automatically handles connection lifecycle, including reconnection and timeout management.

Use NewClient to create a new client instance, and always call Close when finished to free up resources.

#### BroadcastOptions

```go
type BroadcastOptions struct {
    FeeLimit       int64         // Fee limit for the transaction
    PermissionID   int32         // Permission ID for the transaction
    WaitForReceipt bool          // Wait for transaction receipt
    WaitTimeout    time.Duration // Timeout for waiting for receipt
    PollInterval   time.Duration // Polling interval when waiting for receipt
}
```

BroadcastOptions controls high-level signing and broadcasting workflows. Fields with zero values are defaulted by DefaultBroadcastOptions unless explicitly documented otherwise.

These options control how transactions are signed, broadcast, and confirmed. Use DefaultBroadcastOptions() to get sensible defaults, then modify as needed.

#### BroadcastResult

```go
type BroadcastResult struct {
    TxID           string                 `json:"txID"`
    Success        bool                   `json:"success"`
    Code           api.ReturnResponseCode `json:"returnCode"`    // TRON return code
    Message        string                 `json:"returnMessage"` // TRON return message concat with contract return message
    ConstantReturn [][]byte               // test if nil before use
    EnergyUsage    int64                  `json:"energyUsed,omitempty"`
    NetUsage       int64                  `json:"netUsage,omitempty"`
    Logs           []*core.TransactionInfo_Log `json:"logs,omitempty"`
}
```

BroadcastResult summarizes the outcome of a simulation or a broadcasted transaction, including TRON return status, resource usage, and logs.

This struct contains the results of either a Simulate or SignAndBroadcast operation. When WaitForReceipt is true in SignAndBroadcast, additional fields like EnergyUsage and Logs will be populated with data from the transaction receipt.

#### Option

```go
type Option func(*clientOptions)
```

Functional options for Client configuration.

### Functions

#### NewClient

```go
func NewClient(endpoint string, opts ...Option) (*Client, error)
```

NewClient creates a new client to a TRON node using endpoint like grpc://host:port or grpcs://host:port.

The endpoint must include a scheme (grpc:// or grpcs://) followed by host and port. The client maintains a connection pool for improved performance.

Options can be used to configure:
- Connection timeout with WithTimeout()
- Connection pool size with WithPool()

Example:
```go
cli, err := client.NewClient("grpc://127.0.0.1:50051",
    client.WithTimeout(30*time.Second),
    client.WithPool(5, 10))
if err != nil {
    // handle error
}
defer cli.Close()
```

Returns an error if the endpoint is invalid or connection fails.

#### DefaultBroadcastOptions

```go
func DefaultBroadcastOptions() BroadcastOptions
```

DefaultBroadcastOptions returns sane defaults for broadcasting transactions.

The default options are:
- FeeLimit: 150,000,000 SUN (0.15 TRX)
- PermissionID: 0 (owner permission)
- WaitForReceipt: true (wait for transaction confirmation)
- WaitTimeout: 15 seconds
- PollInterval: 3 seconds

#### WithTimeout

```go
func WithTimeout(d time.Duration) Option
```

WithTimeout sets the default timeout for client operations when the context has no deadline.

This option configures the default timeout that will be applied to operations when the context doesn't have a deadline. The default is 30 seconds.

#### WithPool

```go
func WithPool(initConnections, maxConnections int) Option
```

WithPool configures the initial and maximum connections for the pool.

This option configures the connection pool size:
- initConnections: Number of connections to create initially (default: 1)
- maxConnections: Maximum number of connections in the pool (default: 5)

### Client Methods

#### Account

```go
func (c *Client) Account() *account.AccountManager
```

Account is the gateway method to access the AccountManager. It returns an *account.AccountManager, satisfying the high-level API need.

#### SmartContract

```go
func (c *Client) SmartContract() *smartcontract.Manager
```

SmartContract is the gateway method to access the Manager.

#### TRC20

```go
func (c *Client) TRC20(addr *types.Address) *trc20.TRC20Manager
```

TRC20 returns a TRC20 manager for a given token address.

#### Network

```go
func (c *Client) Network() *network.NetworkManager
```

Network returns the high-level NetworkManager.

#### Resources

```go
func (c *Client) Resources() *resources.ResourcesManager
```

Resources returns the high-level ResourcesManager.

#### TRC10

```go
func (c *Client) TRC10() *trc10.TRC10Manager
```

TRC10 returns the high-level TRC10Manager.

#### Voting

```go
func (c *Client) Voting() *voting.VotingManager
```

Voting returns the high-level VotingManager.

#### GetConnection

```go
func (c *Client) GetConnection(ctx context.Context) (*grpc.ClientConn, error)
```

GetConnection safely gets a connection from the pool.

This method should be used in conjunction with ReturnConnection to properly manage connection lifecycle. It applies the client's default timeout if the context doesn't have a deadline.

Returns ErrClientClosed if the client has been closed, or ErrConnectionFailed if no connection is available.

#### ReturnConnection

```go
func (c *Client) ReturnConnection(conn *grpc.ClientConn)
```

ReturnConnection safely returns a connection to the pool.

This method should always be called after GetConnection to return the connection to the pool for reuse. It is safe to call on a closed client.

#### Close

```go
func (c *Client) Close()
```

Close closes the client and all connections in the pool.

This method should be called when the client is no longer needed to free up resources. It is safe to call multiple times.

#### GetTimeout

```go
func (c *Client) GetTimeout() time.Duration
```

GetTimeout returns the client's configured timeout.

This timeout is applied to operations when the context doesn't have a deadline.

#### GetNodeAddress

```go
func (c *Client) GetNodeAddress() string
```

GetNodeAddress returns the configured node address.

The address is in the format scheme://host:port (e.g., grpc://127.0.0.1:50051).

#### IsConnected

```go
func (c *Client) IsConnected() bool
```

IsConnected checks if the client is connected (not closed).

Returns true if the client is still open and can be used for operations, false if it has been closed.

#### Simulate

```go
func (c *Client) Simulate(ctx context.Context, anytx any) (*BroadcastResult, error)
```

Simulate performs a read-only execution of a single-contract transaction and returns a BroadcastResult with constant return data, energy usage, and logs.

This method allows you to test a transaction without actually broadcasting it to the network. It's useful for estimating energy usage and checking if a transaction would succeed before actually sending it.

Supported input types are *api.TransactionExtention and *core.Transaction. The transaction must contain exactly one contract and must not be expired.

Example:
```go
sim, err := cli.Simulate(ctx, txExt)
if err != nil {
    // handle error
}
if !sim.Success {
    // transaction would fail
}
fmt.Printf("Energy usage: %d\n", sim.EnergyUsage)
```

#### SignAndBroadcast

```go
func (c *Client) SignAndBroadcast(ctx context.Context, anytx any, opt BroadcastOptions, signers ...signer.Signer) (*BroadcastResult, error)
```

SignAndBroadcast signs a single-contract transaction using the provided signers (if any), applies BroadcastOptions, broadcasts it to the network, and optionally waits for receipt. It returns a BroadcastResult with txid, TRON return code/message, and, if waiting, resource usage and logs.

This is the primary method for sending transactions to the TRON network. It handles signing, broadcasting, and (optionally) waiting for the transaction to be confirmed.

Supported input types are *api.TransactionExtention and *core.Transaction.

Example:
```go
opts := client.DefaultBroadcastOptions()
opts.FeeLimit = 100_000_000
opts.WaitForReceipt = true

result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
if err != nil {
    // handle error
}
if result.Success {
    fmt.Printf("Transaction successful: %s\n", result.TxID)
}
```

---

## account package

Package account provides high-level helpers to query and mutate TRON accounts, such as retrieving balances, resources, and building TRX transfers.

### Types

#### AccountManager

```go
type AccountManager struct {
    // contains filtered or unexported fields
}
```

AccountManager provides high-level account operations.

The AccountManager allows you to query account information, retrieve balances, and create TRX transfer transactions. It works with a connection provider (typically a *client.Client) to communicate with the TRON network.

### Functions

#### NewManager

```go
func NewManager(conn lowlevel.ConnProvider) *AccountManager
```

NewManager creates a new account manager.

The account manager requires a connection provider (typically a *client.Client) to communicate with the TRON network.

Example:
```go
cli, err := client.NewClient("grpc://127.0.0.1:50051")
if err != nil {
    // handle error
}
defer cli.Close()

accountMgr := account.NewManager(cli)
```

### AccountManager Methods

#### GetAccount

```go
func (m *AccountManager) GetAccount(ctx context.Context, address *types.Address) (*core.Account, error)
```

GetAccount retrieves account information by address.

This method fetches detailed account information from the TRON network, including balance, resources, and other account properties.

Returns an error if the address is invalid or if the account doesn't exist.

#### GetAccountNet

```go
func (m *AccountManager) GetAccountNet(ctx context.Context, address *types.Address) (*api.AccountNetMessage, error)
```

GetAccountNet retrieves account bandwidth information.

#### GetAccountResource

```go
func (m *AccountManager) GetAccountResource(ctx context.Context, address *types.Address) (*api.AccountResourceMessage, error)
```

GetAccountResource retrieves account energy information.

#### GetBalance

```go
func (m *AccountManager) GetBalance(ctx context.Context, address *types.Address) (int64, error)
```

GetBalance retrieves the TRX balance for an address (convenience method).

This method returns the TRX balance in SUN (1 TRX = 1,000,000 SUN). It's a convenience method that fetches the full account information and returns just the balance.

Example:
```go
balance, err := accountMgr.GetBalance(ctx, address)
if err != nil {
    // handle error
}
trxBalance := float64(balance) / 1_000_000
fmt.Printf("Balance: %.6f TRX\n", trxBalance)
```

#### TransferTRX

```go
func (m *AccountManager) TransferTRX(ctx context.Context, from *types.Address, to *types.Address, amount int64) (*api.TransactionExtention, error)
```

TransferTRX creates an unsigned TRX transfer transaction.

This method creates a TRX transfer transaction from one address to another. The transaction is not signed or broadcast - use client.SignAndBroadcast to complete the transfer.

The amount should be specified in SUN (1 TRX = 1,000,000 SUN).

Example:
```go
txExt, err := accountMgr.TransferTRX(ctx, from, to, 1_000_000) // 1 TRX
if err != nil {
    // handle error
}

// Sign and broadcast the transaction
opts := client.DefaultBroadcastOptions()
result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
```

---

## trc20 package

Package trc20 provides a typed, ergonomic interface for TRC20 tokens.

It wraps a generic smart contract client with convenience methods that:
- Cache immutable properties (name, symbol, decimals)
- Convert between human decimals and on-chain integer amounts
- Expose common actions (balance, allowance, approve, transfer)

The manager requires a configured *client.Client and the token contract address. It preloads common metadata using the client's timeout so that subsequent calls are efficient.

### Types

#### TRC20Manager

```go
type TRC20Manager struct {
    // contains filtered or unexported fields
}
```

TRC20Manager provides a high-level, type-safe interface for TRC20 token interactions.

The TRC20Manager wraps a smart contract instance with convenience methods for common TRC20 operations. It automatically handles:
- Conversion between human-readable decimal amounts and on-chain integer values
- Caching of immutable token properties (name, symbol, decimals)
- Encoding and decoding of method calls and return values

Use NewManager to create a new TRC20Manager instance for a specific token contract.

### Functions

#### NewManager

```go
func NewManager(tronClient lowlevel.ConnProvider, contractAddress *types.Address) (*TRC20Manager, error)
```

NewManager constructs a TRC20 manager bound to the given token contract address using the provided TRON connection provider.

This function creates a new TRC20Manager instance for interacting with a specific TRC20 token contract. It automatically fetches and caches the token's metadata (name, symbol, decimals) for efficient subsequent operations.

Example:
```go
cli, err := client.NewClient("grpc://127.0.0.1:50051")
if err != nil {
    // handle error
}
defer cli.Close()

tokenAddr, err := types.NewAddress("TContractAddressHere")
if err != nil {
    // handle error
}

trc20Mgr, err := trc20.NewManager(cli, tokenAddr)
if err != nil {
    // handle error
}
```

#### ToWeiWithDecimals

```go
func ToWeiWithDecimals(amount decimal.Decimal, decimals uint8) (*big.Int, error)
```

ToWeiWithDecimals converts a user-facing decimal amount into on-chain units using the provided decimals.

#### FromWeiWithDecimals

```go
func FromWeiWithDecimals(value *big.Int, decimals uint8) (decimal.Decimal, error)
```

FromWeiWithDecimals converts raw on-chain units into a user-facing decimal using the provided decimals.

### TRC20Manager Methods

#### Name

```go
func (t *TRC20Manager) Name(ctx context.Context) (string, error)
```

Name returns the token name, fetching and caching it on first call.

This method returns the name of the TRC20 token (e.g., "TetherUSD"). The result is cached after the first successful call for improved performance.

Example:
```go
name, err := trc20Mgr.Name(ctx)
if err != nil {
    // handle error
}
fmt.Printf("Token name: %s\n", name)
```

#### Symbol

```go
func (t *TRC20Manager) Symbol(ctx context.Context) (string, error)
```

Symbol returns the token symbol, fetching and caching it on first call.

This method returns the symbol of the TRC20 token (e.g., "USDT"). The result is cached after the first successful call for improved performance.

Example:
```go
symbol, err := trc20Mgr.Symbol(ctx)
if err != nil {
    // handle error
}
fmt.Printf("Token symbol: %s\n", symbol)
```

#### Decimals

```go
func (t *TRC20Manager) Decimals(ctx context.Context) (uint8, error)
```

Decimals returns the token's decimals, fetching and caching it on first call.

This method returns the number of decimal places the token uses for display purposes. For example, USDT typically uses 6 decimals, meaning 1 USDT is represented as 1000000 in on-chain integer values. The result is cached after the first successful call.

Example:
```go
decimals, err := trc20Mgr.Decimals(ctx)
if err != nil {
    // handle error
}
fmt.Printf("Token decimals: %d\n", decimals)
```

#### TotalSupply

```go
func (t *TRC20Manager) TotalSupply(ctx context.Context) (decimal.Decimal, error)
```

TotalSupply retrieves the total supply of the token as a decimal.Decimal.

#### BalanceOf

```go
func (t *TRC20Manager) BalanceOf(ctx context.Context, ownerAddress *types.Address) (decimal.Decimal, error)
```

BalanceOf retrieves the owner's balance as a decimal.Decimal.

This method returns the token balance of the specified address. The balance is automatically converted from the on-chain integer representation to a human-readable decimal value using the token's decimals.

Example:
```go
balance, err := trc20Mgr.BalanceOf(ctx, address)
if err != nil {
    // handle error
}
fmt.Printf("Token balance: %s\n", balance.String())
```

#### Transfer

```go
func (t *TRC20Manager) Transfer(ctx context.Context, fromAddress *types.Address, toAddress *types.Address, amount decimal.Decimal) (*api.TransactionExtention, error)
```

Transfer transfers tokens from the caller to a recipient using a decimal.Decimal amount. Returns txid (hex) and the raw transaction extension.

This method creates a TRC20 token transfer transaction from one address to another. The transaction is not signed or broadcast - use client.SignAndBroadcast to complete the transfer. The amount should be specified as a decimal value (not in the smallest token units).

Example:
```go
amount := decimal.NewFromFloat(10.5) // 10.5 tokens
txExt, err := trc20Mgr.Transfer(ctx, from, to, amount)
if err != nil {
    // handle error
}

// Sign and broadcast the transaction
opts := client.DefaultBroadcastOptions()
opts.FeeLimit = 50_000_000 // 50 TRX max fee for TRC20 operations
result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
```

#### Approve

```go
func (t *TRC20Manager) Approve(ctx context.Context, ownerAddress *types.Address, spenderAddress *types.Address, amount decimal.Decimal) (*api.TransactionExtention, error)
```

Approve authorizes a spender for a given amount using decimal.Decimal.

This method creates an approve transaction that allows a spender address to spend a specified amount of tokens on behalf of the owner. The transaction is not signed or broadcast - use client.SignAndBroadcast to complete the approval.

Example:
```go
amount := decimal.NewFromFloat(100.0) // Allow spending 100 tokens
txExt, err := trc20Mgr.Approve(ctx, owner, spender, amount)
if err != nil {
    // handle error
}

// Sign and broadcast the transaction
opts := client.DefaultBroadcastOptions()
result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
```

#### Allowance

```go
func (t *TRC20Manager) Allowance(ctx context.Context, ownerAddress *types.Address, spenderAddress *types.Address) (decimal.Decimal, error)
```

Allowance retrieves the spender's allowance over the owner's tokens as a decimal.Decimal.

This method returns the amount of tokens that the spender is allowed to spend on behalf of the owner. The allowance is automatically converted from the on-chain integer representation to a human-readable decimal value.

Example:
```go
allowance, err := trc20Mgr.Allowance(ctx, owner, spender)
if err != nil {
    // handle error
}
fmt.Printf("Allowance: %s\n", allowance.String())
```

---

## smartcontract package

Package smartcontract provides high-level helpers to deploy, query, and interact with TRON smart contracts. It includes a package-level Manager for deployment/admin tasks and a per-address Instance for bound interaction.

### Types

#### Manager

```go
type Manager struct {
    // contains filtered or unexported fields
}
```

Manager provides high-level smart contract operations.

The Manager allows you to deploy new smart contracts and perform administrative operations on existing contracts. For interacting with deployed contracts, use the Instance type which provides methods for calling contract functions.

#### Instance

```go
type Instance struct {
    ABI     *core.SmartContract_ABI
    Address *types.Address
    Client  contractClient
    // contains filtered or unexported fields
}
```

Instance represents a high-level client bound to a deployed smart contract address and ABI, providing helpers for encoding inputs, invoking methods, constant calls, and decoding results and events.

The Instance allows you to interact with a deployed smart contract by calling its methods, both state-changing (Invoke) and read-only (Call) functions. It handles ABI encoding/decoding automatically.

#### SimulateResult

```go
type SimulateResult struct {
    Energy    int64
    APIResult *api.Return
    Logs      []*core.TransactionInfo_Log
}
```

SimulateResult captures details from a constant-call simulation.

### Functions

#### NewManager

```go
func NewManager(conn lowlevel.ConnProvider) *Manager
```

NewManager creates a new smart contract manager.

The smart contract manager requires a connection provider (typically a *client.Client) to communicate with the TRON network.

Example:
```go
cli, err := client.NewClient("grpc://127.0.0.1:50051")
if err != nil {
    // handle error
}
defer cli.Close()

contractMgr := smartcontract.NewManager(cli)
```

#### NewInstance

```go
func NewInstance(tronClient contractClient, contractAddress *types.Address, abi ...any) (*Instance, error)
```

NewInstance constructs a contract instance for the given address using the provided TRON client. The ABI can be omitted to fetch from the network, or supplied as either a JSON string or a *core.SmartContract_ABI.

This function creates a new Instance for interacting with a deployed smart contract. If no ABI is provided, it will be fetched from the network (the contract must have its ABI published on-chain).

Example:
```go
// With ABI provided
instance, err := smartcontract.NewInstance(cli, contractAddr, abiJSON)
if err != nil {
    // handle error
}

// Without ABI (fetch from network)
instance, err := smartcontract.NewInstance(cli, contractAddr)
if err != nil {
    // handle error
}
```

### Manager Methods

#### Instance

```go
func (m *Manager) Instance(contractAddress *types.Address, abi ...any) (*Instance, error)
```

Instance creates a bound contract instance for a deployed contract address. The ABI can be omitted to fetch from the network, or supplied as JSON string or *core.SmartContract_ABI.

#### Deploy

```go
func (m *Manager) Deploy(ctx context.Context, ownerAddress *types.Address, contractName string, abi any, bytecode []byte, callValue, consumeUserResourcePercent, originEnergyLimit int64, constructorParams ...interface{}) (*api.TransactionExtention, error)
```

Deploy deploys a smart contract with constructor parameters.

This method creates a transaction to deploy a new smart contract to the TRON network. The transaction is not signed or broadcast - use client.SignAndBroadcast to complete the deployment.

Parameters:
- ownerAddress: Address that will own the contract
- contractName: Human-readable name for the contract
- abi: Contract ABI (string, *core.SmartContract_ABI, or nil)
- bytecode: Compiled contract bytecode
- callValue: TRX amount to send with deployment (in SUN)
- consumeUserResourcePercent: Percentage of energy consumed by user (0-100)
- originEnergyLimit: Maximum energy the contract can consume
- constructorParams: Optional constructor parameters

Example:
```go
txExt, err := contractMgr.Deploy(ctx, owner, "MyContract", abiJSON, bytecode, 0, 100, 30000, param1, param2)
if err != nil {
    // handle error
}

// Sign and broadcast the transaction
opts := client.DefaultBroadcastOptions()
result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
```

#### EstimateEnergy

```go
func (m *Manager) EstimateEnergy(ctx context.Context, ownerAddress, contractAddress *types.Address, data []byte, callValue int64) (*api.EstimateEnergyMessage, error)
```

EstimateEnergy estimates energy required for smart contract execution. Use client.Simulate to know energy required for a transaction.

#### GetContract

```go
func (m *Manager) GetContract(ctx context.Context, contractAddress *types.Address) (*core.SmartContract, error)
```

GetContract gets smart contract information.

#### GetContractInfo

```go
func (m *Manager) GetContractInfo(ctx context.Context, contractAddress *types.Address) (*core.SmartContractDataWrapper, error)
```

GetContractInfo gets smart contract detailed information.

#### UpdateSetting

```go
func (m *Manager) UpdateSetting(ctx context.Context, ownerAddress, contractAddress *types.Address, consumeUserResourcePercent int64) (*api.TransactionExtention, error)
```

UpdateSetting updates smart contract settings.

#### UpdateEnergyLimit

```go
func (m *Manager) UpdateEnergyLimit(ctx context.Context, ownerAddress, contractAddress *types.Address, originEnergyLimit int64) (*api.TransactionExtention, error)
```

UpdateEnergyLimit updates smart contract energy limit.

#### ClearContractABI

```go
func (m *Manager) ClearContractABI(ctx context.Context, ownerAddress, contractAddress *types.Address) (*api.TransactionExtention, error)
```

ClearContractABI clears smart contract ABI.

### Instance Methods

#### Invoke

```go
func (i *Instance) Invoke(ctx context.Context, owner *types.Address, callValue int64, method string, params ...interface{}) (*api.TransactionExtention, error)
```

Invoke builds a transaction that calls a state-changing method on the contract. The result should be signed and broadcasted by the caller.

This method creates a transaction to call a state-changing function on the smart contract. The transaction is not signed or broadcast - use client.SignAndBroadcast to complete the call.

Parameters:
- ctx: Context for the operation
- owner: Address that will execute the transaction
- callValue: Amount of TRX to send with the call (in SUN)
- method: Name of the contract method to call
- params: Optional parameters to pass to the method

Example:
```go
txExt, err := instance.Invoke(ctx, owner, 0, "setValue", uint64(42))
if err != nil {
    // handle error
}

// Sign and broadcast the transaction
opts := client.DefaultBroadcastOptions()
result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
```

#### Call

```go
func (i *Instance) Call(ctx context.Context, owner *types.Address, method string, params ...interface{}) (interface{}, error)
```

Call performs a constant (read-only) method call and returns the decoded result value. If the method has multiple outputs, the return is a []interface{}; if one output, it's that single value; if none, nil.

This method calls a read-only function on the smart contract and returns the result. Unlike Invoke, this method doesn't create a transaction and doesn't change the blockchain state.

Parameters:
- ctx: Context for the operation
- owner: Address making the call (for permission checks)
- method: Name of the contract method to call
- params: Optional parameters to pass to the method

Example:
```go
result, err := instance.Call(ctx, owner, "getValue")
if err != nil {
    // handle error
}
value, ok := result.(uint64)
if !ok {
    // handle type assertion
}
fmt.Printf("Value: %d\n", value)
```

#### Simulate

```go
func (i *Instance) Simulate(ctx context.Context, owner *types.Address, callValue int64, method string, params ...interface{}) (*SimulateResult, error)
```

Simulate performs a read-only execution of the specified method and returns energy usage, raw API result, and logs without decoding the return value.

#### Encode

```go
func (i *Instance) Encode(method string, params ...interface{}) ([]byte, error)
```

Encode encodes a method invocation into call data. For constructors, pass an empty method name and only parameters.

#### DecodeResult

```go
func (i *Instance) DecodeResult(method string, data []byte) (interface{}, error)
```

DecodeResult decodes a method's return bytes into a Go value. Single-output methods return the value directly; multiple outputs return []interface{}.

#### DecodeInput

```go
func (i *Instance) DecodeInput(data []byte) (*utils.DecodedInput, error)
```

DecodeInput decodes input call data to a typed representation.

---

## types package

Package types defines fundamental types and error values used across the TRON SDK, including Address, transaction wrappers, and common constants.

### Types

#### Address

```go
type Address struct {
    // contains filtered or unexported fields
}
```

Address represents a TRON address that can be stored in different formats. Always construct via the NewAddress[...] helpers to ensure validation.

The Address type can represent TRON addresses in multiple formats:
- Base58: T-prefixed 34-character string (e.g., "TLCuBEbV6jp9432t4Xhg5E5j7v7vK4gkgX")
- Bytes: 0x41-prefixed 21-byte array
- Hex: 42-character hex string with 0x41 prefix
- EVM Bytes: 20-byte array without prefix (for Ethereum compatibility)

Use the various constructor functions to create Address instances safely.

#### TronError

```go
type TronError struct {
    Code    int32
    Message string
    Cause   error
}
```

TronError wraps TRON-specific errors with additional context.

#### TransactionError

```go
type TransactionError struct {
    TxID    string
    Message string
    Cause   error
}
```

TransactionError represents transaction-specific errors.

#### ContractError

```go
type ContractError struct {
    ContractAddress string
    Method          string
    Message         string
    Cause           error
}
```

ContractError represents smart contract execution errors.

#### Network

```go
type Network struct {
    Name     string
    ChainID  string
    NodeURLs []string
}
```

Network represents a TRON network.

#### ResourceType

```go
type ResourceType int
```

ResourceType represents the type of resource.

#### ContractType

```go
type ContractType int
```

ContractType represents the type of smart contract.

#### TransactionStatus

```go
type TransactionStatus int
```

TransactionStatus represents the status of a transaction.

### Constants

```go
const (
    // Network constants
    TronMainNet = "mainnet"
    TronTestNet = "testnet"
    TronNileNet = "nile"

    // Address constants
    AddressPrefixByte   = 0x41
    AddressLength       = 21
    AddressHexLength    = 42 // Including 0x prefix
    AddressBase58Length = 34

    // Transaction constants
    DefaultFeeLimit           = 1000000 // 1 TRX in SUN
    DefaultTransactionTimeout = 30 * time.Second
    DefaultExpiration         = 10 * time.Minute

    // Energy constants
    DefaultEnergyLimit = 10000000
    EnergyPerByte      = 1

    // Bandwidth constants
    DefaultBandwidthLimit = 5000
    BandwidthPerByte      = 1

    // Resource constants
    SunPerTRX = 1000000 // 1 TRX = 1,000,000 SUN

    // Contract constants
    DefaultContractCallValue = 0
    MaxContractSize          = 65536 // 64KB

    // TRC20 constants
    TRC20TransferMethodID    = "a9059cbb" // transfer(address,uint256)
    TRC20BalanceOfMethodID   = "70a08231" // balanceOf(address)
    TRC20ApproveMethodID     = "095ea7b3" // approve(address,uint256)
    TRC20AllowanceMethodID   = "dd62ed3e" // allowance(address,address)
    TRC20TotalSupplyMethodID = "18160ddd" // totalSupply()
    TRC20NameMethodID        = "06fdde03" // name()
    TRC20SymbolMethodID      = "95d89b41" // symbol()
    TRC20DecimalsMethodID    = "313ce567" // decimals()

    // Block constants
    BlockTimeMS = 3000 // 3 seconds per block

    // Permission constants
    OwnerPermissionID   = 0
    WitnessPermissionID = 1
    ActivePermissionID  = 2

    MaxResultSize = 64 // used for bandwidth estimation
)
```

### Variables

```go
var (
    // ErrInvalidAddress indicates an invalid address format or value
    ErrInvalidAddress = errors.New("invalid address: check format and ensure it's a valid TRON address")

    // ErrInvalidAmount indicates an invalid amount value
    ErrInvalidAmount = errors.New("invalid amount: must be positive and within valid range")

    // ErrInvalidContract indicates an invalid contract
    ErrInvalidContract = errors.New("invalid contract: check contract address and ABI")

    // ErrInvalidTransaction indicates an invalid transaction
    ErrInvalidTransaction = errors.New("invalid transaction: check transaction format and parameters")

    // ErrInsufficientBalance indicates insufficient balance for operation
    ErrInsufficientBalance = errors.New("insufficient balance: check account balance and required amount")

    // ErrInsufficientEnergy indicates insufficient energy for contract execution
    ErrInsufficientEnergy = errors.New("insufficient energy: freeze TRX for energy or wait for energy regeneration")

    // ErrInsufficientBandwidth indicates insufficient bandwidth for transaction
    ErrInsufficientBandwidth = errors.New("insufficient bandwidth: freeze TRX for bandwidth or wait for bandwidth regeneration")

    // ErrTransactionFailed indicates transaction execution failed
    ErrTransactionFailed = errors.New("transaction failed: check transaction details and account resources")

    // ErrContractExecutionFailed indicates contract execution failed
    ErrContractExecutionFailed = errors.New("contract execution failed: check contract state and parameters")

    // ErrNetworkError indicates a network-related error
    ErrNetworkError = errors.New("network error: check connection and node availability")

    // ErrTimeout indicates operation timeout
    ErrTimeout = errors.New("operation timeout: try again or increase timeout duration")

    // ErrNotFound indicates resource not found
    ErrNotFound = errors.New("not found: check resource identifier and network")

    // ErrAlreadyExists indicates resource already exists
    ErrAlreadyExists = errors.New("already exists: resource with this identifier already exists")

    // ErrPermissionDenied indicates insufficient permissions
    ErrPermissionDenied = errors.New("permission denied: check account permissions and authorization")

    // ErrInvalidParameter indicates invalid parameter value
    ErrInvalidParameter = errors.New("invalid parameter: check parameter value and format")
)
```

### Functions

#### NewAddress

```go
func NewAddress[T addressAllowed](v T) (*Address, error)
```

NewAddress creates an Address from a string, []byte, or base58 string.

This generic function attempts to parse the input in the following order:
1. As a Base58 TRON address (T-prefixed)
2. As a hex string (with or without 0x prefix)
3. As raw bytes

Supported input types:
- string: Base58 address, hex string
- []byte: Raw address bytes
- *Address: Returns the same address
- *eCommon.Address: Ethereum address (will be converted)
- [20]byte: Raw 20-byte address
- [21]byte: Raw 21-byte address with 0x41 prefix

Example:
```go
addr, err := types.NewAddress("TLCuBEbV6jp9432t4Xhg5E5j7v7vK4gkgX")
if err != nil {
    // handle error
}

addr2, err := types.NewAddress("0x41a614f803b6fd780986a42c78ec9c7f77e6ded13c")
if err != nil {
    // handle error
}
```

#### NewAddressFromBase58

```go
func NewAddressFromBase58(base58Addr string) (*Address, error)
```

NewAddressFromBase58 creates an Address from a Base58Check string. The string must be length 34, T-prefixed.

This function parses a Base58-encoded TRON address and validates its checksum. The address must be exactly 34 characters long and start with "T".

Example:
```go
addr, err := types.NewAddressFromBase58("TLCuBEbV6jp9432t4Xhg5E5j7v7vK4gkgX")
if err != nil {
    // handle error
}
fmt.Printf("Address: %s\n", addr.String())
```

#### NewAddressFromHex

```go
func NewAddressFromHex(hexAddr string) (*Address, error)
```

NewAddressFromHex creates an Address from a hex string. Supported forms:
- 0x41-prefixed 21-byte TRON hex
- 41-prefixed 21-byte TRON hex (without 0x)
- 20-byte hex (0x-optional) which will be promoted by adding 0x41 prefix

#### NewAddressFromBytes

```go
func NewAddressFromBytes(byteAddress []byte) (*Address, error)
```

NewAddressFromBytes creates an Address from bytes. Supported lengths:
- 21 bytes (0x41-prefixed TRON address)
- 20 bytes (EVM address), which will be promoted by adding 0x41 prefix

#### MustNewAddressFromBase58

```go
func MustNewAddressFromBase58(base58Addr string) *Address
```

MustNewAddressFromBase58 is a wrapper for NewAddressFromBase58 that panics if the address is invalid.

#### MustNewAddressFromHex

```go
func MustNewAddressFromHex(hexAddr string) *Address
```

MustNewAddressFromHex is a wrapper for NewAddressFromHex that panics if the address is invalid.

#### MustNewAddressFromBytes

```go
func MustNewAddressFromBytes(byteAddress []byte) *Address
```

MustNewAddressFromBytes is a wrapper for NewAddressFromBytes that panics if the address is invalid.

#### NewTronError

```go
func NewTronError(code int32, message string, cause error) *TronError
```

NewTronError creates a new TronError.

#### NewTransactionError

```go
func NewTransactionError(txID, message string, cause error) *TransactionError
```

NewTransactionError creates a new TransactionError.

#### NewContractError

```go
func NewContractError(contractAddress, method, message string, cause error) *ContractError
```

NewContractError creates a new ContractError.

### Address Methods

#### String

```go
func (a *Address) String() string
```

String returns the T prefixed 34 chars base58 representation.

This method implements the fmt.Stringer interface, returning the Base58 representation of the address which is the default string representation.

Example:
```go
addr, _ := types.NewAddressFromBase58("TLCuBEbV6jp9432t4Xhg5E5j7v7vK4gkgX")
fmt.Printf("Address: %s\n", addr.String()) // Prints: TLCuBEbV6jp9432t4Xhg5E5j7v7vK4gkgX
```

#### Base58

```go
func (a *Address) Base58() string
```

Base58 returns the T prefixed 34 chars base58 representation.

#### Bytes

```go
func (a *Address) Bytes() []byte
```

Bytes returns the raw bytes of the address (0x41 prefixed 21 bytes).

This method returns the raw byte representation of the address, which includes the 0x41 prefix followed by the 20-byte address hash.

Example:
```go
addr, _ := types.NewAddressFromBase58("TLCuBEbV6jp9432t4Xhg5E5j7v7vK4gkgX")
bytes := addr.Bytes() // Returns 21 bytes: [0x41, ...]
```

#### BytesEVM

```go
func (a *Address) BytesEVM() []byte
```

BytesEVM returns the raw bytes of the address (20 bytes without prefix).

#### Hex

```go
func (a *Address) Hex() string
```

Hex returns the address as 41-prefixed, 42-character hex string.

#### HexEVM

```go
func (a *Address) HexEVM() string
```

HexEVM returns the EVM-style 0x-prefixed, 40-character hex string.

#### IsValid

```go
func (a *Address) IsValid() bool
```

IsValid checks if the address is valid.

#### Equal

```go
func (a *Address) Equal(other *Address) bool
```

Equal checks if two addresses are equal.

#### EVMAddress

```go
func (a *Address) EVMAddress() eCommon.Address
```

EVMAddress converts the TRON address to an Ethereum compatible address. It panics if the address is nil.

### TronError Methods

#### Error

```go
func (e *TronError) Error() string
```

Error implements error.

#### Unwrap

```go
func (e *TronError) Unwrap() error
```

Unwrap returns the underlying cause.

### TransactionError Methods

#### Error

```go
func (e *TransactionError) Error() string
```

Error implements error.

#### Unwrap

```go
func (e *TransactionError) Unwrap() error
```

Unwrap returns the underlying cause.

### ContractError Methods

#### Error

```go
func (e *ContractError) Error() string
```

Error implements error.

#### Unwrap

```go
func (e *ContractError) Unwrap() error
```

Unwrap returns the underlying cause.

### Network Methods

#### GetNetwork

```go
func GetNetwork(name string) *Network
```

GetNetwork returns a predefined network by name.

### ResourceType Methods

#### String

```go
func (r ResourceType) String() string
```

String returns the string representation of ResourceType.

### ContractType Methods

#### String

```go
func (c ContractType) String() string
```

String returns the string representation of ContractType.

### TransactionStatus Methods

#### String

```go
func (s TransactionStatus) String() string
```

String returns the string representation of TransactionStatus.

---

## signer package

Package signer contains key management and transaction signing utilities, including HD wallet derivation and raw private key signing.

### Types

#### Signer

```go
type Signer interface {
    // Address returns the account's address
    Address() *types.Address

    // PublicKey returns the account's public key
    PublicKey() *ecdsa.PublicKey

    // Sign signs a transaction, supporting both *core.Transaction and *api.TransactionExtention types
    // It modifies the transaction in place by appending the signature
    Sign(tx any) error

    // SignMessageV2 signs a message using TIP-191 format (v2)
    SignMessageV2(message string) (string, error)
}
```

Signer defines the interface for signing Tron transactions and messages.

#### PrivateKeySigner

```go
type PrivateKeySigner struct {
    // contains filtered or unexported fields
}
```

PrivateKeySigner implements the Signer interface using a private key.

The PrivateKeySigner allows you to sign transactions and messages using a private key. It automatically derives the corresponding public key and address.

### Functions

#### NewPrivateKeySigner

```go
func NewPrivateKeySigner(hexPrivKey string) (*PrivateKeySigner, error)
```

NewPrivateKeySigner creates a new PrivateKeySigner from a hex private key.

This function creates a signer from a hexadecimal private key string. The private key can be provided with or without the "0x" prefix.

Example:
```go
signer, err := signer.NewPrivateKeySigner("0xYourPrivateKeyHere")
if err != nil {
    // handle error
}

// Get the address associated with this private key
address := signer.Address()
fmt.Printf("Address: %s\n", address.String())
```

#### NewPrivateKeySignerFromECDSA

```go
func NewPrivateKeySignerFromECDSA(privKey *ecdsa.PrivateKey) (*PrivateKeySigner, error)
```

NewPrivateKeySignerFromECDSA creates a new PrivateKeySigner from an ECDSA private key.

### PrivateKeySigner Methods

#### Address

```go
func (s *PrivateKeySigner) Address() *types.Address
```

Address returns the account's address.

This method returns the TRON address associated with the private key.

Example:
```go
signer, _ := signer.NewPrivateKeySigner("0xYourPrivateKeyHere")
address := signer.Address()
fmt.Printf("Address: %s\n", address.String())
```

#### PublicKey

```go
func (s *PrivateKeySigner) PublicKey() *ecdsa.PublicKey
```

PublicKey returns the account's public key.

#### PrivateKeyHex

```go
func (s *PrivateKeySigner) PrivateKeyHex() string
```

PrivateKeyHex returns the account's private key in hex format.

#### Sign

```go
func (s *PrivateKeySigner) Sign(tx any) error
```

Sign signs a transaction using the private key.

This method signs either a *core.Transaction or *api.TransactionExtention using the private key. The signature is appended to the transaction's Signature field.

Example:
```go
signer, _ := signer.NewPrivateKeySigner("0xYourPrivateKeyHere")
err := signer.Sign(transaction)
if err != nil {
    // handle error
}
```

#### SignMessageV2

```go
func (s *PrivateKeySigner) SignMessageV2(message string) (string, error)
```

SignMessageV2 signs a message using TIP-191 format (v2).

---

## eventdecoder package

Package eventdecoder maintains a compact registry of event signatures and helpers to decode logs into typed values using minimal ABI fragments.

Two usage modes are supported:
1. Register ABI sources at runtime (JSON or *core.SmartContract_ABI)
2. Use builtin signatures generated from common ecosystems (e.g., TRC20)

Given topics and data from a TransactionInfo_Log, DecodeLog will look up the first topic's signature, build a minimal ABI for the matched signature, and return a DecodedEvent. Unknown signatures fall back to a placeholder name instead of failing.

### Types

#### DecodedEvent

```go
type DecodedEvent struct {
    EventName  string                  `json:"eventName"`
    Parameters []DecodedEventParameter `json:"parameters"`
    Contract   string                  `json:"contract"`
}
```

DecodedEvent represents a decoded event.

#### DecodedEventParameter

```go
type DecodedEventParameter struct {
    Name    string `json:"name"`
    Type    string `json:"type"`
    Value   string `json:"value"`
    Indexed bool   `json:"indexed"`
}
```

DecodedEventParameter represents a decoded event parameter.

#### ParamDef

```go
type ParamDef struct {
    Type    string
    Indexed bool
    Name    string
}
```

ParamDef is a compact representation of an event parameter definition.

#### EventDef

```go
type EventDef struct {
    Name   string
    Inputs []ParamDef
}
```

EventDef is a compact representation of an event definition.

### Functions

#### RegisterABIJSON

```go
func RegisterABIJSON(abiJSON string) error
```

RegisterABIJSON registers all event entries from a JSON ABI string.

#### RegisterABIObject

```go
func RegisterABIObject(abi *core.SmartContract_ABI) error
```

RegisterABIObject registers all event entries from a SmartContract_ABI object.

#### RegisterABIEntries

```go
func RegisterABIEntries(entries []*core.SmartContract_ABI_Entry) error
```

RegisterABIEntries registers all event entries from the provided list (non-event entries are ignored).

#### DecodeEventSignature

```go
func DecodeEventSignature(sig []byte) (string, bool)
```

DecodeEventSignature returns the canonical event signature string for the given 4-byte signature if known. The boolean indicates whether the signature was found in the registry.

#### DecodeLog

```go
func DecodeLog(topics [][]byte, data []byte) (*DecodedEvent, error)
```

DecodeLog decodes a single log using the global 4-byte signature registry.

#### DecodeLogs

```go
func DecodeLogs(logs []*core.TransactionInfo_Log) ([]*DecodedEvent, error)
```

DecodeLogs decodes a slice of logs using the global 4-byte signature registry.

---

## utils package

Package utils houses ABI encode/decode logic, type parsing, and common helpers shared by higher-level packages. The ABIProcessor is the central type for converting between Go values and TRON/EVM ABI representations.

### Types

#### ABIProcessor

```go
type ABIProcessor struct {
    // contains filtered or unexported fields
}
```

ABIProcessor handles all smart contract ABI operations including encoding, decoding, parsing, and event processing.

#### DecodedInput

```go
type DecodedInput struct {
    Method     string                  `json:"method"`
    Parameters []DecodedInputParameter `json:"parameters"`
}
```

DecodedInput represents decoded input data.

#### DecodedInputParameter

```go
type DecodedInputParameter struct {
    Name  string      `json:"name"`
    Type  string      `json:"type"`
    Value interface{} `json:"value"`
}
```

DecodedInputParameter represents a decoded parameter.

### Functions

#### NewABIProcessor

```go
func NewABIProcessor(abi *core.SmartContract_ABI) *ABIProcessor
```

NewABIProcessor creates an ABIProcessor bound to the provided ABI. The processor exposes helpers to parse ABI JSON, encode inputs, decode outputs, and decode events.

#### ParseABI

```go
func (p *ABIProcessor) ParseABI(abi string) (*core.SmartContract_ABI, error)
```

ParseABI decodes a standard ABI JSON string into a *core.SmartContract_ABI.

#### GetMethodTypes

```go
func (p *ABIProcessor) GetMethodTypes(methodName string) ([]string, []string, error)
```

GetMethodTypes returns input and output type names for the given method.

#### GetConstructorTypes

```go
func (p *ABIProcessor) GetConstructorTypes(abi *core.SmartContract_ABI) ([]string, error)
```

GetConstructorTypes returns the constructor input type names.

#### EncodeMethod

```go
func (p *ABIProcessor) EncodeMethod(method string, paramTypes []string, params []interface{}) ([]byte, error)
```

EncodeMethod encodes a method call with parameters. For constructors, pass method="" to encode only parameters (no 4-byte method ID).

#### DecodeInputData

```go
func (p *ABIProcessor) DecodeInputData(data []byte, abi *core.SmartContract_ABI) (*DecodedInput, error)
```

DecodeInputData decodes call data into a DecodedInput using the provided ABI.

#### DecodeResult

```go
func (p *ABIProcessor) DecodeResult(data []byte, outputs []*core.SmartContract_ABI_Entry_Param) (interface{}, error)
```

DecodeResult decodes method return bytes. Behavior:
- no outputs: returns nil
- one output: returns the value directly
- many outputs: returns []interface{}