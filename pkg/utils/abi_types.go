package utils

import (
	"github.com/kslamph/tronlib/pb/core"
)

// ABIProcessor handles all smart contract ABI operations including encoding, decoding, parsing, and event processing
type ABIProcessor struct {
	abi *core.SmartContract_ABI
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
