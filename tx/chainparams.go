package tx

import (
	"context"
	"sort"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// Governance chain parameters the cost model and the staking API need.
//
// These are the values that make a cost prediction honest. A multi-signature
// transaction pays a fixed surcharge on top of its bandwidth, and an
// AccountPermissionUpdate pays a further fixed fee — on Mainnet today 1 TRX
// and 100 TRX respectively, which dwarf the bandwidth cost of a small
// transaction. Hard-coding them would be wrong twice: they are governance
// parameters that can change by proposal, and a prediction that omits them is
// not the "all-in" number TotalCost claims to be.
const (
	paramUpdateAccountPermissionFee = "getUpdateAccountPermissionFee"
	paramMultiSignFee               = "getMultiSignFee"
	paramUnfreezeDelayDays          = "getUnfreezeDelayDays"
	paramMaxDelegateLockPeriod      = "getMaxDelegateLockPeriod"
)

// ChainParams is the subset of the node's governance parameters this library
// prices and bounds transactions with, read live from GetChainParameters.
type ChainParams struct {
	// UpdateAccountPermissionFee is the fixed SUN charged for an
	// AccountPermissionUpdateContract (Mainnet: 100 TRX).
	UpdateAccountPermissionFee tron.SUN
	// MultiSignFee is the fixed SUN surcharge charged once when a transaction
	// carries two or more signatures (Mainnet: 1 TRX).
	MultiSignFee tron.SUN
	// UnfreezeDelayDays is the unstake cooldown in DAYS (Mainnet: 14, Nile:
	// 1). It is a chain parameter, never a constant: do not display "14 days"
	// without reading it.
	UnfreezeDelayDays int64
	// MaxDelegateLockPeriod is the longest delegation lock the network
	// accepts, in BLOCKS.
	MaxDelegateLockPeriod int64
	// Missing names the parameters the node did not report, in a stable
	// order. A missing fee parameter is a hard error from ChainParamsOf (the
	// cost would be understated); the bound parameters are informational here
	// and their absence is recorded rather than fatal.
	Missing []string
}

// ChainParamsOf reads the governance parameters. The two fee parameters are
// required: if the node omits either, the read fails with
// contract.bad_metadata rather than pricing a fee as zero. The bounds
// (unstake cooldown, maximum lock) are recorded in Missing when absent, so a
// caller can tell "the node did not say" from "the node said 0".
func ChainParamsOf(ctx context.Context, cp rpc.ConnProvider) (*ChainParams, error) {
	const op = "tx.ChainParamsOf"
	msg, err := rpc.GetChainParameters(cp, ctx, &api.EmptyMessage{})
	if err != nil {
		return nil, err
	}
	values := make(map[string]int64, len(msg.GetChainParameter()))
	for _, p := range msg.GetChainParameter() {
		values[p.GetKey()] = p.GetValue()
	}
	if len(values) == 0 {
		return nil, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
			Hint: "the node returned no chain parameters; the cost model cannot price governance fees or resource bounds"}
	}
	required := func(key string) (int64, error) {
		v, ok := values[key]
		if !ok {
			return 0, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
				Hint: "the node's chain parameters do not include " + key + "; refusing to price it as zero"}
		}
		return v, nil
	}
	out := &ChainParams{}
	if out.UpdateAccountPermissionFee, err = requiredSun(required, paramUpdateAccountPermissionFee); err != nil {
		return nil, err
	}
	if out.MultiSignFee, err = requiredSun(required, paramMultiSignFee); err != nil {
		return nil, err
	}
	optional := func(key string, dst *int64) {
		v, ok := values[key]
		if !ok {
			out.Missing = append(out.Missing, key)
			return
		}
		*dst = v
	}
	optional(paramUnfreezeDelayDays, &out.UnfreezeDelayDays)
	optional(paramMaxDelegateLockPeriod, &out.MaxDelegateLockPeriod)
	sort.Strings(out.Missing)
	return out, nil
}

func requiredSun(required func(string) (int64, error), key string) (tron.SUN, error) {
	v, err := required(key)
	if err != nil {
		return 0, err
	}
	if v < 0 {
		return 0, &tron.Error{Code: tron.CodeContractBadMetadata, Op: "tx.ChainParamsOf",
			Hint: "the node reported a negative value for " + key}
	}
	return tron.SUN(v), nil
}

// contractTypeOf returns the wrapped contract type of a single-contract
// transaction, or false when the transaction is not in that shape.
func contractTypeOf(t Tx) (core.Transaction_Contract_ContractType, bool) {
	if t == nil || t.Transaction() == nil {
		return 0, false
	}
	raw := t.Transaction().GetRawData()
	if raw == nil || len(raw.GetContract()) != 1 {
		return 0, false
	}
	return raw.GetContract()[0].GetType(), true
}
