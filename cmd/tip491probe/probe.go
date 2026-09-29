package main

import (
	"fmt"

	"github.com/kslamph/tronlib/v2/tx"
)

// factorVerdict is the outcome of cross-checking the node's two independent
// reports of a contract's TIP-491 factor against the factor arithmetic the
// library implements:
//
//   - infoFactor: GetContractInfo's stored energy_factor, caught up to the
//     current maintenance cycle by the node itself.
//   - simFactor: the effective factor derived from Simulate's
//     (energy, penalty) pair via Estimate.EffectiveFactor.
//   - predicted: PredictPenalty(base, infoFactor) — what the per-opcode
//     formula says this call's penalty is at most (an upper bound).
//   - actual/base: the Simulate penalty and its penalty-excluded base.
//
// The checks are EXACT one-sided bounds, not tolerances:
//
//   - simFactor > infoFactor is impossible under the model (the aggregate
//     derivation floors at or below the stored factor), so it is a
//     contradiction, never noise.
//   - actual > predicted is impossible under the model (the aggregate is
//     an upper bound on the per-opcode sum), so it is a contradiction.
//
// The loose sides (how far below) are per-opcode flooring and carry no
// verdict — they are printed for a human to judge. pass means the factor
// was observed live AND neither exact bound is violated. live without pass
// is a model mismatch, not a verification. Neither live nor pass is the
// Nile outcome: both independent reads agree at zero.
type factorVerdict struct {
	infoFactor int64
	simFactor  int64
	predicted  int64
	actual     int64
	base       int64
	live       bool
	pass       bool
	detail     string
}

func verifyFactor(dyn *tx.DynamicEnergy, est *tx.Estimate) factorVerdict {
	v := factorVerdict{}
	if dyn == nil {
		v.detail = "no contract state to verify against"
		return v
	}
	if est == nil {
		v.detail = "no simulation to verify"
		return v
	}
	v.infoFactor = dyn.Factor
	v.actual = est.Penalty
	v.base = est.Energy - est.Penalty
	v.live = dyn.Factor > 0

	simFactor, ok := est.EffectiveFactor()
	if !ok {
		v.detail = fmt.Sprintf("energy=%d penalty=%d is degenerate: no factor derivable", est.Energy, est.Penalty)
		return v
	}
	v.simFactor = simFactor

	predicted, err := dyn.PredictPenalty(v.base)
	if err != nil {
		v.detail = fmt.Sprintf("penalty prediction failed: %v", err)
		return v
	}
	v.predicted = predicted

	// A positive stored factor with a zero penalty is uninformative, not
	// a disagreement: per-opcode flooring can zero out a small call's
	// surcharge (e.g. factor 10 on a 651-energy call). The verdict needs
	// a larger call, not a tolerance debate.
	if est.Penalty == 0 && dyn.Factor > 0 {
		v.detail = fmt.Sprintf("the stored factor is %d but this call's penalty floored to zero (base %d); simulate a larger call to observe it", dyn.Factor, v.base)
		return v
	}

	switch {
	case simFactor > dyn.Factor:
		v.detail = fmt.Sprintf("derived factor %d exceeds stored factor %d: impossible under the per-opcode formula — the simulation disagrees with GetContractInfo", simFactor, dyn.Factor)
	case est.Penalty > predicted:
		v.detail = fmt.Sprintf("actual penalty %d exceeds predicted upper bound %d: impossible under the per-opcode formula — the node charged more than the stored factor explains", est.Penalty, predicted)
	default:
		v.pass = v.live
		if v.pass {
			v.detail = fmt.Sprintf("stored factor, derived factor and penalty prediction agree (gaps: factor %d, penalty %d — per-opcode flooring)", dyn.Factor-simFactor, predicted-est.Penalty)
		} else {
			v.detail = "inconclusive: energy penalty is 0 and the stored factor is 0, so architecture §7.5 item 6 is NOT verified — " +
				"Nile's getDynamicEnergyThreshold = 5000000000 (5e9) energy per contract per maintenance period is unreachable at testnet traffic; " +
				"use a Mainnet contract with a non-zero consumption factor, or a private chain with a lowered threshold"
		}
	}
	return v
}
