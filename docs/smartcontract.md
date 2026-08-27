# 📜 Smart Contract Package Reference

The `smartcontract` package provides comprehensive tools for deploying, managing, and interacting with smart contracts on the TRON blockchain. It offers both high-level convenience methods and low-level control for advanced use cases.

## 📚 Learning Path

This document is part of the TronLib learning path:
1. [Quick Start Guide](quickstart.md) - Basic usage
2. [Architecture Overview](architecture.md) - Understanding the design
3. **Smart Contract Package Reference** (this document) - Detailed contract operations
4. [Other Package Documentation](../README.md#package-references) - Additional functionality
5. [API Reference](API_REFERENCE.md) - Complete function documentation

## 📋 Overview

The smartcontract package features:
- **Contract Deployment** - Deploy contracts with constructor parameters
- **Contract Interaction** - Call contract methods with type safety
- **ABI Management** - Automatic ABI parsing and method encoding
- **Energy Estimation** - Predict energy costs before execution
- **Event Handling** - Process contract events and logs
- **Administrative Functions** - Update contract settings and permissions

## 🏗️ Core Components

### Manager vs Instance

The package provides two main types:

1. **Manager** - Package-level operations (deployment, admin functions, energy estimation)
2. **Instance** - Contract-specific operations (method calls, queries, encoding)

```go
// Manager for deployment and admin operations
type Manager struct {
    conn lowlevel.ConnProvider
}

// Instance for contract-specific interactions
type Instance struct {
    conn    lowlevel.ConnProvider
    address *types.Address
    abi     *core.SmartContract_ABI
}
```

## 🚀 Getting Started

### Basic Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    "math/big"

    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/smartcontract"
    "github.com/kslamph/tronlib/pkg/types"
)

func main() {
    // Connect to TRON network
    cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
    if err != nil {
        log.Fatal(err)
    }
    defer cli.Close()

    // Create smart contract manager via the client facade
    mgr := cli.SmartContract()

    // Or create an instance for an existing contract
    contractAddr, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
    abiJSON := `[{"type": "function", "name": "getValue", ...}]`
    instance, err := smartcontract.NewInstance(cli, contractAddr, abiJSON)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    // Ready to interact with contracts...
}
```

## 🏗️ Contract Deployment

### Simple Contract Deployment

```go
// Deploy a basic contract
func DeploySimpleContract(ctx context.Context, scMgr *smartcontract.Manager) error {
    owner, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

    // Contract details
    contractName := "SimpleStorage"
    abiJSON := `[{
        "type": "constructor",
        "inputs": [{"name": "initialValue", "type": "uint256"}]
    }, {
        "type": "function",
        "name": "setValue",
        "inputs": [{"name": "value", "type": "uint256"}],
        "outputs": []
    }]`

    bytecode, _ := hex.DecodeString("608060405234801561001057600080fd5b50...")

    // Constructor parameters
    initialValue := big.NewInt(42)

    // Deploy contract — returns (*api.TransactionExtention, error).
    // Sign and broadcast via cli.SignAndBroadcast to get the contract address
    // from the receipt.
    ext, err := scMgr.Deploy(
        ctx,
        owner,
        contractName,
        abiJSON,
        bytecode,
        0,               // TRX value to send
        100,             // Consume user resource percent
        30000,           // Origin energy limit
        initialValue,    // Constructor parameters...
    )
    if err != nil {
        return fmt.Errorf("deployment failed: %w", err)
    }

    // Sign and broadcast to deploy on-chain
    opts := client.DefaultBroadcastOptions()
    opts.WaitForReceipt = true
    result, err := cli.SignAndBroadcast(ctx, ext, opts, deploySigner)
    if err != nil {
        return fmt.Errorf("broadcast failed: %w", err)
    }

    fmt.Printf("✅ Contract deployed! Transaction: %s\n", result.TxID)
    return nil
}
```

### Advanced Deployment Options

```go
// Deploy with custom energy and bandwidth settings
func DeployWithCustomSettings(ctx context.Context, scMgr *smartcontract.Manager) error {
    owner, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

    // Estimate energy before deploying
    estimatedEnergy, err := scMgr.EstimateEnergy(ctx, owner, contractAddr, callData, 0)
    if err != nil {
        return fmt.Errorf("energy estimation failed: %w", err)
    }

    fmt.Printf("Estimated energy needed: %d\n", estimatedEnergy)

    // Deploy with specific settings
    ext, err := scMgr.Deploy(
        ctx,
        owner,
        contractName,
        abiJSON,
        bytecode,
        1000000,  // Send 1 TRX to contract
        200,      // Higher consume user resource percent
        50000,    // Higher origin energy limit
        constructorParams...,
    )
    if err != nil {
        return fmt.Errorf("deployment failed: %w", err)
    }

    // Verify deployment by fetching the contract
    contract, err := scMgr.GetContract(ctx, contractAddr)
    if err != nil {
        return fmt.Errorf("failed to verify deployment: %w", err)
    }

    fmt.Printf("✅ Contract verified: %s\n", contract.GetName())
    return nil
}
```

## 🔧 Contract Interaction

### Creating a Contract Instance

```go
// Create instance for an existing contract
contractAddr, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

