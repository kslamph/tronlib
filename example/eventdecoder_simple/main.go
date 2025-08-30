// This snippet is from docs/eventdecoder.md
// Simple Event Decoding
package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/eventdecoder"
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

	for _, param := range event.Parameters {
		fmt.Printf("  %s (%s): %v\n", param.Name, param.Type, param.Value)
	}
}
