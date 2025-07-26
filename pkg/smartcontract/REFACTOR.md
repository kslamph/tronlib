# Smart Contract Package Refactor

This document describes the refactor of the smart contract package to move methods from Manager to Contract and add client dependency.

## Changes Made

### 1. Contract Struct Refactor

The `Contract` struct now includes a `*client.Client` field:

```go
type Contract struct {
    ABI     *core.SmartContract_ABI
    Address *types.Address
    Client  *client.Client

    // Utility instances for encoding/decoding
    encoder      *utils.ABIEncoder
    decoder      *utils.ABIDecoder
    eventDecoder *utils.EventDecoder
    parser       *utils.ABIParser
}
```

### 2. NewContract Signature Change

The `NewContract` function now has a cleaner signature with variadic ABI parameter:

```go
func NewContract(tronClient *client.Client, address *types.Address, abi ...any) (*Contract, error)
```

**Parameters:**
- `tronClient`: TRON client instance (required)
- `address`: Contract address as `*types.Address` (required)
- `abi`: Optional ABI parameter that can be:
  - **Omitted**: ABI will be retrieved from the network
  - **string**: ABI JSON string
  - **`*core.SmartContract_ABI`**: Parsed ABI object

### 3. New Contract Methods

#### TriggerSmartContract

```go
func (c *Contract) TriggerSmartContract(ctx context.Context, contract *Contract, owner *types.Address, callValue int64, method string, params ...interface{}) (*api.TransactionExtention, error)
```

This method:
- Triggers a smart contract method that modifies blockchain state
- Takes method name and parameters instead of raw data
- Automatically encodes the method call
- Returns a transaction extension for broadcasting

#### TriggerConstantContract

```go
func (c *Contract) TriggerConstantContract(ctx context.Context, contract *Contract, owner *types.Address, method string, params ...interface{}) (interface{}, error)
```

This method:
- Triggers a constant (read-only) smart contract method
- Takes method name and parameters instead of raw data
- Automatically encodes the method call and decodes the result
- Returns the decoded result as `interface{}`

### 4. Removed Methods from Manager

The following methods were removed from the `Manager` struct as they are now part of the `Contract`:
- `TriggerContract`
- `TriggerConstantContract`

### 5. Updated Method Signatures

The new method signatures match the requirements:
- `TriggerSmartContract`: `(ctx context.Context, contract *smartcontract.Contract, owner *types.Address, callValue int64, method string, params ...interface{}) (*api.TransactionExtention, error)`
- `TriggerConstantContract`: `(ctx context.Context, contract *smartcontract.Contract, owner *types.Address, method string, params ...interface{}) (interface{}, error)`

## Usage Examples

### Creating a Contract

```go
client := client.NewClient(config)
address, _ := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")

// Option 1: With ABI string
contract1, err := smartcontract.NewContract(client, address, abiString)

// Option 2: With parsed ABI
parsedABI, _ := utils.NewABIParser().ParseABI(abiString)
contract2, err := smartcontract.NewContract(client, address, parsedABI)

// Option 3: Retrieve ABI from network
contract3, err := smartcontract.NewContract(client, address) // No ABI parameter
```

### Calling a Constant Method

```go
// Call balanceOf method
result, err := contract.TriggerConstantContract(
    ctx,
    contract,
    ownerAddress,
    "balanceOf",
    "TGzz8gjYiYRqpfmDwnLxfgPuLVNmpCswVp", // address parameter
)
```

### Calling a State-Changing Method

```go
// Call transfer method
tx, err := contract.TriggerSmartContract(
    ctx,
    contract,
    ownerAddress,
    0, // callValue
    "transfer",
    "TGzz8gjYiYRqpfmDwnLxfgPuLVNmpCswVp", // to address
    big.NewInt(1000000),                    // amount
)
```

## Breaking Changes

This refactor introduces breaking changes:

1. **NewContract signature changed** - now takes `(client, address, ...abi)` instead of `(address, abi, client)`
2. **Address parameter type changed** - now requires `*types.Address` instead of `any`
3. **ABI parameter is now variadic** - can be omitted to retrieve from network
4. **Manager methods removed** - TriggerContract and TriggerConstantContract are no longer available on Manager
5. **Method signatures changed** - the new Contract methods have different signatures than the old Manager methods

## Migration Guide

### Before (Old API)
```go
manager := smartcontract.NewManager(client)
tx, err := manager.TriggerContract(ctx, ownerAddr, contractAddr, data, callValue, 0, 0)
result, err := manager.TriggerConstantContract(ctx, ownerAddr, contractAddr, data, 0)
```

### After (New API)
```go
address, _ := types.NewAddress(contractAddr)
contract, err := smartcontract.NewContract(client, address, abi) // or omit abi
tx, err := contract.TriggerSmartContract(ctx, contract, owner, callValue, "methodName", param1, param2)
result, err := contract.TriggerConstantContract(ctx, contract, owner, "methodName", param1, param2)
```

## Benefits

1. **Better encapsulation** - Contract methods are now part of the Contract struct
2. **Automatic encoding/decoding** - No need to manually encode method calls or decode results
3. **Type safety** - Method parameters are type-checked
4. **Cleaner API** - More intuitive method signatures with proper parameter order
5. **Result decoding** - TriggerConstantContract automatically decodes results
6. **Flexible ABI handling** - Can provide ABI or retrieve from network automatically
7. **Stronger typing** - Address parameter is now properly typed as `*types.Address`