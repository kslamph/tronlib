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