package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"sort"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/shopspring/decimal"
)

// handleApprovalIfNeeded handles the TRC20 approval process if needed
func handleApprovalIfNeeded(cli *client.Client, ctx context.Context, key *signer.PrivateKeySigner, tokenAddr, shieldedAddr *types.Address) error {
	if CurrentMode == ModeBurnOnly {
		fmt.Println("\n⏭️  Skipping approval (burn-only mode)")
		return nil
	}

	// Create TRC20 manager
	trc20Mgr := cli.TRC20(tokenAddr)
	if trc20Mgr == nil {
		return fmt.Errorf("failed to create TRC20 manager")
	}

	from := key.Address()

	// Check allowance
	fmt.Println("\nChecking allowance...")
	allowance, err := trc20Mgr.Allowance(ctx, from, shieldedAddr)
	if err != nil {
		return fmt.Errorf("failed to check allowance: %v", err)
	}
	fmt.Printf("Current allowance: %s\n", allowance.String())

	// Check balance
	balance, err := trc20Mgr.BalanceOf(ctx, from)
	if err != nil {
		return fmt.Errorf("failed to check balance: %v", err)
	}
	fmt.Printf("Current balance: %s\n", balance.String())

	// Convert mint amount to decimal for comparison
	mintAmountDecimal, err := decimal.NewFromString(MintAmount)
	if err != nil {
		return fmt.Errorf("failed to parse mint amount: %v", err)
	}

	// Divide by 10^6 to get the actual token amount (6 decimals)
	mintAmountTokens := mintAmountDecimal.Div(decimal.NewFromInt(1000000))

	// Check if we need to approve the shielded contract
	if allowance.LessThan(mintAmountTokens) {
		fmt.Println("\nApproving shielded contract to spend tokens...")

		if CurrentMode == ModeTestOnly {
			fmt.Println("🧪 TEST MODE: Would create approval transaction")
			return nil
		}

		approveTx, err := trc20Mgr.Approve(ctx, from, shieldedAddr, mintAmountTokens.Mul(decimal.NewFromInt(100)))
		if err != nil {
			return fmt.Errorf("failed to create approve transaction: %v", err)
		}

		// Use SignAndBroadcast method
		opts := client.DefaultBroadcastOptions()
		opts.FeeLimit = 50_000_000
		opts.WaitForReceipt = true

		result, err := cli.SignAndBroadcast(ctx, approveTx, opts, key)
		if err != nil {
			return fmt.Errorf("failed to broadcast approve transaction: %v", err)
		}

		if !result.Success {
			return fmt.Errorf("approve transaction failed: %s", result.Message)
		}

		fmt.Printf("Approve transaction broadcasted: %s\n", result.TxID)

		// Wait for confirmation
		time.Sleep(10 * time.Second)
	} else {
		fmt.Println("Allowance is sufficient, no approval needed")
	}

	return nil
}

