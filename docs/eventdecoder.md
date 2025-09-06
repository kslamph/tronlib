# 🎭 Event Decoder Package Reference

The `eventdecoder` package provides powerful tools for decoding smart contract event logs into structured, readable data. It includes built-in support for common TRC20 events, hundreds of common events and allows runtime registration of custom contract ABIs.

## 📚 Learning Path

This document is part of the TronLib learning path:
1. [Quick Start Guide](quickstart.md) - Basic usage
2. [Architecture Overview](architecture.md) - Understanding the design
3. **Event Decoder Package Reference** (this document) - Event log processing
4. [Other Package Documentation](../README.md#package-references) - Additional functionality
5. [API Reference](API_REFERENCE.md) - Complete function documentation

## 📋 Overview

The eventdecoder package features:
- **Built-in TRC20 Support** - Pre-registered Transfer and Approval events
- **Runtime ABI Registration** - Add custom contract ABIs dynamically
- **Structured Output** - Convert raw logs into typed event data
- **Graceful Fallback** - Handle unknown events without errors
- **Signature Caching** - Efficient lookup of event signatures
- **Multi-Contract Support** - Decode events from multiple contracts in one transaction

## 🏗️ Core Components

### DecodedEvent Structure

```go
type DecodedEvent struct {
    EventName  string
    Signature  string
    Parameters []EventParameter
}

type EventParameter struct {
    Name    string
    Type    string
    Value   interface{}
    Indexed bool
}
```

### Built-in Event Registry

The package automatically includes common TRC20 event signatures:
- `Transfer(address indexed from, address indexed to, uint256 value)`
- `Approval(address indexed owner, address indexed spender, uint256 value)`

## 🚀 Basic Usage

### Simple Event Decoding

```go
package main

import (
    "encoding/hex"
    "fmt"
    "log"

    "github.com/kslamph/tronlib/pkg/eventdecoder"
    "github.com/kslamph/tronlib/pkg/types"
)

func main() {
    // Example: TRC20 Transfer event log data
    // This data would typically come from transaction logs
    
    // Transfer event signature hash
    transferSig, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
    
    // Indexed parameters (from, to addresses)
    fromTopic, _ := hex.DecodeString("000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
    toTopic, _ := hex.DecodeString("0000000000000000000000004e83362442b8d1bec281594cea3050c8eb01311c")
    
    // Non-indexed data (amount)
    amountData, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000003e8") // 1000

    topics := [][]byte{transferSig, fromTopic, toTopic}
    
    // Decode the event
    event, err := eventdecoder.DecodeLog(topics, amountData)
    if err != nil {
        log.Fatalf("Failed to decode event: %v", err)
    }

    // Print decoded event
    fmt.Printf("Event: %s\n", event.EventName)
    fmt.Printf("Signature: %s\n", event.Signature)
    
    for _, param := range event.Parameters {
        fmt.Printf("  %s (%s): %v\n", param.Name, param.Type, param.Value)
    }
    
    // Output:
    // Event: Transfer
    // Signature: Transfer(address,address,uint256)
    //   from (address): 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48
    //   to (address): 0x4e83362442b8d1bec281594cea3050c8eb01311c
    //   value (uint256): 1000
}
```

### Decoding Transaction Events

```go
// Decode events from a transaction result
func DecodeTransactionEvents(result *client.BroadcastResult) {
    if len(result.Logs) == 0 {
        fmt.Println("No events emitted")
        return
    }

    fmt.Printf("Transaction %s emitted %d events:\n", result.TxID, len(result.Logs))
    
    for i, log := range result.Logs {
        // Get contract address that emitted the event
        contractAddr := types.MustNewAddressFromBytes(log.GetAddress())
        
        // Decode the event
        event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
        if err != nil {
            fmt.Printf("  [%d] Failed to decode event from %s: %v\n", i, contractAddr, err)
            continue
        }
        
        fmt.Printf("  [%d] %s.%s:\n", i, contractAddr, event.EventName)
        for _, param := range event.Parameters {
            indexedStr := ""
            if param.Indexed {
                indexedStr = " (indexed)"
            }
            fmt.Printf("      %s%s: %v\n", param.Name, indexedStr, param.Value)
        }
    }
}

// Usage after a contract transaction
result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
if err == nil {
    DecodeTransactionEvents(result)
}
```

## 📝 Registering Custom ABIs

### Register ABI from JSON

```go
// Register a custom contract ABI for event decoding
customABI := `{
    "entrys": [
        {
            "type": "event",
            "name": "UserRegistered",
            "inputs": [
                {"name": "user", "type": "address", "indexed": true},
                {"name": "email", "type": "string", "indexed": false},
                {"name": "timestamp", "type": "uint256", "indexed": false}
            ]
        },
        {
            "type": "event", 
            "name": "ProfileUpdated",
            "inputs": [
                {"name": "user", "type": "address", "indexed": true},
                {"name": "field", "type": "string", "indexed": true},
                {"name": "oldValue", "type": "string", "indexed": false},
                {"name": "newValue", "type": "string", "indexed": false}
            ]
        }
    ]
}`

// Register the ABI
err := eventdecoder.RegisterABIJSON(customABI)
if err != nil {
    log.Fatalf("Failed to register ABI: %v", err)
}

fmt.Println("✅ Custom ABI registered successfully")

// Now events from this contract will be automatically decoded
```

### Register ABI from Object

```go
// If you already have a parsed ABI object
var abiObject core.SmartContract_ABI
err := json.Unmarshal([]byte(abiJSON), &abiObject)
if err != nil {
    log.Fatal(err)
}

// Register the ABI object directly
err = eventdecoder.RegisterABIObject(&abiObject)
if err != nil {
    log.Fatalf("Failed to register ABI object: %v", err)
}
```

### Batch ABI Registration

```go
// Register multiple contract ABIs at once
type ContractABI struct {
    Name string
    ABI  string
}

func RegisterContractABIs(abis []ContractABI) error {
    for _, contract := range abis {
        fmt.Printf("Registering ABI for %s...\n", contract.Name)
        
        err := eventdecoder.RegisterABIJSON(contract.ABI)
        if err != nil {
            return fmt.Errorf("failed to register %s ABI: %w", contract.Name, err)
        }
    }
    
    fmt.Printf("✅ Successfully registered %d contract ABIs\n", len(abis))
    return nil
}

// Usage
contractABIs := []ContractABI{
    {"DEX", dexABI},
    {"Lending", lendingABI},
    {"Governance", governanceABI},
}

err := RegisterContractABIs(contractABIs)
if err != nil {
    log.Fatal(err)
}
```

## 🔍 Advanced Event Processing

### Event Filtering by Type

```go
// Filter events by type from transaction logs
func FilterEventsByType(logs []*core.TransactionInfo_Log, eventType string) []eventdecoder.DecodedEvent {
    var filteredEvents []eventdecoder.DecodedEvent
    
    for _, log := range logs {
        event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
        if err != nil {
            continue // Skip events that can't be decoded
        }
        
        if event.EventName == eventType {
            filteredEvents = append(filteredEvents, *event)
        }
    }
    
    return filteredEvents
}

// Usage
transferEvents := FilterEventsByType(result.Logs, "Transfer")
approvalEvents := FilterEventsByType(result.Logs, "Approval")

fmt.Printf("Found %d Transfer events and %d Approval events\n", 
    len(transferEvents), len(approvalEvents))
```

### Event Processing by Contract

```go
// Process events from specific contracts
func ProcessEventsByContract(logs []*core.TransactionInfo_Log, contractAddresses []*types.Address) map[string][]eventdecoder.DecodedEvent {
    eventsByContract := make(map[string][]eventdecoder.DecodedEvent)
    
    // Create address lookup map
    addressMap := make(map[string]bool)
    for _, addr := range contractAddresses {
        addressMap[addr.Hex()] = true
    }
    
    for _, log := range logs {
        // Check if event is from one of our target contracts
        logAddr := types.MustNewAddressFromBytes(log.GetAddress())
        if !addressMap[logAddr.Hex()] {
            continue
        }
        
        // Decode event
        event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
        if err != nil {
            continue
        }
        
        // Group by contract address
        addrStr := logAddr.String()
        eventsByContract[addrStr] = append(eventsByContract[addrStr], *event)
    }
    
    return eventsByContract
}

// Usage
contracts := []*types.Address{
    types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"), // USDT
    types.MustNewAddressFromBase58("TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8"), // USDC
}

eventsByContract := ProcessEventsByContract(result.Logs, contracts)

for contractAddr, events := range eventsByContract {
    fmt.Printf("Contract %s emitted %d events:\n", contractAddr, len(events))
    for _, event := range events {
        fmt.Printf("  - %s\n", event.EventName)
    }
}
```

### Real-time Event Processing

```go
// Process events in real-time from new transactions
func ProcessNewTransactionEvents(ctx context.Context, cli *client.Client, contractAddr *types.Address) {
    // This is a conceptual example - you'd need to implement transaction monitoring
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Get latest transactions (implementation specific)
            transactions := getLatestTransactions(cli, contractAddr)
            
            for _, tx := range transactions {
                // Get transaction info with logs
                txInfo, err := cli.GetTransactionInfo(ctx, tx.TxID)
                if err != nil {
                    continue
                }
                
                // Process events
                for _, log := range txInfo.GetLog() {
                    // Only process events from our contract
                    logAddr := types.MustNewAddressFromBytes(log.GetAddress())
                    if !logAddr.Equal(contractAddr) {
                        continue
                    }
                    
                    event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
                    if err != nil {
                        fmt.Printf("Failed to decode event: %v\n", err)
                        continue
                    }
                    
                    // Handle specific event types
                    switch event.EventName {
                    case "Transfer":
                        handleTransferEvent(event)
                    case "Approval":
                        handleApprovalEvent(event)
                    default:
                        fmt.Printf("Unknown event: %s\n", event.EventName)
                    }
                }
            }
            
            time.Sleep(5 * time.Second) // Poll every 5 seconds
        }
    }
}

func handleTransferEvent(event *eventdecoder.DecodedEvent) {
    // Extract transfer details
    var from, to string
    var amount *big.Int
    
    for _, param := range event.Parameters {
        switch param.Name {
        case "from":
            from = param.Value.(string)
        case "to":
            to = param.Value.(string)
        case "value":
            amount = param.Value.(*big.Int)
        }
    }
    
    fmt.Printf("🔄 Transfer: %s → %s (Amount: %s)\n", from, to, amount.String())
}
```

## 🎯 Event Analysis Patterns

### Transaction Summary Generator

```go
// Generate a human-readable summary of transaction events
func GenerateTransactionSummary(logs []*core.TransactionInfo_Log) string {
    var summary strings.Builder
    eventCounts := make(map[string]int)
    
    summary.WriteString("Transaction Summary:\n")
    
    for _, log := range logs {
        contractAddr := types.MustNewAddressFromBytes(log.GetAddress())
        
        event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
        if err != nil {
            summary.WriteString(fmt.Sprintf("  • Unknown event from %s\n", contractAddr))
            continue
        }
        
        eventCounts[event.EventName]++
        
        switch event.EventName {
        case "Transfer":
            summary.WriteString(generateTransferSummary(event, contractAddr))
        case "Approval":
            summary.WriteString(generateApprovalSummary(event, contractAddr))
        default:
            summary.WriteString(fmt.Sprintf("  • %s event from %s\n", event.EventName, contractAddr))
        }
    }
    
    // Add event counts
    summary.WriteString("\nEvent Counts:\n")
    for eventType, count := range eventCounts {
        summary.WriteString(fmt.Sprintf("  %s: %d\n", eventType, count))
    }
    
    return summary.String()
}

func generateTransferSummary(event *eventdecoder.DecodedEvent, contractAddr *types.Address) string {
    var from, to, amount string
    
    for _, param := range event.Parameters {
        switch param.Name {
        case "from":
            from = param.Value.(string)
        case "to":
            to = param.Value.(string)
        case "value":
            amount = param.Value.(*big.Int).String()
        }
    }
    
    return fmt.Sprintf("  💸 %s tokens transferred from %s to %s\n", amount, from[:10]+"...", to[:10]+"...")
}

func generateApprovalSummary(event *eventdecoder.DecodedEvent, contractAddr *types.Address) string {
    var owner, spender, amount string
    
    for _, param := range event.Parameters {
        switch param.Name {
        case "owner":
            owner = param.Value.(string)
        case "spender":
            spender = param.Value.(string)
        case "value":
            amount = param.Value.(*big.Int).String()
        }
    }
    
    return fmt.Sprintf("  ✅ %s approved %s tokens for %s\n", owner[:10]+"...", amount, spender[:10]+"...")
}
```

### Event Analytics

```go
// Analyze events for insights
type EventAnalytics struct {
    TotalEvents     int
    EventTypes      map[string]int
    ContractCounts  map[string]int
    TransferVolume  *big.Int
    UniqueAddresses map[string]bool
}

func AnalyzeEvents(logs []*core.TransactionInfo_Log) *EventAnalytics {
    analytics := &EventAnalytics{
        EventTypes:      make(map[string]int),
        ContractCounts:  make(map[string]int),
        TransferVolume:  big.NewInt(0),
        UniqueAddresses: make(map[string]bool),
    }
    
    for _, log := range logs {
        contractAddr := types.MustNewAddressFromBytes(log.GetAddress())
        analytics.ContractCounts[contractAddr.String()]++
        analytics.TotalEvents++
        
        event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
        if err != nil {
            analytics.EventTypes["Unknown"]++
            continue
        }
        
        analytics.EventTypes[event.EventName]++
        
        // Special handling for Transfer events
        if event.EventName == "Transfer" {
            for _, param := range event.Parameters {
                switch param.Name {
                case "from", "to":
                    if addr, ok := param.Value.(string); ok {
                        analytics.UniqueAddresses[addr] = true
                    }
                case "value":
                    if amount, ok := param.Value.(*big.Int); ok {
                        analytics.TransferVolume.Add(analytics.TransferVolume, amount)
                    }
                }
            }
        }
    }
    
    return analytics
}

func (a *EventAnalytics) PrintSummary() {
    fmt.Printf("📊 Event Analytics Summary:\n")
    fmt.Printf("Total Events: %d\n", a.TotalEvents)
    fmt.Printf("Unique Addresses: %d\n", len(a.UniqueAddresses))
    fmt.Printf("Total Transfer Volume: %s\n", a.TransferVolume.String())
    
    fmt.Println("\nEvent Types:")
    for eventType, count := range a.EventTypes {
        fmt.Printf("  %s: %d\n", eventType, count)
    }
    
    fmt.Println("\nContracts:")
    for contract, count := range a.ContractCounts {
        fmt.Printf("  %s: %d events\n", contract, count)
    }
}
```

## 🔧 Utilities and Helpers

### Event Signature Utilities

```go
// Generate event signature for manual lookup
func GenerateEventSignature(eventName string, paramTypes []string) []byte {
    signature := fmt.Sprintf("%s(%s)", eventName, strings.Join(paramTypes, ","))
    hash := crypto.Keccak256Hash([]byte(signature))
    return hash.Bytes()
}

// Check if event signature is registered
func IsEventRegistered(topics [][]byte) bool {
    if len(topics) == 0 {
        return false
    }
    
    // Try to decode - if successful, it's registered
    _, err := eventdecoder.DecodeLog(topics, []byte{})
    return err == nil
}

// Get all registered event signatures
func GetRegisteredSignatures() map[string]string {
    // This would need to be implemented in the eventdecoder package
    // Return map of signature -> event name
    return map[string]string{
        "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef": "Transfer(address,address,uint256)",
        "8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925": "Approval(address,address,uint256)",
    }
}
```

### Event Validation

```go
// Validate event structure
func ValidateEvent(event *eventdecoder.DecodedEvent) error {
    if event.EventName == "" {
        return errors.New("event name cannot be empty")
    }
    
    if len(event.Parameters) == 0 {
        return errors.New("event must have at least one parameter")
    }
    
    // Validate parameter structure
    for i, param := range event.Parameters {
        if param.Name == "" {
            return fmt.Errorf("parameter %d has empty name", i)
        }
        if param.Type == "" {
            return fmt.Errorf("parameter %d (%s) has empty type", i, param.Name)
        }
        if param.Value == nil {
            return fmt.Errorf("parameter %d (%s) has nil value", i, param.Name)
        }
    }
    
    return nil
}

// Validate event against expected structure
func ValidateEventStructure(event *eventdecoder.DecodedEvent, expectedName string, expectedParams []string) error {
    if event.EventName != expectedName {
        return fmt.Errorf("expected event %s, got %s", expectedName, event.EventName)
    }
    
    if len(event.Parameters) != len(expectedParams) {
        return fmt.Errorf("expected %d parameters, got %d", len(expectedParams), len(event.Parameters))
    }
    
    for i, expectedParam := range expectedParams {
        if event.Parameters[i].Name != expectedParam {
            return fmt.Errorf("parameter %d: expected %s, got %s", i, expectedParam, event.Parameters[i].Name)
        }
    }
    
    return nil
}
```

## 🚨 Error Handling

### Common Error Patterns

```go
// Handle different types of decoding errors
func SafeDecodeEvent(topics [][]byte, data []byte) (*eventdecoder.DecodedEvent, error) {
    event, err := eventdecoder.DecodeLog(topics, data)
    if err != nil {
        // Check for specific error types
        if strings.Contains(err.Error(), "unknown signature") {
            // Event signature not registered
            return &eventdecoder.DecodedEvent{
                EventName:  "UnknownEvent",
                Signature:  hex.EncodeToString(topics[0]),
                Parameters: []eventdecoder.EventParameter{},
            }, nil
        }
        
        if strings.Contains(err.Error(), "invalid data length") {
            return nil, fmt.Errorf("event data corrupted: %w", err)
        }
        
        if strings.Contains(err.Error(), "invalid topic count") {
            return nil, fmt.Errorf("event topics malformed: %w", err)
        }
        
        return nil, fmt.Errorf("unknown decoding error: %w", err)
    }
    
    return event, nil
}
```

## 🧪 Testing

### Event Decoding Tests

```go
func TestEventDecoding(t *testing.T) {
    // Test TRC20 Transfer event
    transferSig, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
    fromTopic, _ := hex.DecodeString("000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
    toTopic, _ := hex.DecodeString("0000000000000000000000004e83362442b8d1bec281594cea3050c8eb01311c")
    amountData, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000003e8")
    
    topics := [][]byte{transferSig, fromTopic, toTopic}
    
    event, err := eventdecoder.DecodeLog(topics, amountData)
    require.NoError(t, err)
    require.Equal(t, "Transfer", event.EventName)
    require.Len(t, event.Parameters, 3)
    
    // Validate parameters
    assert.Equal(t, "from", event.Parameters[0].Name)
    assert.Equal(t, "to", event.Parameters[1].Name)
    assert.Equal(t, "value", event.Parameters[2].Name)
}

func TestCustomEventRegistration(t *testing.T) {
    customABI := `{"entrys":[{"type":"event","name":"CustomEvent","inputs":[{"name":"param","type":"uint256","indexed":false}]}]}`
    
    err := eventdecoder.RegisterABIJSON(customABI)
    require.NoError(t, err)
    
    // Test that custom event can now be decoded
    // (You would need to construct proper test data)
}
```

### Mock Event Generation

```go
// Generate mock events for testing
func GenerateMockTransferEvent(from, to *types.Address, amount *big.Int) ([][]byte, []byte) {
    // Transfer signature
    transferSig, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
    
    // Encode addresses as topics
    fromTopic := make([]byte, 32)
    copy(fromTopic[12:], from.BytesEVM())
    
    toTopic := make([]byte, 32)
    copy(toTopic[12:], to.BytesEVM())
    
    topics := [][]byte{transferSig, fromTopic, toTopic}
    
    // Encode amount as data
    data := make([]byte, 32)
    amount.FillBytes(data)
    
    return topics, data
}

// Usage in tests
func TestTransferEventDecoding(t *testing.T) {
    from := types.MustNewAddressFromBase58("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH")
    to := types.MustNewAddressFromBase58("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
    amount := big.NewInt(1000)
    
    topics, data := GenerateMockTransferEvent(from, to, amount)
    
    event, err := eventdecoder.DecodeLog(topics, data)
    require.NoError(t, err)
    require.Equal(t, "Transfer", event.EventName)
}
```

The eventdecoder package transforms raw blockchain event logs into meaningful, structured data. Use these patterns to build rich event-driven applications and analytics tools! 🚀