// With ABI JSON string
abiJSON := `[{"type": "function", "name": "getValue", ...}]`
instance, err := smartcontract.NewInstance(cli, contractAddr, abiJSON)
if err != nil {
    log.Fatal(err)
}
```

### Calling Contract Methods

#### View Functions (Read-Only)

```go
// Call view function that returns data
caller, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

// Simple getter function — returns (interface{}, error)
result, err := instance.Call(ctx, caller, "getValue")
if err != nil {
    log.Fatalf("Call failed: %v", err)
}

// result is an interface{}; type-assert as needed:
if val, ok := result.(*big.Int); ok {
    fmt.Printf("Current value: %s\n", val.String())
}

// Function with parameters
owner, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
result, err = instance.Call(ctx, caller, "balanceOf", owner)
if err != nil {
    log.Fatalf("Call failed: %v", err)
}

if balance, ok := result.(*big.Int); ok {
    fmt.Printf("Balance: %s\n", balance.String())
}
```

#### State-Changing Functions

```go
// Call function that modifies contract state
caller, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
newValue := big.NewInt(123)

// Build transaction — returns (*api.TransactionExtention, error)
tx, err := instance.Invoke(ctx, caller, 0, "setValue", newValue)
if err != nil {
    log.Fatalf("Failed to build invoke transaction: %v", err)
}

// Sign and broadcast
signer, _ := signer.NewPrivateKeySigner("your-private-key")
opts := client.DefaultBroadcastOptions()
opts.FeeLimit = 50_000_000 // 50 TRX max

result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
if err != nil {
    log.Fatalf("Transaction failed: %v", err)
}

fmt.Printf("✅ setValue transaction successful: %s\n", result.TxID)
```

#### Payable Functions

```go
// Call function that requires TRX payment
recipient, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
trxAmount := int64(5_000_000) // 5 TRX

// Build transaction with TRX value
tx, err := instance.Invoke(ctx, caller, trxAmount, "donate", recipient)
if err != nil {
    log.Fatalf("Failed to build payable transaction: %v", err)
}

// Sign and broadcast with higher fee limit
opts.FeeLimit = 100_000_000 // 100 TRX max for payable functions
result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
if err != nil {
    log.Fatalf("Payable transaction failed: %v", err)
}

fmt.Printf("💰 Donation successful: %s\n", result.TxID)
```

### Batch Contract Calls

```go
// Execute multiple contract calls efficiently
type ContractCall struct {
    Method string
    Params []interface{}
    Value  int64
}

func ExecuteBatchCalls(ctx context.Context, cli *client.Client, instance *smartcontract.Instance, caller *types.Address, calls []ContractCall, s signer.Signer) error {
    var transactions []*api.TransactionExtention

    // Build all transactions
    for i, call := range calls {
        tx, err := instance.Invoke(ctx, caller, call.Value, call.Method, call.Params...)
        if err != nil {
            return fmt.Errorf("failed to build call %d (%s): %w", i, call.Method, err)
        }
        transactions = append(transactions, tx)
    }

    // Execute all transactions
    opts := client.DefaultBroadcastOptions()
    opts.FeeLimit = 50_000_000
    for i, tx := range transactions {
        result, err := cli.SignAndBroadcast(ctx, tx, opts, s)
        if err != nil {
            return fmt.Errorf("failed to execute call %d: %w", i, err)
        }
        fmt.Printf("Call %d (%s): %s\n", i, calls[i].Method, result.TxID)
    }

    return nil
}

