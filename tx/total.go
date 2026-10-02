package tx

import (
	"context"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// TotalCost is the all-in predicted cost of broadcasting a fully-signed
// transaction: the energy burn (contract calls only) plus the bandwidth burn,
// plus any account-creation fee, plus the governance fees a multi-signature
// transaction or a permission update carries. It is the last number to check
// before broadcast — call it after Sign, never before: the bandwidth half
// is measured on the exact broadcast bytes, and an unsigned transaction
// fails loudly (tx.invalid_argument) instead of silently undercounting
// ~65 bytes per missing signature.
//
// "All-in" is now true: a transaction with two or more signatures adds the
// node's fixed getMultiSignFee surcharge, and an
// AccountPermissionUpdateContract adds getUpdateAccountPermissionFee. Both
// are read live from GetChainParameters (ChainParams) and can be large
// relative to bandwidth — 1 TRX and 100 TRX on Mainnet today — so a total
// that omitted them would be wrong in the direction that hurts.
//
// Two-phase flow (the build pipeline extended to cost):
//
//	build → PreviewCost (energy, pre-sign OK, ContractTx only)
//	      → Sign
//	      → TotalCostOf (all-in, every kind)
//	      → Broadcast
//
// PreviewCost stays valid after signing too (simulation ignores
// signatures), but post-sign callers should prefer TotalCostOf: one call,
// every resource, one total.
type TotalCost struct {
	// Energy is the energy preview, non-nil only for ContractTx —
	// other kinds consume no energy.
	Energy *CostPreview
	// Bandwidth is the bandwidth prediction, always present on success.
	Bandwidth *BandwidthCost
	// PermissionUpdateFee is the fixed AccountPermissionUpdate fee, zero
	// unless the transaction is one.
	PermissionUpdateFee tron.SUN
	// MultiSignFee is the fixed surcharge for carrying two or more
	// signatures, zero otherwise.
	MultiSignFee tron.SUN
	// ChainParams carries the governance values the fees came from, so the
	// breakdown is auditable. It is non-nil whenever a fee was looked up.
	ChainParams *ChainParams
	// Total is the all-in SUN outlay: energy burn + bandwidth burn +
	// creation fee + governance fees (where each applies).
	Total tron.SUN
}

// String renders the total as short lines, one per resource plus the sum.
func (c *TotalCost) String() string {
	gov := ""
	if c.PermissionUpdateFee > 0 || c.MultiSignFee > 0 {
		gov = "; permission update = " + c.PermissionUpdateFee.String() + " sun; multisig = " + c.MultiSignFee.String() + " sun"
	}
	if c.Energy == nil {
		return "total cost: no energy (non-contract); " + c.Bandwidth.String() + gov +
			"; total = " + c.Total.String() + " sun"
	}
	return "total cost: energy = " + c.Energy.TronToBurn.String() + " sun; " +
		c.Bandwidth.String() + gov + "; total = " + c.Total.String() + " sun"
}

// TotalCostOf predicts the all-in cost of broadcasting the fully-signed t
// for owner. ContractTx gets an energy preview plus bandwidth; every other
// kind gets bandwidth only (plus the creation fee where it applies). The
// bandwidth half runs first, so an unsigned transaction fails fast with
// tx.invalid_argument before any simulation RPC is spent.
//
// Governance fees are added when they apply: the multisig surcharge once a
// transaction carries two or more signatures, and the permission-update fee
// for an AccountPermissionUpdateContract. Both are read live from
// GetChainParameters, and that read only happens when one of the two
// applies, so the common single-signature transfer pays nothing extra.
func TotalCostOf(ctx context.Context, cp rpc.ConnProvider, t Tx, owner tron.Address) (*TotalCost, error) {
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
	bw, err := BandwidthCostOf(ctx, cp, t, owner)
	if err != nil {
		return nil, err
	}
	total, err := addSUN(op, bw.Burn, bw.NewAccountFee,
		"bandwidth burn plus creation fee overflows SUN")
	if err != nil {
		return nil, err
	}
	out := &TotalCost{Bandwidth: bw, Total: total}

	// Governance fees: only consult the chain parameters when a fee actually
	// applies, so the common single-signature transfer costs no extra read.
	extra, err := governanceFees(ctx, cp, t, out)
	if err != nil {
		return nil, err
	}
	if total, err = addSUN(op, total, extra,
		"bandwidth outlay plus governance fees overflows SUN"); err != nil {
		return nil, err
	}
	out.Total = total

	contractTx, ok := t.(*ContractTx)
	if !ok {
		return out, nil // non-contract kinds consume no energy
	}
	energy, err := PreviewCost(ctx, cp, contractTx, owner)
	if err != nil {
		return nil, err
	}
	out.Energy = energy
	if total, err = addSUN(op, total, energy.TronToBurn,
		"energy burn plus bandwidth outlay overflows SUN"); err != nil {
		return nil, err
	}
	out.Total = total
	return out, nil
}

// governanceFees fills out's multi-signature and permission-update fee fields
// when they apply, reading the live chain parameters once, and returns their
// sum. It sets out.ChainParams so the breakdown stays auditable.
func governanceFees(ctx context.Context, cp rpc.ConnProvider, t Tx, out *TotalCost) (tron.SUN, error) {
	const op = "tx.TotalCostOf"
	needsMultisig := len(t.Transaction().GetSignature()) >= 2
	ct, hasContract := contractTypeOf(t)
	needsPermissionFee := hasContract && ct == core.Transaction_Contract_AccountPermissionUpdateContract
	if !needsMultisig && !needsPermissionFee {
		return 0, nil
	}
	params, err := ChainParamsOf(ctx, cp)
	if err != nil {
		return 0, err
	}
	out.ChainParams = params
	if needsMultisig {
		out.MultiSignFee = params.MultiSignFee
	}
	if needsPermissionFee {
		out.PermissionUpdateFee = params.UpdateAccountPermissionFee
	}
	return addSUN(op, out.MultiSignFee, out.PermissionUpdateFee, "governance fees overflow SUN")
}

// addSUN adds two amounts, classifying an overflow under the caller's op with
// the caller's hint. Every accumulation in the cost model goes through it, so
// the overflow class cannot drift per call site.
func addSUN(op string, a, b tron.SUN, hint string) (tron.SUN, error) {
	sum, err := a.Add(b)
	if err != nil {
		return 0, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Cause: err, Hint: hint}
	}
	return sum, nil
}
