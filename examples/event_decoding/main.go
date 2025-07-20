package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	// USDT contract ABI (simplified for demonstration)
	usdtABI := `[
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "from", "type": "address"},
				{"indexed": true, "name": "to", "type": "address"},
				{"indexed": false, "name": "value", "type": "uint256"}
			],
			"name": "Transfer",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "owner", "type": "address"},
				{"indexed": true, "name": "spender", "type": "address"},
				{"indexed": false, "name": "value", "type": "uint256"}
			],
			"name": "Approval",
			"type": "event"
		}
	]`

	// Create contract instance
	contract, err := types.NewContract(usdtABI, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err != nil {
		log.Fatalf("Failed to create contract: %v", err)
	}

	// Example Transfer event data
	// Event signature: ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
	// From address: 1db1cb12db47db0b3b722d489f424bd6fe6276f6
	// To address: e1719cd39147e9c501fbf9374f9e382ffa6d3043
	// Value: 100000000 (in data field)

	topics := [][]byte{
		hexDecode("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"), // Event signature
		hexDecode("0000000000000000000000001db1cb12db47db0b3b722d489f424bd6fe6276f6"), // From address
		hexDecode("000000000000000000000000e1719cd39147e9c501fbf9374f9e382ffa6d3043"), // To address
	}

	data := hexDecode("0000000000000000000000000000000000000000000000000000000005f5e100") // Value: 100000000

	fmt.Println("=== Event Decoding Example ===")
	fmt.Println()

	// 1. Decode the complete event log
	fmt.Println("1. Decoding complete event log:")
	decodedEvent, err := contract.DecodeEventLog(topics, data)
	if err != nil {
		log.Fatalf("Failed to decode event: %v", err)
	}

	fmt.Printf("   Event Name: %s\n", decodedEvent.EventName)
	fmt.Println("   Parameters:")
	for _, param := range decodedEvent.Parameters {
		fmt.Printf("     %s (%s): %v (indexed: %t)\n",
			param.Name, param.Type, param.Value, param.Indexed)
	}
	fmt.Println()

	// 2. Decode just the event signature
	fmt.Println("2. Decoding event signature only:")
	eventSignature, err := contract.DecodeEventSignature(topics[0])
	if err != nil {
		log.Fatalf("Failed to decode event signature: %v", err)
	}
	fmt.Printf("   Event Signature: %s\n", eventSignature)
	fmt.Println()

	// 3. Explain the event structure
	fmt.Println("3. Event Structure Explanation:")
	fmt.Println("   - topics[0]: Event signature (32 bytes)")
	fmt.Println("   - topics[1]: 'from' address (indexed parameter)")
	fmt.Println("   - topics[2]: 'to' address (indexed parameter)")
	fmt.Println("   - data: 'value' amount (non-indexed parameter)")
	fmt.Println()
	fmt.Println("   Indexed parameters (address, uint256, etc.) go into topics")
	fmt.Println("   Non-indexed parameters (string, bytes, etc.) go into data")
	fmt.Println()

	// 4. Show raw data for reference
	fmt.Println("4. Raw Event Data:")
	fmt.Printf("   Event Signature: 0x%s\n", hex.EncodeToString(topics[0]))
	fmt.Printf("   From Address: 0x%s\n", hex.EncodeToString(topics[1]))
	fmt.Printf("   To Address: 0x%s\n", hex.EncodeToString(topics[2]))
	fmt.Printf("   Value Data: 0x%s\n", hex.EncodeToString(data))
}

// Helper function to decode hex strings
func hexDecode(s string) []byte {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}