// handleMintTransaction handles the minting process
func handleMintTransaction(cli *client.Client, ctx context.Context, key *signer.PrivateKeySigner, shieldedAddr *types.Address, ovk []byte, paymentAddress string) (*client.BroadcastResult, error) {
	if CurrentMode == ModeBurnOnly {
		fmt.Println("\n⏭️  Skipping mint transaction (burn-only mode)")
		return nil, nil
	}

	fmt.Println("\nGenerating mint parameters...")

	// Generate rcm (random commitment)
	rcmResp, err := lowlevel.GetRcm(cli, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate rcm: %v", err)
	}
	rcm := rcmResp.GetValue()
	fmt.Printf("Generated rcm: %x\n", rcm)

	// Calculate scaled values for minting
	mintAmountInt, ok := new(big.Int).SetString(MintAmount, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse mint amount")
	}

	// Calculate scaled value: from_amount = value * scalingFactor
	// So value = from_amount / scalingFactor
	scaledMintValue := new(big.Int).Div(mintAmountInt, big.NewInt(ScalingFactor))
	fmt.Printf("Mint amount (from_amount): %s\n", MintAmount)
	fmt.Printf("Scaled mint value: %s (scaling factor: %d)\n", scaledMintValue.String(), ScalingFactor)

	// Create mint parameters
	mintParams := &api.PrivateShieldedTRC20Parameters{
		Ovk:            ovk,
		FromAmount:     MintAmount,
		ShieldedSpends: []*api.SpendNoteTRC20{}, // Empty for minting
		ShieldedReceives: []*api.ReceiveNote{
			{
				Note: &api.Note{
					Value:          scaledMintValue.Int64(),
					PaymentAddress: paymentAddress,
					Rcm:            rcm,
				},
			},
		},
		Shielded_TRC20ContractAddress: shieldedAddr.Bytes(),
	}

	fmt.Printf("Mint parameters:\n")
	fmt.Printf("  From Amount: %s\n", mintParams.FromAmount)
	fmt.Printf("  Shielded Contract: %s\n", ShieldedContract)
	fmt.Printf("  Payment Address: %s\n", paymentAddress)
	fmt.Printf("  Scaled Value: %d\n", scaledMintValue.Int64())

	// Create shielded contract parameters for mint
	mintResult, err := lowlevel.CreateShieldedContractParameters(cli, ctx, mintParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create mint parameters: %v", err)
	}
	fmt.Printf("Created mint parameters successfully\n")

	if CurrentMode == ModeTestOnly {
		fmt.Println("🧪 TEST MODE: Would broadcast mint transaction")
		return &client.BroadcastResult{
			Success: true,
			TxID:    "test-mint-tx-id",
		}, nil
	}

	// Execute mint transaction
	fmt.Println("Executing mint transaction...")

	// Prepare the function selector and trigger input for mint
	// For mint, the function selector is "855d175e"
	triggerContractInput := "855d175e" + mintResult.GetTriggerContractInput()
	fmt.Printf("Trigger contract input length: %d\n", len(triggerContractInput))

	// Decode the hex string to bytes for the contract data
	triggerData, err := hex.DecodeString(triggerContractInput)
	if err != nil {
		return nil, fmt.Errorf("failed to decode trigger contract input: %v", err)
	}

	from := key.Address()

	// Build the transaction using TriggerContract
	mintTx, err := lowlevel.TriggerContract(cli, ctx, &core.TriggerSmartContract{
		OwnerAddress:    from.Bytes(),
		ContractAddress: shieldedAddr.Bytes(),
		Data:            triggerData,
		CallValue:       0, // call_value: 0 TRX
		CallTokenValue:  0,
		TokenId:         0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mint transaction: %v", err)
	}

	// Set transaction options
	opts := client.DefaultBroadcastOptions()
	opts.FeeLimit = 350_000_000 // 350 TRX fee limit
	opts.WaitForReceipt = true

	// Sign and broadcast the mint transaction
	mintTxResult, err := cli.SignAndBroadcast(ctx, mintTx, opts, key)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast mint transaction: %v", err)
	}

	if !mintTxResult.Success {
		return nil, fmt.Errorf("mint transaction failed: %s", mintTxResult.Message)
	}

	fmt.Printf("✅ Mint transaction successful!\n")
	fmt.Printf("Transaction ID: %s\n", mintTxResult.TxID)
	fmt.Printf("Energy Used: %d\n", mintTxResult.EnergyUsage)
	fmt.Printf("Net Usage: %d\n", mintTxResult.NetUsage)

	return mintTxResult, nil
}

