// Package eventdecoder maintains a compact registry of event signatures and
// helpers to decode logs into typed values using minimal ABI fragments.
//
// Two usage modes are supported:
//  1. Register ABI sources at runtime (JSON or *core.SmartContract_ABI)
//  2. Use builtin signatures generated from common ecosystems (e.g., TRC20)
//
// Given topics and data from a TransactionInfo_Log, DecodeLog will look up
// the first topic's signature, build a minimal ABI for the matched
// signature, and return a DecodedEvent. Unknown signatures fall back to a
// placeholder name instead of failing.
//
// # Quick Start
//
//	ev, _ := eventdecoder.DecodeLog(topics, data)
//	_ = ev.EventName
//
// To expand coverage, call RegisterABIJSON or RegisterABIObject with known
// contract ABIs. Builtins include TRC20 events.
//
// # Built-in Events
//
// The package includes built-in support for common TRC20 events:
//   - Transfer(address,address,uint256)
//   - Approval(address,address,uint256)
//
// # Registering Custom ABIs
//
// To decode custom events, register their ABIs:
//
//	eventdecoder.RegisterABIJSON(abiJSON)
//	// or
//	var abi core.SmartContract_ABI
//	_ = json.Unmarshal([]byte(abiJSON), &abi)
//	eventdecoder.RegisterABIObject(&abi)
//
// # Decoded Event Structure
//
// The decoded event has the following structure:
//
//	type DecodedEvent struct {
//	    EventName string
//	    Signature string
//	    Inputs    []EventInput
//	}
//
//	type EventInput struct {
//	    Name  string
//	    Type  string
//	    Value interface{}
//	}
//
// # Error Handling
//
// Common error types:
//   - ErrNoMatchingABI - No ABI registered for the event signature
//   - ErrInvalidTopicCount - Mismatch between expected and actual topic count
//   - ErrInvalidDataLength - Data length doesn't match expected size
//
// Always check for errors in production code.
package eventdecoder
