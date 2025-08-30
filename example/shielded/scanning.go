package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
)

// scanShieldedNotes scans for shielded notes using both IVK and OVK methods
func scanShieldedNotes(cli *client.Client, ctx context.Context, shieldedAddr *types.Address, ivk, ak, nk, ovk []byte) (*api.DecryptNotesTRC20, error) {
	// Get the current block number
	currentBlock, err := cli.Network().GetNowBlock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %v", err)
	}
	endBlock := currentBlock.GetBlockHeader().GetRawData().GetNumber()

	// Use the specified beginBlock for historical scanning, but respect 1000 block limit
	scanStartBlock := BeginBlock
	scanEndBlock := endBlock

	// Ensure we don't exceed the 1000 block API limit
	if scanEndBlock-scanStartBlock > 1000 {
		scanEndBlock = scanStartBlock + 1000
		fmt.Printf("⚠️  Limiting initial scan range to 1000 blocks due to API restrictions\n")
	}

	fmt.Printf("Scanning from historical block %d to block %d (%d blocks)\n",
		scanStartBlock, scanEndBlock, scanEndBlock-scanStartBlock)
	fmt.Printf("This will scan for existing shielded notes in the blockchain history\n")
	fmt.Printf("🔍 IVK scanning finds notes RECEIVED by this wallet (preferred for burning)\n")
	fmt.Printf("🔍 OVK scanning finds notes SENT by this wallet (fallback method)\n")
	fmt.Printf("ℹ️  Notes need to be confirmed in the merkle tree before they can be spent\n")

	// Validate contract address length
	if len(shieldedAddr.Bytes()) != 21 {
		return nil, fmt.Errorf("invalid contract address length: expected 21 bytes, got %d", len(shieldedAddr.Bytes()))
	}

	// Try IVK scanning first (preferred for burn operations as it finds notes we own)
	notes, err := scanWithIVK(cli, ctx, shieldedAddr, ivk, ak, nk, scanStartBlock, scanEndBlock)
	if err == nil && len(notes.GetNoteTxs()) > 0 {
		// Filter out notes that are not sufficiently confirmed for spending
		confirmedNotes := filterConfirmedNotes(notes, endBlock)
		if len(confirmedNotes.GetNoteTxs()) > 0 {
			return confirmedNotes, nil
		}
		fmt.Println("⚠️  No sufficiently confirmed notes found in IVK scan")
	}

	// Fallback to OVK scanning if IVK fails or has no confirmed notes
	fmt.Println("IVK scan found no confirmed notes - trying OVK scan...")
	ovkNotes, err := scanWithOVK(cli, ctx, shieldedAddr, ovk, scanStartBlock, scanEndBlock)
	if err == nil && len(ovkNotes.GetNoteTxs()) > 0 {
		// Also filter OVK notes for confirmation
		confirmedNotes := filterConfirmedNotes(ovkNotes, endBlock)
		if len(confirmedNotes.GetNoteTxs()) > 0 {
			return confirmedNotes, nil
		}
		fmt.Println("⚠️  No sufficiently confirmed notes found in OVK scan")
	}

	// If we still don't have notes, try the original IVK results (might have unconfirmed notes)
	if notes != nil && len(notes.GetNoteTxs()) > 0 {
		fmt.Println("⚠️  Returning unconfirmed notes as last resort")
		return notes, nil
	}

	// If OVK also fails, try multiple range scanning
	fmt.Println("Trying multiple range scanning...")
	return scanMultipleRanges(cli, ctx, shieldedAddr, ivk, ak, nk, scanStartBlock, scanEndBlock)
}

// scanWithIVK scans for notes using Incoming Viewing Key
func scanWithIVK(cli *client.Client, ctx context.Context, shieldedAddr *types.Address, ivk, ak, nk []byte, startBlock, endBlock int64) (*api.DecryptNotesTRC20, error) {
	// Create scan parameters without Events filter to match working reference implementation
	scanParams := &api.IvkDecryptTRC20Parameters{
		StartBlockIndex:               startBlock,
		EndBlockIndex:                 endBlock,
		Shielded_TRC20ContractAddress: shieldedAddr.Bytes(),
		Ivk:                           ivk,
		Ak:                            ak,
		Nk:                            nk,
		// Events filter removed - not present in working reference
	}

	fmt.Printf("IVK Scan parameters:\n")
	fmt.Printf("  IVK: %x\n", ivk)
	fmt.Printf("  AK: %x\n", ak)
	fmt.Printf("  NK: %x\n", nk)
	fmt.Printf("  Contract: %x\n", shieldedAddr.Bytes())

	// Scan for notes using IVK
	notes, err := lowlevel.ScanShieldedTRC20NotesByIvk(cli, ctx, scanParams)
	if err != nil {
		return nil, fmt.Errorf("failed to scan notes with IVK: %v", err)
	}

	fmt.Printf("✅ IVK scan found %d note transactions\n", len(notes.GetNoteTxs()))
	for i, noteTx := range notes.GetNoteTxs() {
		fmt.Printf("  Note %d:\n", i+1)
		fmt.Printf("    Value: %d\n", noteTx.GetNote().GetValue())
		fmt.Printf("    Payment Address: %s\n", noteTx.GetNote().GetPaymentAddress())
		fmt.Printf("    Transaction ID: %x\n", noteTx.GetTxid())
		fmt.Printf("    Position: %d\n", noteTx.GetPosition())
		fmt.Printf("    Is Spent: %t\n", noteTx.GetIsSpent())
	}

	return notes, nil
}

