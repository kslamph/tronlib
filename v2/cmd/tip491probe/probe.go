package main

import "fmt"

// checkPenalty reports whether a simulated TIP-491 penalty verifies spec §7.5
// item 6, which asks whether a node's energy estimate already includes the
// dynamic-energy surcharge.
//
// A positive penalty proves it does. Zero is INCONCLUSIVE, not a pass: Nile
// enables dynamic energy (getAllowDynamicEnergy = 1) but its
// getDynamicEnergyThreshold is 5000000000 (5e9) energy per contract per
// maintenance period, which Nile traffic does not reach — so a zero factor
// says nothing about the estimate's semantics.
func checkPenalty(penalty int64) error {
	if penalty > 0 {
		return nil
	}
	return fmt.Errorf("inconclusive: energy penalty is %d (want > 0), so spec §7.5 item 6 is NOT verified — "+
		"Nile's getDynamicEnergyThreshold = 5000000000 (5e9) energy per contract per maintenance period is unreachable at testnet traffic; "+
		"use a Mainnet contract with a non-zero consumption factor, or a private chain with a lowered threshold", penalty)
}