// selectNoteForBurning selects the best note for burning and gets its merkle path
// using the smartcontract package for more reliable results
func selectNoteForBurning(cli *client.Client, ctx context.Context, key *signer.PrivateKeySigner, shieldedAddr *types.Address, notes *api.DecryptNotesTRC20, burnAmount string) (*api.DecryptNotesTRC20_NoteTx, []byte, []byte, error) {
	// Convert burn amount to big.Int for comparison
	burnAmountInt, ok := new(big.Int).SetString(burnAmount, 10)
	if !ok {
		return nil, nil, nil, fmt.Errorf("failed to parse burn amount")
	}

	// Calculate scaled value for burning
	scaledBurnValue := new(big.Int).Div(burnAmountInt, big.NewInt(ScalingFactor))

	fmt.Printf("Burn requirements:\n")
	fmt.Printf("  Burn amount: %s\n", burnAmount)
	fmt.Printf("  Scaled burn value needed: %s\n", scaledBurnValue.String())
	fmt.Printf("  Scaling factor: %d\n", ScalingFactor)

	// Collect all suitable notes for burning
	var suitableNotes []*api.DecryptNotesTRC20_NoteTx
	var suitableIndices []int

	fmt.Println("\nAnalyzing available notes (preferring older notes):")
	availableNotes := notes.GetNoteTxs()

	// Sort notes by position (older notes have lower positions)
	sort.Slice(availableNotes, func(i, j int) bool {
		return availableNotes[i].GetPosition() < availableNotes[j].GetPosition()
	})

	for i, noteTx := range availableNotes {
		fmt.Printf("  Note %d: Value=%d, IsSpent=%t, Position=%d, TxID=%x\n",
			i+1, noteTx.GetNote().GetValue(), noteTx.GetIsSpent(),
			noteTx.GetPosition(), noteTx.GetTxid())

		// Check if note is unspent and has sufficient value
		if !noteTx.GetIsSpent() && noteTx.GetNote().GetValue() >= scaledBurnValue.Int64() {
			suitableNotes = append(suitableNotes, noteTx)
			suitableIndices = append(suitableIndices, i)
			fmt.Printf("    ✅ Suitable for burning (sufficient value: %d >= %d, position: %d)\n",
				noteTx.GetNote().GetValue(), scaledBurnValue.Int64(), noteTx.GetPosition())
		} else if noteTx.GetIsSpent() {
			fmt.Printf("    ❌ Note already spent\n")
		} else {
			fmt.Printf("    ❌ Insufficient value (%d < %d)\n",
				noteTx.GetNote().GetValue(), scaledBurnValue.Int64())
		}
	}

	if len(suitableNotes) == 0 {
		return nil, nil, nil, fmt.Errorf("no suitable notes found for burning")
	}

	fmt.Printf("\n🔍 Found %d suitable notes, trying to find one with valid merkle path...\n", len(suitableNotes))

	from := key.Address()

	// Try each suitable note until we find one with a valid merkle path
	// Load the ABI for the shielded contract
	abiBytes, err := os.ReadFile("/home/kslam/goproj/tronlib/cmd/setup_nile_testnet/test_contract/build/ShieldedTRC20.abi")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read shielded contract ABI: %v", err)
	}
	abiString := string(abiBytes)

	// Create a smart contract instance
	contractInstance, err := smartcontract.NewInstance(cli, shieldedAddr, abiString)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create contract instance: %v", err)
	}

	for i, noteTx := range suitableNotes {
		fmt.Printf("\n🔍 Trying note %d (position %d)...\n", suitableIndices[i]+1, noteTx.GetPosition())

		// Validate the note position before attempting to get path
		if err := validateNotePosition(noteTx.GetPosition()); err != nil {
			fmt.Printf("❌ Invalid note position %d: %v\n", noteTx.GetPosition(), err)
			continue
		}

		// Call getPath function on the shielded contract to get merkle path
		// Using the smartcontract package for better reliability
		fmt.Printf("Calling getPath with position: %d\n", noteTx.GetPosition())

		// Convert position to *big.Int for uint256 parameter
		positionBigInt := big.NewInt(noteTx.GetPosition())

		// Use the Call method for constant functions
		result, err := contractInstance.Call(ctx, from, "getPath", positionBigInt)
		if err != nil {
			fmt.Printf("❌ Failed to get merkle path for note %d: %v\n", i+1, err)
			continue // Try next note
		}

		fmt.Printf("Raw path result: %+v\n", result)
		fmt.Printf("Raw path result type: %T\n", result)

		// The result should be a slice with two elements:
		// 1. The root (bytes32)
		// 2. The path (bytes32[32])
		resultSlice, ok := result.([]interface{})
		if !ok || len(resultSlice) != 2 {
			fmt.Printf("❌ Invalid path result format for note %d\n", i+1)
			fmt.Printf("Expected []interface{} with 2 elements, got %T with length %d\n", result, len(resultSlice))
			continue // Try next note
		}

		// Extract root and path from the result
		rootInterface, pathInterface := resultSlice[0], resultSlice[1]

		fmt.Printf("Root type: %T\n", rootInterface)
		fmt.Printf("Path type: %T\n", pathInterface)

		// Convert root to bytes
		var rootBytes []byte
		switch r := rootInterface.(type) {
		case [32]byte:
			rootBytes = r[:]
			fmt.Printf("Root is [32]byte: %x\n", rootBytes)
		case []byte:
			if len(r) >= 32 {
				rootBytes = r[:32]
				fmt.Printf("Root is []byte: %x\n", rootBytes)
			} else {
				fmt.Printf("❌ Invalid root format for note %d - []byte too short: %d\n", i+1, len(r))
				continue // Try next note
			}
		default:
			fmt.Printf("❌ Invalid root format for note %d - unexpected type: %T\n", i+1, r)
			continue // Try next note
		}

		// Convert path to bytes
		var pathBytes []byte
		switch p := pathInterface.(type) {
		case [][32]byte:
			fmt.Printf("Path is [][32]byte with %d elements\n", len(p))
			for _, pathElement := range p {
				pathBytes = append(pathBytes, pathElement[:]...)
			}
		case [][]byte:
			fmt.Printf("Path is [][]byte with %d elements\n", len(p))
			for _, pathElement := range p {
				if len(pathElement) >= 32 {
					pathBytes = append(pathBytes, pathElement[:32]...)
				}
			}
		case [32][32]byte: // This is the type we're seeing
			fmt.Printf("Path is [32][32]byte with %d elements\n", len(p))
			for _, pathElement := range p {
				pathBytes = append(pathBytes, pathElement[:]...)
			}
		case []interface{}:
			fmt.Printf("Path is []interface{} with %d elements\n", len(p))
			for _, pathElement := range p {
				switch pe := pathElement.(type) {
				case [32]byte:
					pathBytes = append(pathBytes, pe[:]...)
				case []byte:
					if len(pe) >= 32 {
						pathBytes = append(pathBytes, pe[:32]...)
					}
				default:
					fmt.Printf("❌ Invalid path element type: %T\n", pe)
				}
			}
		default:
			fmt.Printf("❌ Invalid path format for note %d - unexpected type: %T\n", i+1, p)
			continue // Try next note
		}

		fmt.Printf("✅ Found valid merkle path for note %d!\n", i+1)
		fmt.Printf("  Root: %x\n", rootBytes)
		fmt.Printf("  Path length: %d bytes\n", len(pathBytes))

		// Validate path structure
		if len(pathBytes)%32 != 0 {
			fmt.Printf("⚠️  Warning: Path length (%d) is not a multiple of 32 bytes\n", len(pathBytes))
		}

		pathElements := len(pathBytes) / 32
		fmt.Printf("  Path elements: %d\n", pathElements)

		// Success! Return this note with its merkle path
		return noteTx, rootBytes, pathBytes, nil
	}

	return nil, nil, nil, fmt.Errorf("no notes with valid merkle paths found")
}

