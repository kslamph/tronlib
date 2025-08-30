# 📜 Smart Contract Package Reference

The `smartcontract` package provides comprehensive tools for deploying, managing, and interacting with smart contracts on the TRON blockchain. It offers both high-level convenience methods and low-level control for advanced use cases.

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

The package provides two main interfaces:

1. **Manager** - Package-level operations (deployment, admin functions)
2. **Instance** - Contract-specific operations (method calls, queries)

```go
// Manager for deployment and admin operations
type Manager struct {
    client Client
}

// Instance for contract-specific interactions
type Instance struct {
    client  Client
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

    // Create smart contract manager
    mgr := smartcontract.NewManager(cli)
    
    // Or create an instance for existing contract
    contractAddr, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
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
func DeploySimpleContract(ctx context.Context, mgr *smartcontract.Manager) error {
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
    
    bytecode := "608060405234801561001057600080fd5b50..." // Contract bytecode
    
    // Constructor parameters
    initialValue := big.NewInt(42)
    
    // Deploy contract
    contractAddr, txid, err := mgr.Deploy(
        ctx,
        owner,           // Contract owner
        contractName,    // Contract name
        abiJSON,         // Contract ABI
        bytecode,        // Contract bytecode
        0,               // TRX value to send
        100,             // Fee limit percentage
        30000,           // Consume user resource percentage  
        initialValue,    // Constructor parameters...
    )
    if err != nil {
        return fmt.Errorf("deployment failed: %w", err)
    }

    fmt.Printf("✅ Contract deployed!\n")
    fmt.Printf("Address: %s\n", contractAddr)
    fmt.Printf("Transaction: %s\n", txid)
    
    return nil
}
```

### Token Contract Deployment

```go
// Deploy a TRC20 token contract
func DeployTRC20Token(ctx context.Context, mgr *smartcontract.Manager, owner *types.Address) (*types.Address, error) {
    // TRC20 constructor parameters
    tokenName := "MyToken"
    tokenSymbol := "MTK"
    decimals := uint8(18)
    totalSupply := new(big.Int).Mul(big.NewInt(1000000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) // 1M tokens

    contractAddr, txid, err := mgr.Deploy(
        ctx,
        owner,
        "TRC20Token",
        trc20ABI,      // Standard TRC20 ABI
        trc20Bytecode, // Compiled TRC20 bytecode
        0,             // No TRX value
        100,           // Fee limit percentage
        30000,         // Consume user resource percentage
        tokenName,     // Constructor parameters
        tokenSymbol,
        decimals,
        totalSupply,
    )
    if err != nil {
        return nil, fmt.Errorf("TRC20 deployment failed: %w", err)
    }

    fmt.Printf("🪙 TRC20 Token deployed!\n")
    fmt.Printf("Name: %s (%s)\n", tokenName, tokenSymbol)
    fmt.Printf("Address: %s\n", contractAddr)
    fmt.Printf("Total Supply: %s\n", totalSupply.String())
    fmt.Printf("Transaction: %s\n", txid)

    return contractAddr, nil
}
```

### Advanced Deployment Options

```go
// Deploy with custom energy and bandwidth settings
func DeployWithCustomSettings(ctx context.Context, mgr *smartcontract.Manager) error {
    owner, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

    // Pre-calculate deployment cost
    estimatedEnergy, err := mgr.EstimateEnergy(ctx, owner, contractName, abiJSON, bytecode, 0, constructorParams...)
    if err != nil {
        return fmt.Errorf("energy estimation failed: %w", err)
    }

    fmt.Printf("Estimated energy needed: %d\n", estimatedEnergy)

    // Deploy with specific settings
    contractAddr, txid, err := mgr.Deploy(
        ctx,
        owner,
        contractName,
        abiJSON,
        bytecode,
        1000000,  // Send 1 TRX to contract
        200,      // Higher fee limit percentage
        50000,    // Higher consume user resource percentage
        constructorParams...,
    )
    if err != nil {
        return fmt.Errorf("deployment failed: %w", err)
    }

    // Verify deployment
    contract, err := mgr.GetContract(ctx, contractAddr)
    if err != nil {
        return fmt.Errorf("failed to verify deployment: %w", err)
    }

    fmt.Printf("✅ Contract verified: %s\n", contract.GetName())
    return nil
}
```

## 🔧 Contract Interaction

### Creating Contract Instance

```go
// Create instance for existing contract
contractAddr, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

// Option 1: With ABI JSON string
abiJSON := `[{"type": "function", "name": "getValue", ...}]`
instance, err := smartcontract.NewInstance(cli, contractAddr, abiJSON)
if err != nil {
    log.Fatal(err)
}

// Option 2: Fetch ABI from network (if available)
instance, err := smartcontract.NewInstanceFromNetwork(cli, contractAddr)
if err != nil {
    log.Fatal(err)
}
```

### Calling Contract Methods

#### View Functions (Read-Only)