// Usage
calls := []ContractCall{
    {"setValue", []interface{}{big.NewInt(100)}, 0},
    {"approve", []interface{}{spender, big.NewInt(1000)}, 0},
    {"transfer", []interface{}{recipient, big.NewInt(50)}, 0},
}

err := ExecuteBatchCalls(ctx, cli, instance, caller, calls, signer)
```

## 📊 Contract Information

### Getting Contract Details

```go
// Get contract information from the network
contractAddr, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

contract, err := mgr.GetContract(ctx, contractAddr)
if err != nil {
    log.Fatalf("Failed to get contract: %v", err)
}

fmt.Printf("Contract Details:\n")
fmt.Printf("  Name: %s\n", contract.GetName())
fmt.Printf("  Creator: %s\n", hex.EncodeToString(contract.GetOriginAddress()))
fmt.Printf("  Creation Time: %d\n", contract.GetCreateTime())

// Get detailed contract information
contractInfo, err := mgr.GetContractInfo(ctx, contractAddr)
if err != nil {
    log.Fatalf("Failed to get contract info: %v", err)
}

fmt.Printf("Contract Info:\n")
fmt.Printf("  Runtime: %x\n", contractInfo.GetRuntimeCode()[:50]) // First 50 bytes
fmt.Printf("  ABI: %s\n", contractInfo.GetAbi().String()[:100])   // First 100 chars
```

## ⚡ Energy Management

### Energy Estimation

Energy estimation is a **Manager** method (not an Instance method):

```go
import (
    "fmt"
    "log"
)

// Estimate energy cost before execution
caller, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

// Manager.EstimateEnergy(ctx, owner, contractAddr, callData, callValue)
// CallData is the ABI-encoded method selector + params.
estimatedEnergy, err := scMgr.EstimateEnergy(ctx, caller, contractAddr, encodedData, 0)
if err != nil {
    log.Printf("Energy estimation failed: %v", err)
} else {
    fmt.Printf("Estimated energy: %d\n", estimatedEnergy)
}
```

### Optimizing Energy Usage

```go
func OptimizedContractCall(ctx context.Context, cli *client.Client, scMgr *smartcontract.Manager, instance *smartcontract.Instance, caller *types.Address, method string, signer signer.Signer, params ...interface{}) error {
    // First, estimate energy
    estimatedEnergy, err := scMgr.EstimateEnergy(ctx, caller, contractAddr, encodedData, 0)
    if err != nil {
        return fmt.Errorf("energy estimation failed: %w", err)
    }

    // Check account resources
    acctMgr := cli.Account()
    account, err := acctMgr.GetAccount(ctx, caller)
    if err != nil {
        return fmt.Errorf("failed to get account info: %w", err)
    }

    // Build transaction with appropriate fee limit
    tx, err := instance.Invoke(ctx, caller, 0, method, params...)
    if err != nil {
        return fmt.Errorf("failed to build transaction: %w", err)
    }

    // Set fee limit based on estimation
    opts := client.DefaultBroadcastOptions()
    opts.FeeLimit = estimatedEnergy * 420 * 2 // 2x safety margin
    opts.WaitForReceipt = true

    result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
    if err != nil {
        return fmt.Errorf("transaction failed: %w", err)
    }

    fmt.Printf("✅ Success! Energy used: %d (estimated: %d)\n",
        result.EnergyUsage, estimatedEnergy)

    return nil
}
```

## 🎭 Event Processing

### Decoding Contract Events

```go
import (
    "github.com/kslamph/tronlib/pkg/eventdecoder"
)