// getCurrentBlockHeight gets the current block height from the network
func getCurrentBlockHeight(cli *client.Client, ctx context.Context) int64 {
	block, err := cli.Network().GetNowBlock(ctx)
	if err != nil {
		fmt.Printf("Warning: Could not get current block height: %v\n", err)
		return -1
	}
	return block.GetBlockHeader().GetRawData().GetNumber()
}

// validateNotePosition checks if a note position is valid and likely to have a merkle path
func validateNotePosition(position int64) error {
	// Basic validation - position should be non-negative
	if position < 0 {
		return fmt.Errorf("invalid note position: %d (must be >= 0)", position)
	}

	// In a real implementation, we might check against the current tree size
	// For now, we'll just ensure it's a reasonable value
	if position > 1000000000 { // Arbitrary large number check
		return fmt.Errorf("note position seems too large: %d", position)
	}

	return nil
}

// waitForNoteConfirmation waits for a note to be confirmed in the merkle tree
// This is a simplified implementation - in practice you might want to check the actual merkle tree size
func waitForNoteConfirmation(cli *client.Client, ctx context.Context, notePosition int64, maxWaitTime time.Duration) error {
	fmt.Printf("⏳ Waiting for note at position %d to be confirmed in merkle tree (max wait: %v)...\n", notePosition, maxWaitTime)

	startTime := time.Now()
	checkInterval := 10 * time.Second

	for time.Since(startTime) < maxWaitTime {
		// In a real implementation, we would check the merkle tree size or note status
		// For now, we'll just wait and then try again
		time.Sleep(checkInterval)
		fmt.Printf("    Still waiting... (%v elapsed)\n", time.Since(startTime))
	}

	fmt.Printf("⚠️  Wait time exceeded. Note may still be processing.\n")
	return nil // Don't fail completely, let the caller try anyway
}

