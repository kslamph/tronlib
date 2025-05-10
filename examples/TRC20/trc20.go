package main

import (
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

const (
	ContractAddress = "TCFbhVVNz3KcviKZgAkoW5JUnTAyuwWJY5"
)

func main() {
	// Create client connection
	nodes := client.ShastaNodes()
	tronClient, err := client.NewClient(client.ClientConfig{
		Nodes: nodes,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer tronClient.Close()

	// Create new TRC20 contract instance
	contract, err := smartcontract.NewTRC20Contract(ContractAddress, tronClient)
	if err != nil {
		log.Fatalf("Failed to create TRC20 contract: %v", err)
	}

	// Example: Get token symbol
	symbol, err := contract.Symbol()
	if err != nil {
		log.Fatalf("Failed to get symbol: %v", err)
	}
	fmt.Printf("Token Symbol: %s\n", symbol)

	// Example: Get token decimals
	decimals, err := contract.Decimals()
	if err != nil {
		log.Fatalf("Failed to get decimals: %v", err)
	}
	fmt.Printf("Token Decimals: %d\n", decimals)

	// Example: Get token name
	name, err := contract.Name()
	if err != nil {
		log.Fatalf("Failed to get name: %v", err)
	}
	fmt.Printf("Token Name: %s\n", name)

	// Example: Check allowance
	ownerAddress, err := types.NewAddress("TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x")
	if err != nil {
		log.Fatalf("Failed to parse query address: %v", err)
	}

	spenderAddr, err := types.NewAddress("TBXW4hS5KYjjbJXDpnrPf4zhkLwrpUjbyz")
	if err != nil {
		log.Fatalf("Failed to parse spender address: %v", err)
	}

	allowance, err := contract.Allowance(ownerAddress.String(), spenderAddr.String())
	if err != nil {
		log.Fatalf("Failed to get allowance: %v", err)
	}
	fmt.Printf("Allowance: %v\n", allowance)

	// Example: Check balance
	balance, err := contract.BalanceOf(ownerAddress.String())
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}
	fmt.Printf("Balance: %v\n", balance)

	// Example: Transfer tokens
	transferto, _ := types.NewAddress("TSGkU4jYbYCosYFtrVSYMWGhatFjgSRfnq")
	ownerAccount, _ := types.NewAccountFromPrivateKey("f8c6f45b2aa8b68ab5f3910bdeb5239428b731618113e2881f46e374bf796b02")

	tx := contract.Transfer(ownerAccount.Address().String(), transferto.String(), decimal.NewFromInt(1234)).
		Sign(ownerAccount).
		Broadcast()

	if tx.GetError() != nil {
		log.Println(tx.GetReceipt().Err)
	}
	// Get receipt
	receipt := tx.GetReceipt()
	fmt.Printf("Transaction ID: %s\n", receipt.TxID)
	fmt.Printf("Result: %v\n", receipt.Result)
	if receipt.Message != "" {
		fmt.Printf("Message: %s\n", receipt.Message)
	}

	// Wait for transaction confirmation
	rcp, err := tronClient.WaitForTransactionInfo(receipt.TxID, 9)

	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}
	fmt.Printf("\nTransaction Information:\n")
	fmt.Printf("====================\n")
	fmt.Println("tx result", rcp.Result)
}