// Process events from a broadcast result
func ProcessContractEvents(result *client.BroadcastResult, contractAddr *types.Address) {
    if len(result.Logs) == 0 {
        fmt.Println("No events emitted")
        return
    }

    fmt.Printf("Processing %d events:\n", len(result.Logs))

    for i, log := range result.Logs {
        // Check if event is from our contract
        logAddr := types.MustNewAddressFromBytes(log.GetAddress())
        if !logAddr.Equal(contractAddr) {
            continue
        }

        // Decode event
        event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
        if err != nil {
            fmt.Printf("  [%d] Failed to decode: %v\n", i, err)
            continue
        }

        fmt.Printf("  [%d] %s:\n", i, event.EventName)
        for _, param := range event.Parameters {
            // Note: param.Value is a string (ABI-encoded), not interface{}
            fmt.Printf("      %s (%s): %s\n", param.Name, param.Type, param.Value)
        }
    }
}
```

### Event Filtering

```go
// Filter specific events from transaction logs
func FilterTransferEvents(logs []*core.TransactionInfo_Log, tokenAddr *types.Address) []TransferEvent {
    var transfers []TransferEvent

    // Transfer event signature
    transferSig := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

    for _, log := range logs {
        // Check contract address
        logAddr := types.MustNewAddressFromBytes(log.GetAddress())
        if !logAddr.Equal(tokenAddr) {
            continue
        }

        topics := log.GetTopics()
        if len(topics) < 3 {
            continue
        }

        // Check event signature
        if !bytes.Equal(topics[0], transferSig.Bytes()) {
            continue
        }

        // Decode Transfer event
        from := types.MustNewAddressFromBytes(topics[1][12:]) // Last 20 bytes
        to := types.MustNewAddressFromBytes(topics[2][12:])   // Last 20 bytes
        amount := new(big.Int).SetBytes(log.GetData())

        transfers = append(transfers, TransferEvent{
            From:   from,
            To:     to,
            Amount: amount,
        })
    }

    return transfers
}

type TransferEvent struct {
    From   *types.Address
    To     *types.Address
    Amount *big.Int
}
```

## 🔧 Administrative Functions

### Contract Updates

```go
// Update contract settings (requires owner permissions)
func UpdateContractSettings(ctx context.Context, cli *client.Client, scMgr *smartcontract.Manager, contractAddr *types.Address, owner *types.Address, s signer.Signer) error {
    // Update consume user resource percent
    newPercent := int64(50)

    ext, err := scMgr.UpdateSetting(ctx, owner, contractAddr, newPercent)
    if err != nil {
        return fmt.Errorf("failed to update settings: %w", err)
    }

    opts := client.DefaultBroadcastOptions()
    result, err := cli.SignAndBroadcast(ctx, ext, opts, s)
    if err != nil {
        return fmt.Errorf("failed to broadcast settings update: %w", err)
    }

    fmt.Printf("✅ Contract settings updated: %s\n", result.TxID)

    // Update energy limit
    newEnergyLimit := int64(10_000_000)

    ext, err = scMgr.UpdateEnergyLimit(ctx, owner, contractAddr, newEnergyLimit)
    if err != nil {
        return fmt.Errorf("failed to update energy limit: %w", err)
    }

    result, err = cli.SignAndBroadcast(ctx, ext, opts, s)
    if err != nil {
        return fmt.Errorf("failed to broadcast energy limit update: %w", err)
    }

    fmt.Printf("✅ Energy limit updated: %s\n", result.TxID)

    return nil
}
```

### Contract ABI Management

```go
// Clear contract ABI (requires owner permissions)
func ClearContractABI(ctx context.Context, cli *client.Client, scMgr *smartcontract.Manager, contractAddr *types.Address, owner *types.Address, s signer.Signer) error {
    ext, err := scMgr.ClearContractABI(ctx, owner, contractAddr)
    if err != nil {
        return fmt.Errorf("failed to clear ABI: %w", err)
    }

    opts := client.DefaultBroadcastOptions()
    result, err := cli.SignAndBroadcast(ctx, ext, opts, s)
    if err != nil {
        return fmt.Errorf("failed to broadcast: %w", err)
    }

    fmt.Printf("✅ Contract ABI cleared: %s\n", result.TxID)
    return nil
}
```

## 🎯 Advanced Patterns

### Contract Factory Pattern

```go
type ContractFactory struct {
    mgr          *smartcontract.Manager
    abiJSON      string
    bytecode     []byte
    defaultOwner *types.Address
}