// handleBurnTransaction handles the complete burn process using the smartcontract package
func handleBurnTransaction(cli *client.Client, ctx context.Context, key *signer.PrivateKeySigner, shieldedAddr *types.Address, notes *api.DecryptNotesTRC20, ask, nsk, ovk []byte) (*client.BroadcastResult, error) {
	fmt.Println("\n\xE2\x9B\x85 Starting burn transaction process...")

	// Check if we have any notes to burn
	if notes == nil || len(notes.GetNoteTxs()) == 0 {
		return nil, fmt.Errorf("no shielded notes available for burning")
	}

	fmt.Printf("Found %d shielded notes to analyze for burning!\n", len(notes.GetNoteTxs()))

	// Validate note positions before proceeding
	fmt.Println("Validating note positions...")
	validNotes := make([]*api.DecryptNotesTRC20_NoteTx, 0)
	for _, noteTx := range notes.GetNoteTxs() {
		if err := validateNotePosition(noteTx.GetPosition()); err != nil {
			fmt.Printf("⚠️  Skipping note at position %d: %v\n", noteTx.GetPosition(), err)
			continue
		}
		validNotes = append(validNotes, noteTx)
	}

	if len(validNotes) == 0 {
		return nil, fmt.Errorf("no notes with valid positions found")
	}

	// Create a temporary notes object with only valid notes
	tempNotes := &api.DecryptNotesTRC20{
		NoteTxs: validNotes,
	}

	// Select the best note for burning
	selectedNoteTx, root, path, err := selectNoteForBurning(cli, ctx, key, shieldedAddr, tempNotes, BurnAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to select note for burning: %v", err)
	}

	fmt.Printf("\n✅ Using note for burning:\n")
	fmt.Printf("  Value: %d\n", selectedNoteTx.GetNote().GetValue())
	fmt.Printf("  Payment Address: %s\n", selectedNoteTx.GetNote().GetPaymentAddress())
	fmt.Printf("  Transaction ID: %x\n", selectedNoteTx.GetTxid())
	fmt.Printf("  Position: %d\n", selectedNoteTx.GetPosition())

	// Generate alpha for spending
	fmt.Println("\nGenerating spending parameters...")
	alphaResp, err := lowlevel.GetRcm(cli, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate alpha: %v", err)
	}
	alpha := alphaResp.GetValue()

	// Create spend note
	spendNote := &api.SpendNoteTRC20{
		Note: &api.Note{
			Value:          selectedNoteTx.GetNote().GetValue(),
			PaymentAddress: selectedNoteTx.GetNote().GetPaymentAddress(),
			Rcm:            selectedNoteTx.GetNote().GetRcm(),
		},
		Alpha: alpha,
		Root:  root,
		Path:  path,
		Pos:   selectedNoteTx.GetPosition(),
	}

	from := key.Address()

	// Create burn parameters
	fmt.Println("\nCreating burn parameters...")
	burnParams := &api.PrivateShieldedTRC20Parameters{
		Ask:                           ask,
		Nsk:                           nsk,
		Ovk:                           ovk,
		ShieldedSpends:                []*api.SpendNoteTRC20{spendNote},
		TransparentToAddress:          from.Bytes(),
		ToAmount:                      BurnAmount,
		Shielded_TRC20ContractAddress: shieldedAddr.Bytes(),
	}

	fmt.Printf("🔍 DETAILED BURN PARAMETERS DEBUG OUTPUT:\n")
	fmt.Printf("========================================\n")

	// Debug the main parameters
	fmt.Printf("1. Main Parameters:\n")
	fmt.Printf("   Ask (spending key): %x (len=%d, type=%T)\n", ask, len(ask), ask)
	fmt.Printf("   Nsk (nullifier key): %x (len=%d, type=%T)\n", nsk, len(nsk), nsk)
	fmt.Printf("   Ovk (outgoing viewing key): %x (len=%d, type=%T)\n", ovk, len(ovk), ovk)
	fmt.Printf("   ToAmount: %s (type=%T)\n", burnParams.ToAmount, burnParams.ToAmount)
	fmt.Printf("   TransparentToAddress: %x (len=%d, type=%T)\n", burnParams.TransparentToAddress, len(burnParams.TransparentToAddress), burnParams.TransparentToAddress)
	fmt.Printf("   Shielded_TRC20ContractAddress: %x (len=%d, type=%T)\n", burnParams.Shielded_TRC20ContractAddress, len(burnParams.Shielded_TRC20ContractAddress), burnParams.Shielded_TRC20ContractAddress)

	// Debug the spend note
	fmt.Printf("\n2. SpendNote Details:\n")
	if len(burnParams.ShieldedSpends) > 0 {
		spendNote := burnParams.ShieldedSpends[0]
		fmt.Printf("   Alpha (random commitment): %x (len=%d, type=%T)\n", spendNote.Alpha, len(spendNote.Alpha), spendNote.Alpha)
		fmt.Printf("   Root (merkle root): %x (len=%d, type=%T)\n", spendNote.Root, len(spendNote.Root), spendNote.Root)
		fmt.Printf("   Path (merkle path): %x (len=%d, type=%T)\n", spendNote.Path, len(spendNote.Path), spendNote.Path)
		fmt.Printf("   Pos (position): %d (type=%T)\n", spendNote.Pos, spendNote.Pos)

		if spendNote.Note != nil {
			fmt.Printf("   Note.Value: %d (type=%T)\n", spendNote.Note.Value, spendNote.Note.Value)
			fmt.Printf("   Note.PaymentAddress: %s (type=%T)\n", spendNote.Note.PaymentAddress, spendNote.Note.PaymentAddress)
			fmt.Printf("   Note.Rcm: %x (len=%d, type=%T)\n", spendNote.Note.Rcm, len(spendNote.Note.Rcm), spendNote.Note.Rcm)
		} else {
			fmt.Printf("   Note: nil\n")
		}
	} else {
		fmt.Printf("   No spend notes found!\n")
	}

	fmt.Printf("\n3. Additional Validation:\n")
	fmt.Printf("   ASK length: %d bytes\n", len(ask))
	fmt.Printf("   NSK length: %d bytes\n", len(nsk))
	fmt.Printf("   OVK length: %d bytes\n", len(ovk))
	fmt.Printf("   SpendNotes count: %d\n", len(burnParams.ShieldedSpends))
	fmt.Printf("   Path length: %d bytes\n", len(path))
	fmt.Printf("   Path elements: %d\n", len(path)/32)

	// Compare with expected values
	fmt.Printf("\n4. Expected vs Actual:\n")
	fmt.Printf("   Expected ASK length: 32, Actual: %d\n", len(ask))
	fmt.Printf("   Expected NSK length: 32, Actual: %d\n", len(nsk))
	fmt.Printf("   Expected OVK length: 32, Actual: %d\n", len(ovk))
	fmt.Printf("   Expected Alpha length: 32, Actual: %d\n", len(spendNote.Alpha))
	fmt.Printf("   Expected Root length: 32, Actual: %d\n", len(spendNote.Root))
	fmt.Printf("   Path length should be multiple of 32: %d\n", len(path))

	fmt.Printf("========================================\n")
	fmt.Printf("END DEBUG OUTPUT\n\n")

	// Create burn contract parameters
	fmt.Println("Generating burn contract parameters...")
	burnResult, err := lowlevel.CreateShieldedContractParameters(cli, ctx, burnParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create burn parameters: %v", err)
	}

	fmt.Printf("✅ Burn contract parameters generated successfully\n")
	fmt.Printf("  Trigger contract input length: %d\n", len(burnResult.GetTriggerContractInput()))

	if CurrentMode == ModeTestOnly {
		fmt.Println("🧪 TEST MODE: Would broadcast burn transaction")
		return &client.BroadcastResult{
			Success: true,
			TxID:    "test-burn-tx-id",
		}, nil
	}

	// Execute burn transaction using smartcontract package
	fmt.Println("\nExecuting burn transaction using smartcontract package...")

	// Load the ABI for the shielded contract
	abiBytes, err := os.ReadFile("/home/kslam/goproj/tronlib/cmd/setup_nile_testnet/test_contract/build/ShieldedTRC20.abi")
	if err != nil {
		return nil, fmt.Errorf("failed to read shielded contract ABI: %v", err)
	}
	abiString := string(abiBytes)

	// Create a smart contract instance
	contractInstance, err := smartcontract.NewInstance(cli, shieldedAddr, abiString)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract instance: %v", err)
	}

	// Get the trigger contract input from the burn result
	triggerInput := burnResult.GetTriggerContractInput()
	fmt.Printf("Trigger contract input: %s\n", triggerInput)

	// For the burn function, we need to parse the parameters correctly
	// The function selector for burn is "cc105875"
	// The parameters are complex, so we'll use a raw approach but with better error handling

	// Create the transaction data
	burnTriggerInput := "cc105875" + triggerInput
	burnTriggerData, err := hex.DecodeString(burnTriggerInput)
	if err != nil {
		return nil, fmt.Errorf("failed to decode burn trigger input: %v", err)
	}

	// Build burn transaction using the contract instance for better error handling
	// We'll use the Invoke method which is safer than the low-level TriggerContract
	burnTx, err := contractInstance.Invoke(ctx, from, 0, "burn", burnTriggerData)
	if err != nil {
		// If the Invoke method fails, fall back to the low-level approach with better logging
		fmt.Printf("⚠️  Failed to create burn transaction with smartcontract.Invoke: %v\n", err)
		fmt.Println("🔄 Falling back to low-level approach...")

		burnTx, err = lowlevel.TriggerContract(cli, ctx, &core.TriggerSmartContract{
			OwnerAddress:    from.Bytes(),
			ContractAddress: shieldedAddr.Bytes(),
			Data:            burnTriggerData,
			CallValue:       0,
			CallTokenValue:  0,
			TokenId:         0,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create burn transaction: %v", err)
		}
	}

	fmt.Printf("✅ Burn transaction created successfully\n")

	// Set transaction options
	opts := client.DefaultBroadcastOptions()
	opts.FeeLimit = 350_000_000 // 350 TRX fee limit
	opts.WaitForReceipt = true

	fmt.Printf("Transaction options: FeeLimit=%d, WaitForReceipt=%t\n", opts.FeeLimit, opts.WaitForReceipt)

	// Sign and broadcast burn transaction
	fmt.Println("🔐 Signing and broadcasting burn transaction...")
	burnTxResult, err := cli.SignAndBroadcast(ctx, burnTx, opts, key)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast burn transaction: %v", err)
	}

	fmt.Printf("📡 Burn transaction broadcasted: %s\n", burnTxResult.TxID)

	if !burnTxResult.Success {
		return nil, fmt.Errorf("burn transaction failed: %s", burnTxResult.Message)
	}

	fmt.Printf("✅ Burn transaction successful!\n")
	fmt.Printf("Transaction ID: %s\n", burnTxResult.TxID)
	fmt.Printf("Energy Used: %d\n", burnTxResult.EnergyUsage)
	fmt.Printf("Net Usage: %d\n", burnTxResult.NetUsage)

	// Wait for burn confirmation
	fmt.Println("⏳ Waiting for burn confirmation...")
	time.Sleep(15 * time.Second)

	return burnTxResult, nil
}

