package main

import (
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	// Network configuration
	NetworkEndpoint = "grpc.trongrid.io:50051"

	// USDT contract address on mainnet
	USDTContractAddress = "TXDk8mbtRbXeYuMNS83CfKPaYYT8XWv9Hz"

	// USDT TRC20 ABI - only including methods we need
	USDT_ABI = `[
		{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"}
	]`
)

func main() {
	// Create client connection
	c, err := client.NewClient(client.DefaultClientConfig())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Create new Contract instance
	contract, err := types.NewContract(USDT_ABI, USDTContractAddress)
	if err != nil {
		log.Fatalf("Failed to create contract: %v", err)
	}

	// Get token information using ContractTrigger

	// Get symbol
	symbolData, err := contract.EncodeInput("symbol")
	if err != nil {
		log.Fatalf("Failed to create symbol call: %v", err)
	}

	// Get decimals
	decimalsData, err := contract.EncodeInput("decimals")
	if err != nil {
		log.Fatalf("Failed to create decimals call: %v", err)
	}

	// Get name
	nameData, err := contract.EncodeInput("name")
	if err != nil {
		log.Fatalf("Failed to create name call: %v", err)
	}

	// Example addresses for queries
	queryAddress, err := types.NewAddress("TH98ue5uws6i5YevPm6J9HkppdrFSuHsy3")
	if err != nil {
		log.Fatalf("Failed to parse query address: %v", err)
	}

	spenderAddr, err := types.NewAddress("TBXW4hS5KYjjbJXDpnrPf4zhkLwrpUjbyz")
	if err != nil {
		log.Fatalf("Failed to parse spender address: %v", err)
	}

	// Get allowance
	allowanceData, err := contract.EncodeInput("allowance", queryAddress.String(), spenderAddr.String())
	if err != nil {
		log.Fatalf("Failed to create allowance call: %v", err)
	}

	// Get balance
	balanceData, err := contract.EncodeInput("balanceOf", queryAddress.String())
	if err != nil {
		log.Fatalf("Failed to create balance call: %v", err)
	}

	// Now let's decode and print the results
	symbol, err := contract.DecodeResult("symbol", [][]byte{symbolData})
	if err != nil {
		log.Fatalf("Failed to decode symbol: %v", err)
	}
	fmt.Printf("Token Symbol: %v\n", symbol)

	decimals, err := contract.DecodeResult("decimals", [][]byte{decimalsData})
	if err != nil {
		log.Fatalf("Failed to decode decimals: %v", err)
	}
	fmt.Printf("Token Decimals: %v\n", decimals)

	name, err := contract.DecodeResult("name", [][]byte{nameData})
	if err != nil {
		log.Fatalf("Failed to decode name: %v", err)
	}
	fmt.Printf("Token Name: %v\n", name)

	allowance, err := contract.DecodeResult("allowance", [][]byte{allowanceData})
	if err != nil {
		log.Fatalf("Failed to decode allowance: %v", err)
	}
	fmt.Printf("Allowance: %v\n", allowance)

	balance, err := contract.DecodeResult("balanceOf", [][]byte{balanceData})
	if err != nil {
		log.Fatalf("Failed to decode balance: %v", err)
	}
	fmt.Printf("Balance: %v\n", balance)
}