```go
// Call view function that returns data
caller, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

// Simple getter function
result, err := instance.Call(ctx, caller, "getValue")
if err != nil {
    log.Fatalf("Call failed: %v", err)
}

// Extract return value
if len(result) > 0 {
    value := new(big.Int).SetBytes(result[0])
    fmt.Printf("Current value: %s\n", value.String())
}

// Function with parameters
owner, _ := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
result, err = instance.Call(ctx, caller, "balanceOf", owner)
if err != nil {
    log.Fatalf("Call failed: %v", err)
}

if len(result) > 0 {
    balance := new(big.Int).SetBytes(result[0])
    fmt.Printf("Balance: %s\n", balance.String())
}
```

#### State-Changing Functions

```go
// Call function that modifies contract state
caller, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
newValue := big.NewInt(123)

// Build transaction
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

func ExecuteBatchCalls(ctx context.Context, instance *smartcontract.Instance, caller *types.Address, calls []ContractCall) error {
    var transactions []*types.TransactionExtension

    // Build all transactions
    for i, call := range calls {
        tx, err := instance.Invoke(ctx, caller, call.Value, call.Method, call.Params...)
        if err != nil {
            return fmt.Errorf("failed to build call %d (%s): %w", i, call.Method, err)
        }
        transactions = append(transactions, tx)
    }

    // Execute all transactions
    for i, tx := range transactions {
        result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
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

err := ExecuteBatchCalls(ctx, instance, caller, calls)
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

### Contract Resource Usage

```go
// Check contract resource usage
resourceUsage, err := mgr.GetContractResourceUsage(ctx, contractAddr)
if err == nil {
    fmt.Printf("Resource Usage:\n")
    fmt.Printf("  Energy Used: %d\n", resourceUsage.EnergyUsed)
    fmt.Printf("  Energy Limit: %d\n", resourceUsage.EnergyLimit)
    fmt.Printf("  Origin Energy Limit: %d\n", resourceUsage.OriginEnergyLimit)
}
```

## ⚡ Energy Management

### Energy Estimation

```go
// Estimate energy cost before execution
caller, _ := types.NewAddress("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")

// Estimate for method call
estimatedEnergy, err := instance.EstimateEnergy(ctx, caller, "setValue", big.NewInt(42))
if err != nil {
    log.Printf("Energy estimation failed: %v", err)
} else {
    fmt.Printf("Estimated energy: %d\n", estimatedEnergy)
    
    // Calculate cost in TRX (approximate)
    energyPriceInSun := int64(420) // Current energy price
    costInSun := estimatedEnergy * energyPriceInSun
    costInTRX := float64(costInSun) / 1_000_000
    
    fmt.Printf("Estimated cost: %.6f TRX\n", costInTRX)
}
```

### Optimizing Energy Usage

```go
// Optimize contract calls for energy efficiency
func OptimizedContractCall(ctx context.Context, instance *smartcontract.Instance, caller *types.Address, method string, params ...interface{}) error {
    // First, estimate energy
    estimatedEnergy, err := instance.EstimateEnergy(ctx, caller, method, params...)
    if err != nil {
        return fmt.Errorf("energy estimation failed: %w", err)
    }

    // Check if caller has enough energy
    account, err := cli.GetAccount(ctx, caller)
    if err != nil {
        return fmt.Errorf("failed to get account info: %w", err)
    }

    availableEnergy := account.GetEnergyRemaining()
    if availableEnergy < estimatedEnergy {
        fmt.Printf("⚠️  Insufficient energy: have %d, need %d\n", availableEnergy, estimatedEnergy)
        fmt.Println("Consider freezing more TRX for energy")
    }

    // Build transaction with appropriate fee limit
    tx, err := instance.Invoke(ctx, caller, 0, method, params...)
    if err != nil {
        return fmt.Errorf("failed to build transaction: %w", err)
    }

    // Set fee limit based on estimation
    opts := client.DefaultBroadcastOptions()
    opts.FeeLimit = estimatedEnergy * 420 * 2 // 2x safety margin

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
// Process events from contract transaction
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
            continue // Skip events from other contracts
        }

        // Decode event using eventdecoder package
        event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
        if err != nil {
            fmt.Printf("  [%d] Failed to decode: %v\n", i, err)
            continue
        }

        fmt.Printf("  [%d] %s:\n", i, event.EventName)
        for _, param := range event.Parameters {
            fmt.Printf("      %s: %v\n", param.Name, param.Value)
        }
    }
}

