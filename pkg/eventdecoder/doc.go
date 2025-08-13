// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
