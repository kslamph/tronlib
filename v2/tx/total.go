package tx

import (
	"context"

	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// TotalCost is the all-in predicted cost of broadcasting a fully-signed
// transaction: the energy burn (contract calls only) plus the bandwidth
// burn plus any account-creation fee. It is the last number to check
// before broadcast — call it after Sign, never before: the bandwidth half
// is measured on the exact broadcast bytes, and an unsigned transaction
// fails loudly (tx.invalid_argument) instead of silently undercounting
// ~65 bytes per missing signature.
//
// Two-phase flow (the pipeline in §6 extended to cost):
//
//	build → PreviewCost (energy, pre-sign OK, ContractTx only)
//	      → Sign
//	      → TotalCostOf (all-in, every kind)
//	      → Broadcast
//
// PreviewCost stays valid after signing too (simulation ignores
// signatures), but post-sign callers should prefer TotalCostOf: one call,
// both resources, one total.
type TotalCost struct {
	// Energy is the energy preview, non-nil only for ContractTx —
	// other kinds consume no energy.
	Energy *CostPreview
	// Bandwidth is the bandwidth prediction, always present on success.
	Bandwidth *BandwidthCost
	// Total is the all-in SUN outlay: energy burn + bandwidth burn +
	// creation fee (where each applies).
	Total tron.SUN
}

// String renders the total as short lines, one per resource plus the sum.
func (c *TotalCost) String() string {
	if c.Energy == nil {
		return "total cost: no energy (non-contract); " + c.Bandwidth.String() +
			"; total = " + c.Total.String() + " sun"
	}
	return "total cost: energy = " + c.Energy.TronToBurn.String() + " sun; " +
		c.Bandwidth.String() + "; total = " + c.Total.String() + " sun"
}

// TotalCostOf predicts the all-in cost of broadcasting the fully-signed t
// for owner. ContractTx gets an energy preview plus bandwidth; every other
// kind gets bandwidth only (plus the creation fee where it applies). The
// bandwidth half runs first, so an unsigned transaction fails fast with
// tx.invalid_argument before any simulation RPC is spent.
func TotalCostOf(cp rpc.ConnProvider, ctx context.Context, t Tx, owner tron.Address) (*TotalCost, error) {
	const op = "tx.TotalCostOf"
	if cp == nil {
		return nil, &tron.Error{Code: tron.CodeChainConnection, Op: op, Hint: "cp is nil; pass a connected *rpc.Client"}
	}
	if t == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	if owner.IsZero() {
		return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Hint: "owner address is unset; parse it with tron.ParseAddress"}
	}
	// Bandwidth first: it rejects unsigned transactions before the
	// simulation RPCs run.
	bw, err := BandwidthCostOf(cp, ctx, t, owner)
	if err != nil {
		return nil, err
	}
	total := bw.Burn
	if total, err = total.Add(bw.NewAccountFee); err != nil {
		return nil, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Cause: err,
			Hint: "bandwidth burn plus creation fee overflows SUN",
		}
	}
	out := &TotalCost{Bandwidth: bw, Total: total}
	ct, ok := t.(*ContractTx)
	if !ok {
		return out, nil // non-contract kinds consume no energy
	}
	energy, err := PreviewCost(cp, ctx, ct, owner)
	if err != nil {
		return nil, err
	}
	out.Energy = energy
	if total, err = total.Add(energy.TronToBurn); err != nil {
		return nil, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Cause: err,
			Hint: "energy burn plus bandwidth outlay overflows SUN",
		}
	}
	out.Total = total
	return out, nil
}
