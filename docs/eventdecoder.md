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
    EventName  string                  `json:"eventName"`
    Parameters []DecodedEventParameter `json:"parameters"`
    Contract   string                  `json:"contract"`
}

type DecodedEventParameter struct {
    Name    string `json:"name"`
    Type    string `json:"type"`
    Value   string `json:"value"` // ABI-encoded string, not interface{}
    Indexed bool   `json:"indexed"`
}
```

> **Note:** `Value` is a `string`, not `interface{}`. It contains the
> ABI-encoded representation (e.g. an address string for address params, a
> decimal string for uint256 params). To obtain the numeric value, parse it
> with `new(big.Int).SetBytes([]byte(value))` or `decimal.NewFromString(value)`
> as appropriate for the param's `Type`.

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
)

func main() {
    // Example: TRC20 Transfer event log data
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
    fmt.Printf("Contract: %s\n", event.Contract)

    for _, param := range event.Parameters {
        fmt.Printf("  %s (%s): %s\n", param.Name, param.Type, param.Value)
    }

    // Output:
    // Event: Transfer
    // Contract: TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t
    //   from (address): 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48
    //   to (address): 0x4e83362442b8d1bec281594cea3050c8eb01311c
    //   value (uint256): 1000
}
```

### Decoding Transaction Events

```go
import (
    "github.com/kslamph/tronlib/pkg/eventdecoder"
    "github.com/kslamph/tronlib/pkg/network"
    "github.com/kslamph/tronlib/pkg/types"
)

// Decode events from a broadcast result
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
            fmt.Printf("      %s%s: %s\n", param.Name, indexedStr, param.Value)
        }
    }
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
        logAddr := types.MustNewAddressFromBytes(log.GetAddress())
        if !addressMap[logAddr.Hex()] {
            continue
        }

        event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
        if err != nil {
            continue
        }

        addrStr := logAddr.String()
        eventsByContract[addrStr] = append(eventsByContract[addrStr], *event)
    }

    return eventsByContract
}
```

### Real-time Event Processing

To fetch transaction info by ID, use the **network** package (there is no
`cli.GetTransactionInfo` method):

```go
import (
    "github.com/kslamph/tronlib/pkg/eventdecoder"
    "github.com/kslamph/tronlib/pkg/network"
)

func ProcessNewTransactionEvents(ctx context.Context, netMgr *network.NetworkManager, contractAddr *types.Address) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Fetch transaction info by ID (implementation-specific txID source)
            txInfo, err := netMgr.GetTransactionInfoById(ctx, txID)
            if err != nil {
                continue
            }

            for _, log := range txInfo.GetLog() {
                logAddr := types.MustNewAddressFromBytes(log.GetAddress())
                if !logAddr.Equal(contractAddr) {
                    continue
                }

                event, err := eventdecoder.DecodeLog(log.GetTopics(), log.GetData())
                if err != nil {
                    fmt.Printf("Failed to decode event: %v\n", err)
                    continue
                }

                switch event.EventName {
                case "Transfer":
                    handleTransferEvent(event)
                case "Approval":
                    handleApprovalEvent(event)
                default:
                    fmt.Printf("Unknown event: %s\n", event.EventName)
                }
            }

            time.Sleep(5 * time.Second)
        }
    }
}

func handleTransferEvent(event *eventdecoder.DecodedEvent) {
    // param.Value is a string — parse for numeric values
    var from, to string
    var amount *big.Int

    for _, param := range event.Parameters {
        switch param.Name {
        case "from":
            from = param.Value
        case "to":
            to = param.Value
        case "value":
            amount = new(big.Int)
            amount.SetString(param.Value, 10) // decimal string
        }
    }

    fmt.Printf("🔄 Transfer: %s → %s (Amount: %s)\n", from, to, amount.String())
}
```

## 🎯 Event Analysis Patterns

### Transaction Summary Generator

```go
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
            from = param.Value
        case "to":
            to = param.Value
        case "value":
            amount = param.Value
        }
    }

    return fmt.Sprintf("  💸 %s tokens transferred from %s to %s\n", amount, from[:10]+"...", to[:10]+"...")
}

func generateApprovalSummary(event *eventdecoder.DecodedEvent, contractAddr *types.Address) string {
    var owner, spender, amount string

    for _, param := range event.Parameters {
        switch param.Name {
        case "owner":
            owner = param.Value
        case "spender":
            spender = param.Value
        case "value":
            amount = param.Value
        }
    }

    return fmt.Sprintf("  ✅ %s approved %s tokens for %s\n", owner[:10]+"...", amount, spender[:10]+"...")
}
```

### Event Analytics

```go
type EventAnalytics struct {
    TotalEvents     int
    EventTypes      map[string]int
    ContractCounts  map[string]int
    UniqueAddresses map[string]bool
}

func AnalyzeEvents(logs []*core.TransactionInfo_Log) *EventAnalytics {
    analytics := &EventAnalytics{
        EventTypes:      make(map[string]int),
        ContractCounts:  make(map[string]int),
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

        if event.EventName == "Transfer" {
            for _, param := range event.Parameters {
                switch param.Name {
                case "from", "to":
                    analytics.UniqueAddresses[param.Value] = true
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

    for i, param := range event.Parameters {
        if param.Name == "" {
            return fmt.Errorf("parameter %d has empty name", i)
        }
        if param.Type == "" {
            return fmt.Errorf("parameter %d (%s) has empty type", i, param.Name)
        }
        if param.Value == "" {
            return fmt.Errorf("parameter %d (%s) has empty value", i, param.Name)
        }
    }

    return nil
}
```

## 🚨 Error Handling

### Common Error Patterns

```go
func SafeDecodeEvent(topics [][]byte, data []byte) (*eventdecoder.DecodedEvent, error) {
    event, err := eventdecoder.DecodeLog(topics, data)
    if err != nil {
        if strings.Contains(err.Error(), "unknown signature") {
            return &eventdecoder.DecodedEvent{
                EventName:  "UnknownEvent",
                Contract:   hex.EncodeToString(topics[0]),
                Parameters: []eventdecoder.DecodedEventParameter{},
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
    transferSig, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
    fromTopic, _ := hex.DecodeString("000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
    toTopic, _ := hex.DecodeString("0000000000000000000000004e83362442b8d1bec281594cea3050c8eb01311c")
    amountData, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000003e8")

    topics := [][]byte{transferSig, fromTopic, toTopic}

    event, err := eventdecoder.DecodeLog(topics, amountData)
    require.NoError(t, err)
    require.Equal(t, "Transfer", event.EventName)
    require.Len(t, event.Parameters, 3)

    // Validate parameters — param.Value is a string
    assert.Equal(t, "from", event.Parameters[0].Name)
    assert.Equal(t, "to", event.Parameters[1].Name)
    assert.Equal(t, "value", event.Parameters[2].Name)
    // To parse the value as a big.Int: new(big.Int).SetBytes([]byte(event.Parameters[2].Value))
}
```

### Mock Event Generation

```go
// Generate mock events for testing
func GenerateMockTransferEvent(from, to *types.Address, amount *big.Int) ([][]byte, []byte) {
    transferSig, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

    fromTopic := make([]byte, 32)
    copy(fromTopic[12:], from.BytesEVM())

    toTopic := make([]byte, 32)
    copy(toTopic[12:], to.BytesEVM())

    topics := [][]byte{transferSig, fromTopic, toTopic}

    data := make([]byte, 32)
    amount.FillBytes(data)

    return topics, data
}
```

The eventdecoder package transforms raw blockchain event logs into meaningful, structured data. Use these patterns to build rich event-driven applications and analytics tools! 🚀
