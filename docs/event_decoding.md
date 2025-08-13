# Event Decoding Guide

Use `pkg/eventdecoder` to decode logs from receipts or simulations.

## Quick Start

```go
sigTopic, _ := hex.DecodeString("ddf252ad1b...3b3ef") // Transfer(address,address,uint256)
fromTopic, _ := hex.DecodeString("0000...06eb48")
toTopic, _ := hex.DecodeString("0000...1311c")
amountData, _ := hex.DecodeString("...0003e8") // 1000

ev, _ := eventdecoder.DecodeLog([][]byte{sigTopic, fromTopic, toTopic}, amountData)
fmt.Println(ev.EventName)
```

## Built-in Events

The event decoder comes with built-in support for common TRC20 events:

- `Transfer(address,address,uint256)`
- `Approval(address,address,uint256)`

## Registering Custom ABIs

To extend coverage for custom events, register ABIs:

```go
eventdecoder.RegisterABIJSON(abiJSON)
// or
var abi core.SmartContract_ABI
_ = json.Unmarshal([]byte(abiJSON), &abi)
eventdecoder.RegisterABIObject(&abi)
```

## Decoding from Broadcast Results

When you have a broadcast result with logs:

```go
for _, lg := range res.Logs {
    ev, err := eventdecoder.DecodeLog(lg.GetTopics(), lg.GetData())
    if err != nil { /* handle */ }
    fmt.Printf("%s\n", ev.EventName)
}
```

## Event Structure

The decoded event has the following structure:

```go
type DecodedEvent struct {
    EventName string
    Signature string
    Inputs    []EventInput
}

type EventInput struct {
    Name  string
    Type  string
    Value interface{}
}
```

## Error Handling

The decoder returns specific errors for common issues:

- `ErrNoMatchingABI` - No ABI registered for the event signature
- `ErrInvalidTopicCount` - Mismatch between expected and actual topic count
- `ErrInvalidDataLength` - Data length doesn't match expected size

Always check for errors when decoding events in production code.
