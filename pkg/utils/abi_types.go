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

package utils

import (
	"sync"

	"github.com/kslamph/tronlib/pb/core"
)

// ABIProcessor handles all smart contract ABI operations including encoding, decoding, parsing, and event processing
type ABIProcessor struct {
	abi *core.SmartContract_ABI

	// Event signature caches using sync.Once pattern
	eventCacheOnce      sync.Once
	eventSignatureCache map[[32]byte]*core.SmartContract_ABI_Entry

	event4ByteCacheOnce      sync.Once
	event4ByteSignatureCache map[[4]byte]*core.SmartContract_ABI_Entry
}

// DecodedInput represents decoded input data
type DecodedInput struct {
	Method     string                  `json:"method"`
	Parameters []DecodedInputParameter `json:"parameters"`
}

// DecodedInputParameter represents a decoded parameter
type DecodedInputParameter struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// DecodedEvent represents a decoded event
type DecodedEvent struct {
	EventName  string                  `json:"eventName"`
	Parameters []DecodedEventParameter `json:"parameters"`
}

// DecodedEventParameter represents a decoded event parameter
type DecodedEventParameter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Indexed bool   `json:"indexed"`
}