func NewContractFactory(scMgr *smartcontract.Manager, abiJSON string, bytecode []byte, defaultOwner *types.Address) *ContractFactory {
    return &ContractFactory{
        mgr:          scMgr,
        abiJSON:      abiJSON,
        bytecode:     bytecode,
        defaultOwner: defaultOwner,
    }
}

func (f *ContractFactory) DeployToken(ctx context.Context, name, symbol string, decimals uint8, supply *big.Int) (*api.TransactionExtention, error) {
    return f.mgr.Deploy(
        ctx,
        f.defaultOwner,
        fmt.Sprintf("Token_%s", symbol),
        f.abiJSON,
        f.bytecode,
        0,
        100,
        30000,
        name, symbol, decimals, supply,
    )
}
```

### Multi-Contract Manager

```go
type MultiContractManager struct {
    client    lowlevel.ConnProvider
    contracts map[string]*smartcontract.Instance
    mutex     sync.RWMutex
}

func NewMultiContractManager(cp lowlevel.ConnProvider) *MultiContractManager {
    return &MultiContractManager{
        client:    cp,
        contracts: make(map[string]*smartcontract.Instance),
    }
}

func (m *MultiContractManager) AddContract(name, address, abi string) error {
    addr, err := types.NewAddress(address)
    if err != nil {
        return err
    }

    instance, err := smartcontract.NewInstance(m.client, addr, abi)
    if err != nil {
        return err
    }

    m.mutex.Lock()
    m.contracts[name] = instance
    m.mutex.Unlock()

    return nil
}

func (m *MultiContractManager) GetContract(name string) (*smartcontract.Instance, bool) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()

    instance, exists := m.contracts[name]
    return instance, exists
}
```

## 🚨 Error Handling

### Comprehensive Error Handling

```go
func SafeContractCall(ctx context.Context, cli *client.Client, instance *smartcontract.Instance, caller *types.Address, method string, signer signer.Signer, params ...interface{}) error {
    // Build transaction
    tx, err := instance.Invoke(ctx, caller, 0, method, params...)
    if err != nil {
        if strings.Contains(err.Error(), "method not found") {
            return fmt.Errorf("method %s not found in contract ABI", method)
        }
        if strings.Contains(err.Error(), "invalid parameter") {
            return fmt.Errorf("invalid parameters for method %s: %w", method, err)
        }
        return fmt.Errorf("failed to build transaction: %w", err)
    }

    // Sign and broadcast
    opts := client.DefaultBroadcastOptions()
    opts.FeeLimit = 50_000_000
    opts.WaitForReceipt = true
    result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
    if err != nil {
        if strings.Contains(err.Error(), "REVERT") {
            return fmt.Errorf("contract execution reverted: %w", err)
        }
        if strings.Contains(err.Error(), "OUT_OF_ENERGY") {
            return fmt.Errorf("insufficient energy for execution: %w", err)
        }
        return fmt.Errorf("transaction failed: %w", err)
    }

    if !result.Success {
        return fmt.Errorf("contract call failed: %s", result.Message)
    }

    fmt.Printf("✅ Contract call successful: %s\n", result.TxID)
    return nil
}
```

## 🧪 Testing

### Unit Testing Contract Interactions

```go
func TestContractDeployment(t *testing.T) {
    mockCP := &MockConnProvider{}
    mgr := smartcontract.NewManager(mockCP)

    owner := types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

    // Test deployment
    ext, err := mgr.Deploy(
        context.Background(),
        owner,
        "TestContract",
        testABI,
        testBytecode,
        0,
        100,
        30000,
    )

    require.NoError(t, err)
    require.NotNil(t, ext)
}

func TestContractMethodCall(t *testing.T) {
    mockCP := &MockConnProvider{}
    contractAddr := types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

    instance, err := smartcontract.NewInstance(mockCP, contractAddr, testABI)
    require.NoError(t, err)

    caller := types.MustNewAddressFromBase58("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")

    // Test method call
    tx, err := instance.Invoke(context.Background(), caller, 0, "setValue", big.NewInt(42))
    require.NoError(t, err)
    require.NotNil(t, tx)
}
```

The smartcontract package provides powerful tools for all your contract deployment and interaction needs. Use these patterns to build sophisticated decentralized applications on TRON! 🚀
