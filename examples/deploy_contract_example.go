// Example of using the updated DeployContract method
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
)

func main() {
	// Create client
	c, err := client.NewClient(client.DefaultClientConfig("grpc.nile.trongrid.io:50051"))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Create smartcontract manager
	manager := smartcontract.NewManager(c)
	ctx := context.Background()

	// Example 1: Deploy a simple contract without constructor parameters
	simpleContractBytecode, _ := hex.DecodeString("608060405234801561001057600080fd5b50")
	simpleContractABI := `[]` // Empty ABI for simple contract

	fmt.Println("Example 1: Deploy simple contract without constructor")
	_, err = manager.DeployContract(
		ctx,
		"TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g", // Owner address
		"SimpleContract",                       // Contract name (can be empty)
		simpleContractABI,                      // ABI JSON string
		simpleContractBytecode,                 // Bytecode as []byte
		0,                                      // Call value (TRX to send)
		50,                                     // Consume user resource percent (0-100)
		1000000,                                // Origin energy limit
		// No constructor parameters
	)
	if err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
	}

	// Example 2: Deploy TRC20 token with constructor parameters
	trc20Bytecode, _ := hex.DecodeString("60806040523480156100...")
	trc20ABI := `[{
		"inputs": [
			{"internalType": "string", "name": "name_", "type": "string"},
			{"internalType": "string", "name": "symbol_", "type": "string"},
			{"internalType": "uint8", "name": "decimals_", "type": "uint8"},
			{"internalType": "uint256", "name": "initialSupply_", "type": "uint256"}
		],
		"stateMutability": "nonpayable",
		"type": "constructor"
	}]`

	fmt.Println("Example 2: Deploy TRC20 token with constructor parameters")
	_, err = manager.DeployContract(
		ctx,
		"TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g", // Owner address
		"My Test Token",                        // Contract name
		trc20ABI,                               // ABI JSON string
		trc20Bytecode,                          // Bytecode as []byte
		0,                                      // Call value
		100,                                    // Consume user resource percent
		10000000,                               // Origin energy limit
		// Constructor parameters (automatically encoded)
		"TestToken",    // name_
		"TTK",          // symbol_
		uint8(18),      // decimals_
		"1000000000000000000000000", // initialSupply_ (1M tokens with 18 decimals)
	)
	if err != nil {
		fmt.Printf("TRC20 deployment failed: %v\n", err)
	}

	// Example 3: Deploy contract with mixed parameter types
	complexABI := `[{
		"inputs": [
			{"internalType": "address", "name": "_myAddress", "type": "address"},
			{"internalType": "bool", "name": "_myBool", "type": "bool"},
			{"internalType": "uint256", "name": "_myUint", "type": "uint256"}
		],
		"stateMutability": "nonpayable",
		"type": "constructor"
	}]`

	fmt.Println("Example 3: Deploy contract with mixed parameter types")
	_, err = manager.DeployContract(
		ctx,
		"TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g", // Owner address
		"",                                     // Empty contract name (allowed)
		complexABI,                             // ABI JSON string
		trc20Bytecode,                          // Bytecode as []byte
		0,                                      // Call value
		75,                                     // Consume user resource percent
		5000000,                                // Origin energy limit
		// Constructor parameters with different types
		"TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g", // address parameter
		true,                                   // bool parameter
		42,                                     // uint256 parameter
	)
	if err != nil {
		fmt.Printf("Complex contract deployment failed: %v\n", err)
	}

	fmt.Println("All deployment examples completed!")
}