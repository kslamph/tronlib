# DeployContract Function Refactoring

## Overview

The `DeployContract` function in the `transaction` package has been refactored to provide a high-level interface for deploying smart contracts on the TRON network. The previous low-level implementation that required manually creating a `CreateSmartContract` message has been moved to the client layer.

## Changes Made

### 1. High-Level DeployContract Function

The new `DeployContract` function in `pkg/transaction/transaction.go` provides a simplified interface:

```go
func (tx *Transaction) DeployContract(
    ctx context.Context, 
    bytecode []byte, 
    abi []byte, 
    name string, 
    originEnergyLimit int64, 
    consumeUserResourcePercent int64, 
    constructorParams ...interface{}
) *Transaction
```

### 2. Parameters

- `ctx context.Context`: Context for the operation
- `bytecode []byte`: Compiled contract bytecode
- `abi []byte`: Contract ABI (JSON format)
- `name string`: Contract name
- `originEnergyLimit int64`: Energy limit for contract execution
- `consumeUserResourcePercent int64`: Percentage of user resources to consume (0-100)
- `constructorParams ...interface{}`: Optional constructor parameters

### 3. Features

- **Parameter Validation**: Validates all required parameters
- **Constructor Support**: Automatically handles constructor parameter encoding
- **ABI Decoding**: Automatically decodes ABI for proper contract creation
- **Error Handling**: Comprehensive error handling with descriptive messages
- **Fluent Interface**: Supports method chaining with other transaction methods

## Usage Examples

### Basic Contract Deployment (No Constructor)

```go
// Load contract files
bytecode, _ := loadContractBytecode("contract.bin")
abi, _ := loadContractABI("contract.abi")

// Deploy contract
tx := transaction.NewTransaction(client).
    SetOwner(owner.Address()).
    DeployContract(
        ctx,
        bytecode,           // contract bytecode
        abi,               // contract ABI
        "MyContract",      // contract name
        1000000,           // origin energy limit
        100,               // consume user resource percent
    ).
    SetFeelimit(150000000).
    Sign(owner).
    Broadcast()

receipt := tx.GetReceipt()
if receipt.Err != nil {
    log.Fatalf("Deployment failed: %v", receipt.Err)
}
fmt.Printf("Contract deployed! TX ID: %s\n", receipt.TxID)
```

### Contract Deployment with Constructor Parameters

```go
// Deploy token contract with constructor parameters
tx := transaction.NewTransaction(client).
    SetOwner(owner.Address()).
    DeployContract(
        ctx,
        bytecode,           // contract bytecode
        abi,               // contract ABI
        "MyToken",         // contract name
        1000000,           // origin energy limit
        100,               // consume user resource percent
        "MyToken",         // constructor param 1: token name
        "MTK",             // constructor param 2: token symbol
        uint64(1000000),   // constructor param 3: total supply
    ).
    SetFeelimit(150000000).
    Sign(owner).
    Broadcast()
```

## Migration from Old Implementation

### Before (Low-level approach)

```go
// Old way - required manual CreateSmartContract creation
createReq := &core.CreateSmartContract{
    OwnerAddress: owner.Address().Bytes(),
    NewContract: &core.SmartContract{
        Name:                       "MyContract",
        Bytecode:                   bytecode,
        Abi:                        contractABI,
        OriginAddress:              owner.Address().Bytes(),
        OriginEnergyLimit:          1000000,
        ConsumeUserResourcePercent: 100,
    },
}

tx := transaction.NewTransaction(client).
    SetOwner(owner.Address()).
    DeployContract(ctx, createReq).
    Sign(owner).
    Broadcast()
```

### After (High-level approach)

```go
// New way - simplified interface
tx := transaction.NewTransaction(client).
    SetOwner(owner.Address()).
    DeployContract(
        ctx,
        bytecode,      // contract bytecode
        abi,          // contract ABI
        "MyContract", // contract name
        1000000,      // origin energy limit
        100,          // consume user resource percent
    ).
    Sign(owner).
    Broadcast()
```

## Error Handling

The new function provides comprehensive error handling:

- **Empty bytecode**: "bytecode cannot be empty"
- **Empty ABI**: "abi cannot be empty"
- **Empty name**: "contract name cannot be empty"
- **Invalid energy limit**: "origin energy limit must be greater than 0"
- **Invalid resource percent**: "consume user resource percent must be between 0 and 100"
- **Missing owner**: "owner address must be set before deploying contract"
- **ABI decoding errors**: "failed to decode ABI"
- **Constructor encoding errors**: "failed to encode constructor parameters"

## Backward Compatibility

The low-level `CreateDeployContractTransaction` method remains available in the client package for advanced use cases that require direct control over the `CreateSmartContract` message structure.

## Example Implementation

See `examples/contract/deploy_contract_example.go` for a complete working example of how to use the new high-level `DeployContract` function. 