// verifyBurnResult checks if the burn was successful by checking balance changes
func verifyBurnResult(cli *client.Client, ctx context.Context, key *signer.PrivateKeySigner, tokenAddr *types.Address, initialBalance decimal.Decimal) error {
	// Create TRC20 manager to check balance
	trc20Mgr := cli.TRC20(tokenAddr)
	if trc20Mgr == nil {
		return fmt.Errorf("failed to create TRC20 manager")
	}

	from := key.Address()
	fmt.Println("\nVerifying burn result...")
	newBalance, err := trc20Mgr.BalanceOf(ctx, from)
	if err != nil {
		log.Printf("Failed to check new balance: %v", err)
		return err
	}

	fmt.Printf("Initial transparent balance: %s\n", initialBalance.String())
	fmt.Printf("New transparent balance: %s\n", newBalance.String())
	balanceIncrease := newBalance.Sub(initialBalance)
	fmt.Printf("Balance increase: %s\n", balanceIncrease.String())

	expectedIncrease, err := decimal.NewFromString(BurnAmount)
	if err != nil {
		return fmt.Errorf("failed to parse burn amount: %v", err)
	}
	expectedIncreaseTokens := expectedIncrease.Div(decimal.NewFromInt(1000000)) // Convert from 6 decimals

	if balanceIncrease.GreaterThanOrEqual(expectedIncreaseTokens) {
		fmt.Printf("✅ Burn verification successful! Expected: %s, Got: %s\n", expectedIncreaseTokens.String(), balanceIncrease.String())
	} else {
		fmt.Printf("⚠️  Balance increase (%s) is less than expected (%s)\n", balanceIncrease.String(), expectedIncreaseTokens.String())
	}

	return nil
}
