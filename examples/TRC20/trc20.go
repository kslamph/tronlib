package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/helper"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

const (
	ContractAddress = "TCFbhVVNz3KcviKZgAkoW5JUnTAyuwWJY5"
)

func main() {
	// Create client connection
	tronClient, err := client.NewClient(client.ClientConfig{
		NodeAddress: "grpc.shasta.trongrid.io:50051",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer tronClient.Close()

	ctx := context.Background()

	// Create new TRC20 contract instance
	contract, err := trc20.NewTRC20Contract(tronClient, ContractAddress)
	if err != nil {
		log.Fatalf("Failed to create TRC20 contract: %v", err)
	}

	// Example: Get token symbol
	symbol, err := contract.Symbol(ctx)
	if err != nil {
		log.Fatalf("Failed to get symbol: %v", err)
	}
	fmt.Printf("Token Symbol: %s\n", symbol)

	// Example: Get token decimals
	decimals, err := contract.Decimals(ctx)
	if err != nil {
		log.Fatalf("Failed to get decimals: %v", err)
	}
	fmt.Printf("Token Decimals: %d\n", decimals)

	// Example: Get token name
	name, err := contract.Name(ctx)
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

	allowance, err := contract.Allowance(ctx, ownerAddress.String(), spenderAddr.String())
	if err != nil {
		log.Fatalf("Failed to get allowance: %v", err)
	}
	fmt.Printf("Allowance: %v\n", allowance)

	// Example: Check balance
	balance, err := contract.BalanceOf(ctx, ownerAddress.String())
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}
	fmt.Printf("Balance: %v\n", balance)

	// Example: Transfer tokens
	transferto, _ := types.NewAddress("TSGkU4jYbYCosYFtrVSYMWGhatFjgSRfnq")
	ownerAccount, _ := types.NewAccountFromPrivateKey("f8c6f45b2aa8b68ab5f3910bdeb5239428b731618113e2881f46e374bf796b02")

	tx, err := contract.Transfer(ctx, ownerAccount.Address().String(), transferto.String(), decimal.NewFromInt(1234))
	if err != nil {
		log.Fatalf("Failed to transfer tokens: %v", err)
	}
	signed, err := ownerAccount.Sign(tx.GetTransaction())

	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}
	txid := helper.GetTxid(signed)
	fmt.Printf("Transaction ID: %s\n", txid)
	receipt, err := tronClient.BroadcastTransaction(ctx, signed)
	if err != nil {
		log.Fatalf("Failed to broadcast transaction: %v", err)
	}

	fmt.Printf("Result: %v\n", receipt.Result)
	if receipt.Message != nil {
		fmt.Printf("Message: %s\n", string(receipt.Message))
	}

	// Wait for transaction confirmation
	confirmation, err := tronClient.WaitForTransactionInfo(ctx, txid)
	if err != nil {
		log.Fatalf("Failed to get transaction info: %v", err)
	}
	fmt.Printf("\nTransaction Information:\n")
	fmt.Printf("====================\n")
	fmt.Println("tx result", confirmation.Result)
}