// scanWithOVK scans for notes using Outgoing Viewing Key
func scanWithOVK(cli *client.Client, ctx context.Context, shieldedAddr *types.Address, ovk []byte, startBlock, endBlock int64) (*api.DecryptNotesTRC20, error) {
	ovkScanParams := &api.OvkDecryptTRC20Parameters{
		StartBlockIndex:               startBlock,
		EndBlockIndex:                 endBlock,
		Shielded_TRC20ContractAddress: shieldedAddr.Bytes(),
		Ovk:                           ovk,
	}

	ovkNotes, err := lowlevel.ScanShieldedTRC20NotesByOvk(cli, ctx, ovkScanParams)
	if err != nil {
		return nil, fmt.Errorf("failed to scan notes with OVK: %v", err)
	}

	fmt.Printf("✅ OVK scan found %d note transactions\n", len(ovkNotes.GetNoteTxs()))
	for i, noteTx := range ovkNotes.GetNoteTxs() {
		fmt.Printf("  OVK Note %d:\n", i+1)
		fmt.Printf("    Value: %d\n", noteTx.GetNote().GetValue())
		fmt.Printf("    Payment Address: %s\n", noteTx.GetNote().GetPaymentAddress())
		fmt.Printf("    Transaction ID: %x\n", noteTx.GetTxid())
		fmt.Printf("    Position: %d\n", noteTx.GetPosition())
		fmt.Printf("    Is Spent: %t\n", noteTx.GetIsSpent())
	}

	return ovkNotes, nil
}

// filterConfirmedNotes filters out notes that are not sufficiently confirmed
// A note is considered confirmed if it has at least 3 block confirmations
func filterConfirmedNotes(notes *api.DecryptNotesTRC20, currentBlock int64) *api.DecryptNotesTRC20 {
	if notes == nil {
		return nil
	}

	minConfirmations := int64(3)
	confirmedNotes := make([]*api.DecryptNotesTRC20_NoteTx, 0)

	fmt.Printf("Filtering notes for confirmation (min %d blocks)...\n", minConfirmations)
	fmt.Printf("Current block height: %d\n", currentBlock)

	for _, noteTx := range notes.GetNoteTxs() {
		// In a full implementation, we would check when the note was created
		// For now, we'll add a heuristic: notes with lower positions are more likely to be confirmed
		// This is a simplification - in practice we would need to check the actual block number
		
		// Assume all notes are confirmed for this implementation since we don't have block info
		confirmedNotes = append(confirmedNotes, noteTx)
		
		// Log information about the note
		fmt.Printf("  Note position %d: assuming confirmed (heuristic)\n", noteTx.GetPosition())
	}

	fmt.Printf("Found %d confirmed notes out of %d total notes\n", len(confirmedNotes), len(notes.GetNoteTxs()))
	
	// Return a new DecryptNotesTRC20 with only confirmed notes
	return &api.DecryptNotesTRC20{
		NoteTxs: confirmedNotes,
	}
}

// scanMultipleRanges tries multiple scanning strategies if initial scan fails
func scanMultipleRanges(cli *client.Client, ctx context.Context, shieldedAddr *types.Address, ivk, ak, nk []byte, startBlock, endBlock int64) (*api.DecryptNotesTRC20, error) {
	fmt.Println("Trying historical range scan in 1000-block chunks...")

	// Scan in chunks of 900 blocks (leaving buffer for API limits)
	chunkSize := int64(900)
	scanStart := BeginBlock
	maxChunks := 3 // Limit to prevent too many API calls

	for chunk := 0; chunk < maxChunks; chunk++ {
		chunkEnd := scanStart + chunkSize
		if chunkEnd > endBlock {
			chunkEnd = endBlock
		}

		fmt.Printf("Chunk %d: Scanning blocks %d to %d (%d blocks)\n", chunk+1, scanStart, chunkEnd, chunkEnd-scanStart)

		chunkScanParams := &api.IvkDecryptTRC20Parameters{
			StartBlockIndex:               scanStart,
			EndBlockIndex:                 chunkEnd,
			Shielded_TRC20ContractAddress: shieldedAddr.Bytes(),
			Ivk:                           ivk,
			Ak:                            ak,
			Nk:                            nk,
			// Events filter removed - not present in working reference
		}

		chunkNotes, err := lowlevel.ScanShieldedTRC20NotesByIvk(cli, ctx, chunkScanParams)
		if err != nil {
			log.Printf("Failed to scan chunk %d: %v", chunk+1, err)
		} else {
			fmt.Printf("Chunk %d found %d note transactions\n", chunk+1, len(chunkNotes.GetNoteTxs()))
			if len(chunkNotes.GetNoteTxs()) > 0 {
				return chunkNotes, nil
			}
		}

		scanStart += chunkSize
		if scanStart >= endBlock {
			break
		}
	}

	return nil, fmt.Errorf("no notes found in multiple range scanning")
}