// Usage after contract transaction
result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
if err == nil {
    ProcessContractEvents(result, contractAddr)
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
func UpdateContractSettings(ctx context.Context, mgr *smartcontract.Manager, contractAddr *types.Address, owner *types.Address) error {
    // Update consume user resource percent
    newPercent := int64(50)
    
    txid, err := mgr.UpdateSetting(ctx, owner, contractAddr, newPercent)
    if err != nil {
        return fmt.Errorf("failed to update settings: %w", err)
    }

    fmt.Printf("✅ Contract settings updated: %s\n", txid)

    // Update energy limit
    newEnergyLimit := int64(10_000_000)
    
    txid, err = mgr.UpdateEnergyLimit(ctx, owner, contractAddr, newEnergyLimit)
    if err != nil {
        return fmt.Errorf("failed to update energy limit: %w", err)
    }

    fmt.Printf("✅ Energy limit updated: %s\n", txid)
    
    return nil
}
```

### Contract ABI Management

```go
// Clear contract ABI (requires owner permissions)
func ClearContractABI(ctx context.Context, mgr *smartcontract.Manager, contractAddr *types.Address, owner *types.Address) error {
    txid, err := mgr.ClearContractABI(ctx, owner, contractAddr)
    if err != nil {
        return fmt.Errorf("failed to clear ABI: %w", err)
    }

    fmt.Printf("✅ Contract ABI cleared: %s\n", txid)
    return nil
}
```

## 🎯 Advanced Patterns

### Contract Factory Pattern

```go
// Factory for deploying multiple similar contracts
type ContractFactory struct {
    mgr         *smartcontract.Manager
    abiJSON     string
    bytecode    string
    defaultOwner *types.Address
}

func NewContractFactory(mgr *smartcontract.Manager, abiJSON, bytecode string, defaultOwner *types.Address) *ContractFactory {
    return &ContractFactory{
        mgr:         mgr,
        abiJSON:     abiJSON,
        bytecode:    bytecode,
        defaultOwner: defaultOwner,
    }
}

func (f *ContractFactory) DeployToken(ctx context.Context, name, symbol string, decimals uint8, supply *big.Int) (*types.Address, error) {
    contractAddr, _, err := f.mgr.Deploy(
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
    
    return contractAddr, err
}

// Deploy multiple tokens
tokens := []struct {
    name, symbol string
    decimals     uint8
    supply       *big.Int
}{
    {"Test Token A", "TTA", 18, big.NewInt(1000000)},
    {"Test Token B", "TTB", 6, big.NewInt(500000)},
}

factory := NewContractFactory(mgr, tokenABI, tokenBytecode, owner)

for _, token := range tokens {
    addr, err := factory.DeployToken(ctx, token.name, token.symbol, token.decimals, token.supply)
    if err != nil {
        log.Printf("Failed to deploy %s: %v", token.symbol, err)
        continue
    }
    fmt.Printf("✅ %s deployed at %s\n", token.symbol, addr)
}
```

### Multi-Contract Manager

```go
// Manage multiple contract instances
type MultiContractManager struct {
    client     *client.Client
    contracts  map[string]*smartcontract.Instance
    mutex      sync.RWMutex
}

func NewMultiContractManager(cli *client.Client) *MultiContractManager {
    return &MultiContractManager{
        client:    cli,
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

func (m *MultiContractManager) CallAll(ctx context.Context, caller *types.Address, method string, params ...interface{}) map[string]interface{} {
    results := make(map[string]interface{})
    
    m.mutex.RLock()
    contracts := make(map[string]*smartcontract.Instance)
    for name, instance := range m.contracts {
        contracts[name] = instance
    }
    m.mutex.RUnlock()

    for name, instance := range contracts {
        result, err := instance.Call(ctx, caller, method, params...)
        if err != nil {
            results[name] = err
        } else {
            results[name] = result
        }
    }

    return results
}
```

## 🚨 Error Handling

### Comprehensive Error Handling

```go
// Handle various contract interaction errors
func SafeContractCall(ctx context.Context, instance *smartcontract.Instance, caller *types.Address, method string, params ...interface{}) error {
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
    result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
    if err != nil {
        if strings.Contains(err.Error(), "REVERT") {
            return fmt.Errorf("contract execution reverted: %w", err)
        }
        if strings.Contains(err.Error(), "OUT_OF_ENERGY") {
            return fmt.Errorf("insufficient energy for execution: %w", err)
        }
        if strings.Contains(err.Error(), "OUT_OF_TIME") {
            return fmt.Errorf("transaction timeout: %w", err)
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
    // Mock client for testing
    mockClient := &MockClient{}
    mgr := smartcontract.NewManager(mockClient)

    owner := types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
    
    // Test deployment
    contractAddr, txid, err := mgr.Deploy(
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
    require.NotNil(t, contractAddr)
    require.NotEmpty(t, txid)
}

func TestContractMethodCall(t *testing.T) {
    mockClient := &MockClient{}
    contractAddr := types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
    
    instance, err := smartcontract.NewInstance(mockClient, contractAddr, testABI)
    require.NoError(t, err)

    caller := types.MustNewAddressFromBase58("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
    
    // Test method call
    tx, err := instance.Invoke(context.Background(), caller, 0, "setValue", big.NewInt(42))
    require.NoError(t, err)
    require.NotNil(t, tx)
}
```

The smartcontract package provides powerful tools for all your contract deployment and interaction needs. Use these patterns to build sophisticated decentralized applications on TRON! 🚀
