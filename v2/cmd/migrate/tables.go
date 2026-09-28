package main

// v2Pkgs lists the v2 package directories scanned for symbols: the curated
// packages plus the root facade ("." — this module's root, package tronlib).
var v2Pkgs = []string{"tron", "key", "event", "rpc", "tx", "contract", "token", "."}

// removedPaths scopes C3 removals by source file. v1 has no pkg/shielded
// package — every C3-excluded shielded function lives in the one lowlevel
// file, so the file is the removal unit.
var removedPaths = []string{"lowlevel/shielded.go"}

// removedPrefixes scopes C3 removals by name. TRC-10 *issuance* is excluded;
// TRC-10 *transfer* is in scope (spec §13: tx.AssetTx / TransferToken), and
// "TransferAsset" does not match this prefix.
var removedPrefixes = []string{"AssetIssue"}

// renames maps a verified v1 key to its v2 key. classify fails closed when a
// target is absent from the scanned v2 tree, so a dead mapping can never reach
// the guide. Entries are seeded only after grep-verifying the target exists
// (spec §5.3; every entry carries a reason). Empty until Task 4 seeds it.
var renames = map[string]string{}

func curatedInputs() Inputs {
	return Inputs{
		RemovedPaths:    removedPaths,
		RemovedPrefixes: removedPrefixes,
		Renames:         renames,
	}
}
