package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

func main() {
	// Connect to Nile testnet
	cli, err := client.NewClient("grpc://grpc.nile.trongrid.io:50051")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer cli.Close()

	// Create HD wallet signer from mnemonic
	mnemonic := "real payment expose media seed token frequent initial winter alpha glad change hen wheel cancel domain trigger upset reform equal aware mixture drill give"
	// Using standard BIP-44 path for TRON: m/44'/195'/0'/0/0
	path := "m/44'/195'/0'/0/0"
	signer, err := signer.NewHDWalletSigner(mnemonic, "", path)
	if err != nil {
		log.Fatalf("Invalid mnemonic or path: %v", err)
	}

	from := signer.Address()
	fmt.Printf("Wallet Address: %s\n", from.String())

	ctx := context.Background()

	// Check balance
	balance, err := cli.Account().GetBalance(ctx, from)
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}

	// Convert SUN to TRX using utils package for proper formatting
	trxBalance, err := utils.HumanReadableBalance(balance, 6) // 6 decimal places for TRX
	if err != nil {
		log.Printf("Warning: Failed to format balance: %v", err)
		fmt.Printf("Balance: %d SUN\n", balance)
	} else {
		fmt.Printf("Balance: %s TRX\n", trxBalance)
	}

	// Calculate transfer amount (1/100 of balance)
	transferAmount := balance / 100
	if transferAmount <= 0 {
		log.Fatalf("Insufficient balance for transfer")
	}

	transferAmountTRX, _ := utils.HumanReadableBalance(transferAmount, 6)
	fmt.Printf("Transfer Amount: %s TRX\n", transferAmountTRX)

	// Destination address
	to, err := types.NewAddress("TFBLnw94JkSeB2juA1RRtXs5um4uns6yve")
	if err != nil {
		log.Fatalf("Invalid destination address: %v", err)
	}

	// Build and send transaction
	tx, err := cli.Account().TransferTRX(ctx, from, to, transferAmount)
	if err != nil {
		log.Fatalf("Failed to build transaction: %v", err)
	}

	opts := client.DefaultBroadcastOptions()
	opts.WaitForReceipt = true
	opts.WaitTimeout = 30 * time.Second
	opts.FeeLimit = 100_000_000

	result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}

	fmt.Printf("✅ Success! TxID: %s\n", result.TxID)
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Energy Used: %d\n", result.EnergyUsage)
	fmt.Printf("Net Used: %d\n", result.NetUsage)
}
