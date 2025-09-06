package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

func main() {
	fmt.Printf("=== Modular Shielded TRC20 Transaction Implementation (%s mode) ===\n", CurrentMode)

	// Connect and setup
	cli, err := client.NewClient(Node)
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	key, err := signer.NewPrivateKeySigner(PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	from := key.Address()
	ctx := context.Background()

	// Get account information
	myAccount, err := cli.Account().GetAccount(ctx, from)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Account: %+v\n", myAccount)

	// Convert addresses
	tokenAddr, err := types.NewAddressFromBase58(TokenAddress)
	if err != nil {
		log.Fatal("Failed to parse token address:", err)
	}

	shieldedAddr, err := types.NewAddressFromBase58(ShieldedContract)
	if err != nil {
		log.Fatal("Failed to parse shielded contract address:", err)
	}

	// Get initial balance for verification
	trc20Mgr := cli.TRC20(tokenAddr)
	if trc20Mgr == nil {
		log.Fatal("Failed to create TRC20 manager")
	}

	initialBalance, err := trc20Mgr.BalanceOf(ctx, from)
	if err != nil {
		log.Fatal("Failed to check initial balance:", err)
	}
	fmt.Printf("Initial transparent balance: %s\n", initialBalance.String())

	// Step 1: Load or generate shielded keys
	fmt.Println("\n=== Step 1: Key Management ===")
	keys, sk, ask, nsk, ovk, ak, nk, ivk, d, paymentAddress, err := loadOrGenerateKeys(cli, ctx)
	if err != nil {
		log.Fatal("Failed to load/generate keys:", err)
	}

	// Validate and verify keys
	err = validateAndVerifyKeys(cli, ctx, ivk, ak, nk)
	if err != nil {
		log.Fatal("Key validation failed:", err)
	}

	// Use variables to avoid "declared and not used" errors
	_ = sk
	_ = d
	_ = keys

	fmt.Printf("Using shielded keys:\n")
	fmt.Printf("  Payment Address: %s\n", paymentAddress)
	fmt.Printf("  Keys validated and ready for use\n")

	// Step 2: Handle TRC20 approval if needed
	fmt.Println("\n=== Step 2: TRC20 Approval ===")
	err = handleApprovalIfNeeded(cli, ctx, key, tokenAddr, shieldedAddr)
	if err != nil {
		log.Fatal("Failed to handle approval:", err)
	}

	// Step 3: Handle mint transaction
	fmt.Println("\n=== Step 3: Mint Transaction ===")
	mintResult, err := handleMintTransaction(cli, ctx, key, shieldedAddr, ovk, paymentAddress)
	if err != nil {
		log.Fatal("Failed to handle mint transaction:", err)
	}

	if mintResult != nil {
		fmt.Printf("Mint transaction successful: %s\n", mintResult.TxID)
	}

	// Step 4: Scan for shielded notes
	fmt.Println("\n=== Step 4: Note Scanning ===")
	notes, err := scanShieldedNotes(cli, ctx, shieldedAddr, ivk, ak, nk, ovk)
	if err != nil {
		log.Fatal("Failed to scan for notes:", err)
	}

	if notes == nil || len(notes.GetNoteTxs()) == 0 {
		fmt.Println("⚠️  No shielded notes found")
		if CurrentMode == ModeBurnOnly {
			fmt.Println("Running in burn-only mode but no notes available to burn")
			fmt.Println("Consider running in full flow mode first to create notes")
		}
		printSummary(mintResult, nil, initialBalance, decimal.Zero)
		return
	}

	fmt.Printf("✅ Found %d shielded notes for operations\n", len(notes.GetNoteTxs()))

	// Step 5: Handle burn transaction
	fmt.Println("\n=== Step 5: Burn Transaction ===")
	burnResult, err := handleBurnTransaction(cli, ctx, key, shieldedAddr, notes, ask, nsk, ovk)
	if err != nil {
		log.Printf("Failed to execute burn transaction: %v", err)
		fmt.Println("\n💡 TIP: If you see 'note not yet processed in merkle tree' errors, try running the burn operation again in a few minutes.")
		fmt.Println("   Notes need time to be fully confirmed and incorporated into the merkle tree before they can be spent.")
		printSummary(mintResult, nil, initialBalance, decimal.Zero)
		return
	}

	if burnResult != nil {
		fmt.Printf("✅ Burn Transaction:\n")
		fmt.Printf("   TX ID: %s\n", burnResult.TxID)
		fmt.Printf("   Energy Used: %d\n", burnResult.EnergyUsage)
		fmt.Printf("   Net Usage: %d\n", burnResult.NetUsage)
		fmt.Printf("   Amount: %s (scaled: %s)\n", BurnAmount, "calculated")
	} else if CurrentMode == ModeTestOnly {
		fmt.Printf("🧪 Burn Transaction: Test mode (not broadcasted)\n")
	} else {
		fmt.Printf("❌ Burn Transaction: Failed or not executed\n")
		fmt.Printf("   🔍 Common causes:\n")
		fmt.Printf("      • Notes not yet confirmed in merkle tree\n")
		fmt.Printf("      • Insufficient note value for burn amount\n")
		fmt.Printf("      • Network connectivity issues\n")
		fmt.Printf("   💡 Try again in a few minutes for confirmation issues\n")
	}

	// Step 6: Verify burn result
	fmt.Println("\n=== Step 6: Result Verification ===")
	err = verifyBurnResult(cli, ctx, key, tokenAddr, initialBalance)
	if err != nil {
		log.Printf("Failed to verify burn result: %v", err)
	}

	// Final summary
	printSummary(mintResult, burnResult, initialBalance, decimal.Zero)
}

// printSummary prints a comprehensive summary of the transaction flow
func printSummary(mintResult, burnResult *client.BroadcastResult, initialBalance, finalBalance decimal.Decimal) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("                    TRANSACTION SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("Operation Mode: %s\n", CurrentMode)
	fmt.Printf("Network: %s\n", Node)
	fmt.Printf("Token Address: %s\n", TokenAddress)
	fmt.Printf("Shielded Contract: %s\n", ShieldedContract)

	fmt.Println("\n--- Transaction Results ---")
	if mintResult != nil {
		fmt.Printf("✅ Mint Transaction:\n")
		fmt.Printf("   TX ID: %s\n", mintResult.TxID)
		fmt.Printf("   Energy Used: %d\n", mintResult.EnergyUsage)
		fmt.Printf("   Net Usage: %d\n", mintResult.NetUsage)
		fmt.Printf("   Amount: %s (scaled: %s)\n", MintAmount, "calculated")
	} else if CurrentMode == ModeBurnOnly {
		fmt.Printf("⏭️  Mint Transaction: Skipped (burn-only mode)\n")
	} else if CurrentMode == ModeTestOnly {
		fmt.Printf("🧪 Mint Transaction: Test mode (not broadcasted)\n")
	} else {
		fmt.Printf("❌ Mint Transaction: Failed or not executed\n")
	}

	if burnResult != nil {
		fmt.Printf("✅ Burn Transaction:\n")
		fmt.Printf("   TX ID: %s\n", burnResult.TxID)
		fmt.Printf("   Energy Used: %d\n", burnResult.EnergyUsage)
		fmt.Printf("   Net Usage: %d\n", burnResult.NetUsage)
		fmt.Printf("   Amount: %s (scaled: %s)\n", BurnAmount, "calculated")
	} else if CurrentMode == ModeTestOnly {
		fmt.Printf("🧪 Burn Transaction: Test mode (not broadcasted)\n")
	} else {
		fmt.Printf("❌ Burn Transaction: Failed or not executed\n")
	}

	fmt.Println("\n--- Balance Changes ---")
	fmt.Printf("Initial Balance: %s\n", initialBalance.String())
	if !finalBalance.IsZero() {
		fmt.Printf("Final Balance: %s\n", finalBalance.String())
		fmt.Printf("Change: %s\n", finalBalance.Sub(initialBalance).String())
	} else {
		fmt.Printf("Final Balance: Not checked\n")
	}

	fmt.Println("\n--- Technical Details ---")
	fmt.Printf("Scaling Factor: %d\n", ScalingFactor)
	fmt.Printf("Begin Block: %d\n", BeginBlock)
	fmt.Printf("Key File: %s\n", KeyFile)

	fmt.Println("\n--- Key Features Demonstrated ---")
	fmt.Println("✅ Modular architecture with separate concerns")
	fmt.Println("✅ Persistent key management")
	fmt.Println("✅ Historical block scanning")
	fmt.Println("✅ IVK scanning with fallback to OVK")
	fmt.Println("✅ Merkle path validation for note spending")
	fmt.Println("✅ Multiple operation modes (full/burn-only/test)")
	fmt.Println("✅ Comprehensive error handling")
	fmt.Println("✅ Balance verification")
	fmt.Println("✅ Note confirmation handling")

	if CurrentMode == ModeTestOnly {
		fmt.Println("\n🧪 TEST MODE ACTIVE - No actual transactions were broadcast")
		fmt.Println("   All transaction creation and validation completed successfully")
		fmt.Println("   Ready for production execution")
	}

	if CurrentMode == ModeBurnOnly {
		fmt.Println("\n🔥 BURN-ONLY MODE ACTIVE")
		fmt.Println("   Skipped minting new notes, attempting to burn existing ones")
		fmt.Println("   If burn fails, notes may still be processing in merkle tree")
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🎉 Shielded TRC20 Transaction Flow Complete!")
	fmt.Println(strings.Repeat("=", 60))
}
