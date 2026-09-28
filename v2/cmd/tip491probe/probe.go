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
//     formula says this call's penalty should be (an upper bound).
//   - actual/base: the Simulate penalty and its penalty-excluded base.
//
// pass means the factor was observed live AND every cross-check agrees.
// live without pass is a model mismatch, not a verification. Neither live
// nor pass is the Nile outcome: both independent reads agree at zero.
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

// factorTolerance bounds how far the Simulate-derived factor may sit below
// the stored factor. The derivation aggregates per-opcode flooring into one
// division, so it is a lower bound; the gap grows with the call's opcode
// count relative to its base cost. This is a heuristic harness guardrail
// (1% of the factor, floor 8), not a protocol constant — live mainnet data
// showed a 117-unit gap on a 34000 factor (0.34%), all of it explained by
// opcode flooring. The printed numbers let a human judge any near-miss.
func factorTolerance(f int64) int64 {
	if t := f / 100; t > 8 {
		return t
	}
	return 8
}

// penaltyTolerance bounds how far the node's penalty may sit below the
// aggregate prediction, for the same per-opcode flooring reason (0.5% of
// the prediction, floor 64 energy units).
func penaltyTolerance(p int64) int64 {
	if t := p / 200; t > 64 {
		return t
	}
	return 64
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
		v.detail = fmt.Sprintf("derived factor %d exceeds stored factor %d: the simulation disagrees with GetContractInfo", simFactor, dyn.Factor)
	case dyn.Factor-simFactor > factorTolerance(dyn.Factor):
		v.detail = fmt.Sprintf("derived factor %d is below stored factor %d beyond flooring tolerance: the simulation disagrees with GetContractInfo", simFactor, dyn.Factor)
	case est.Penalty > predicted:
		v.detail = fmt.Sprintf("actual penalty %d exceeds predicted upper bound %d: the per-opcode formula does not explain the node", est.Penalty, predicted)
	case predicted-est.Penalty > penaltyTolerance(predicted):
		v.detail = fmt.Sprintf("actual penalty %d is below prediction %d beyond flooring tolerance", est.Penalty, predicted)
	default:
		v.pass = v.live
		if v.pass {
			v.detail = "stored factor, derived factor and penalty prediction agree"
		} else {
			v.detail = "inconclusive: energy penalty is 0 and the stored factor is 0, so spec §7.5 item 6 is NOT verified — " +
				"Nile's getDynamicEnergyThreshold = 5000000000 (5e9) energy per contract per maintenance period is unreachable at testnet traffic; " +
				"use a Mainnet contract with a non-zero consumption factor, or a private chain with a lowered threshold"
		}
	}
	return v
}
