// Example: Enhanced TRC20 implementation using the new smartcontract interface
package main

import (
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/smartcontract"
)

// ERC20 ABI for demonstration
const ERC20_ABI = `[
	{
		"constant": true,
		"inputs": [],
		"name": "name",
		"outputs": [{"name": "", "type": "string"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "symbol",
		"outputs": [{"name": "", "type": "string"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "decimals",
		"outputs": [{"name": "", "type": "uint8"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_to", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "transfer",
		"outputs": [{"name": "", "type": "bool"}],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`

func main() {
	// Create a contract instance using the enhanced interface
	contract, err := smartcontract.NewContract(ERC20_ABI, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err != nil {
		log.Fatalf("Failed to create contract: %v", err)
	}

	fmt.Printf("Contract Address: %s\n", contract.Address)
	fmt.Printf("Contract Address Bytes: %x\n", contract.AddressBytes)

	// Example 1: Encode a simple method call (name)
	nameCallData, err := contract.EncodeInput("name")
	if err != nil {
		log.Fatalf("Failed to encode name call: %v", err)
	}
	fmt.Printf("name() call data: %x\n", nameCallData)

	// Example 2: Encode a method call with parameters (balanceOf)
	balanceCallData, err := contract.EncodeInput("balanceOf", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err != nil {
		log.Fatalf("Failed to encode balanceOf call: %v", err)
	}
	fmt.Printf("balanceOf(address) call data: %x\n", balanceCallData)

	// Example 3: Encode a method call with multiple parameters (transfer)
	transferCallData, err := contract.EncodeInput("transfer", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", "1000000000000000000")
	if err != nil {
		log.Fatalf("Failed to encode transfer call: %v", err)
	}
	fmt.Printf("transfer(address,uint256) call data: %x\n", transferCallData)

	// Example 4: Decode input data
	decodedInput, err := contract.DecodeInputData(transferCallData)
	if err != nil {
		log.Fatalf("Failed to decode input data: %v", err)
	}
	fmt.Printf("Decoded method: %s\n", decodedInput.Method)
	fmt.Printf("Decoded parameters:\n")
	for i, param := range decodedInput.Parameters {
		fmt.Printf("  %d. %s (%s): %v\n", i+1, param.Name, param.Type, param.Value)
	}

	fmt.Println("\n=== Enhanced TRC20 Usage Example ===")
	fmt.Println("This demonstrates how the TRC20 package can now use:")
	fmt.Println("1. Method names instead of hardcoded signatures")
	fmt.Println("2. Proper parameter encoding/decoding")
	fmt.Println("3. Clean ABI-based interface")
	fmt.Println("4. Event decoding capabilities")
}