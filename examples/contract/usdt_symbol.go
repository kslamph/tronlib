package main

import (
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	// Network configuration
	NetworkEndpoint = "grpc.trongrid.io:50051"

	// USDT contract address on mainnet
	USDTContractAddress = "TXDk8mbtRbXeYuMNS83CfKPaYYT8XWv9Hz"

	// Example wallet address to use as owner
	OwnerAddress = "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"
)

func main() {
	// Create client connection
	c, err := client.NewClient(client.DefaultClientConfig())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Create new TRC20 contract instance
	contract, err := smartcontract.NewTRC20Contract(USDTContractAddress, c)
	if err != nil {
		log.Fatalf("Failed to create TRC20 contract: %v", err)
	}

	// Get token information
	symbol, err := contract.Symbol()
	if err != nil {
		log.Fatalf("Failed to get symbol: %v", err)
	}
	fmt.Printf("Token Symbol: %s\n", symbol)

	decimals, err := contract.Decimals()
	if err != nil {
		log.Fatalf("Failed to get decimals: %v", err)
	}
	fmt.Printf("Token Decimals: %d\n", decimals)

	name, err := contract.Name()
	if err != nil {
		log.Fatalf("Failed to get name: %v", err)
	}
	fmt.Printf("Token Name: %s\n", name)

	// Example: Check allowance
	queryAddress, err := types.NewAddress("TH98ue5uws6i5YevPm6J9HkppdrFSuHsy3")
	if err != nil {
		log.Fatalf("Failed to parse query address: %v", err)
	}

	spenderAddr, err := types.NewAddress("TBXW4hS5KYjjbJXDpnrPf4zhkLwrpUjbyz")
	if err != nil {
		log.Fatalf("Failed to parse spender address: %v", err)
	}

	allowance, err := contract.Allowance(queryAddress.String(), spenderAddr.String())
	if err != nil {
		log.Fatalf("Failed to get allowance: %v", err)
	}
	fmt.Printf("Allowance: %v\n", allowance)

	// Example: Check balance
	balance, err := contract.BalanceOf(queryAddress.String())
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}
	fmt.Printf("Balance: %v\n", balance)
}
