package tx

import (
	"context"
	"math"
	"math/bits"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// FactorDecimal is the fixed-point scale of a contract's TIP-491
// dynamic-energy factor. It mirrors java-tron's
// Constant.DYNAMIC_ENERGY_FACTOR_DECIMAL (10_000): a contract's stored
// energy_factor is the surcharge in parts per FactorDecimal above the base
// cost, and the node's VM charges each opcode
// floor(base*(energy_factor+FactorDecimal)/FactorDecimal)
// (actuator/src/main/java/org/tron/core/vm/VM.java,
// Program.updateContextContractFactor).
const FactorDecimal = 10_000

// DynamicEnergy is a contract's TIP-491 dynamic-energy state as the node
// reports it. The node catches the stored state up to the current
// maintenance cycle at read time (java-tron Wallet.getContractInfo runs
// ContractStateCapsule.catchUpToCycle before answering), so this is the
// effective state for the cycle in UpdateCycle — the factor the node's VM
// is applying to this contract right now.
//
//   - Factor is the surcharge in parts per FactorDecimal above the base
//     cost. 0 means no penalty: every opcode costs its base energy.
//   - Usage is the contract's penalty-excluded base energy consumption in
//     the tracked window. It is meaningful only against the
//     getDynamicEnergyThreshold chain parameter: usage above the threshold
//     raises the factor next cycle, usage below lets it decay.
//   - UpdateCycle is the maintenance cycle the state is effective for.
//
// A contract with no state row yet (fresh contract) reads as the zero
// value: factor 0, no penalty.
type DynamicEnergy struct {
	Factor      int64
	Usage       int64
	UpdateCycle int64
}

// HasPenalty reports whether the contract currently carries a TIP-491
// surcharge. Nil-safe: an unknown state has no known penalty.
func (d *DynamicEnergy) HasPenalty() bool { return d != nil && d.Factor > 0 }

// PredictPenalty returns the TIP-491 surcharge for a call whose
// penalty-excluded base energy cost is base, under this factor:
//
//	floor(base*(Factor+FactorDecimal)/FactorDecimal) - base
//
// The node applies the same formula per opcode (VM.play, with CALL-family
// opcodes folding an identically-computed call penalty into their cost),
// so the aggregate returned here is an UPPER BOUND on what the node
// charges: per-opcode flooring means the actual penalty sits within
// (opcode count) below it. A base of 0 costs 0 under any factor.
//
// This is a planning bound, not the estimator. The exact surcharge for a
// specific call comes from Simulate (Estimate.Penalty), which runs the
// node's own VM and matches the broadcast receipt exactly (architecture §7.5).
// Use PredictPenalty to budget calls you have not simulated — e.g. what
// a larger call would cost under the current factor — never to second-
// guess a simulation you already have.
//
// Single-frame scope: the bound assumes the whole call executes under
// this factor. Calls with internal transactions into other contracts run
// each frame under that frame's own factor, which this bound does not
// model.
//
// A negative base is amount.negative; a negative factor is
// tx.invalid_argument (the node floors the stored factor at 0, so a
// negative one did not come from GetContractInfo); an overflowing
// base*(Factor+FactorDecimal) is amount.overflow.
func (d *DynamicEnergy) PredictPenalty(base int64) (int64, error) {
	const op = "tx.DynamicEnergy.PredictPenalty"
	if d == nil {
		return 0, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "DynamicEnergy is nil; read it with DynamicEnergyOf first"}
	}
	if d.Factor < 0 {
		return 0, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "factor is negative; the node floors it at 0, so this state did not come from GetContractInfo"}
	}
	if base < 0 {
		return 0, &tron.Error{Code: tron.CodeAmountNegative, Op: op, Hint: "negative base energy has no penalty"}
	}
	if d.Factor == 0 || base == 0 {
		return 0, nil
	}
	if d.Factor > math.MaxInt64-FactorDecimal {
		return 0, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Hint: "Factor+FactorDecimal overflows int64; the state is not representable"}
	}
	mult := uint64(d.Factor + FactorDecimal)
	hi, lo := bits.Mul64(uint64(base), mult)
	if hi != 0 {
		return 0, &tron.Error{Code: tron.CodeAmountOverflow, Op: op, Hint: "base*(Factor+FactorDecimal) overflows int64; the call cannot be priced"}
	}
	// lo < 2^64 and FactorDecimal = 10_000, so lo/FactorDecimal < 2^64/10^4
	// < MaxInt64: the conversion cannot overflow. mult >= FactorDecimal,
	// so total >= base and the penalty is non-negative.
	total := lo / FactorDecimal
	return int64(total) - base, nil
}

// DynamicEnergyOf reads a contract's TIP-491 dynamic-energy state via the
// node's GetContractInfo (SmartContractDataWrapper.contract_state). The
// read is constant-call cheap: no key, no signature, no spend — the caller
// address is irrelevant to a per-contract factor, so any valid address
// (including a freshly generated one) can own the surrounding calls.
//
// A wrapper with no SmartContract means no contract lives at the address
// (contract.not_found): the node returns null and gRPC transports it as an
// empty message, so the nil SmartContract — not a nil wrapper — is the
// not-found signal. A deployed contract with no state row yet (fresh
// contract) carries a SmartContract but no ContractState and reads as the
// zero DynamicEnergy (factor 0).
func DynamicEnergyOf(cp rpc.ConnProvider, ctx context.Context, contract tron.Address) (*DynamicEnergy, error) {
	const op = "tx.DynamicEnergyOf"
	if cp == nil {
		return nil, &tron.Error{Code: tron.CodeChainConnection, Op: op, Hint: "cp is nil; pass a connected *rpc.Client"}
	}
	if contract.IsZero() {
		return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Hint: "contract address is unset; parse it with tron.ParseAddress"}
	}
	wrapper, err := rpc.GetContractInfo(cp, ctx, &api.BytesMessage{Value: contract.Bytes()})
	if err != nil {
		return nil, err
	}
	if wrapper == nil || wrapper.GetSmartContract() == nil {
		return nil, &tron.Error{Code: tron.CodeContractNotFound, Op: op, Hint: "no contract at the address (or the account does not exist)"}
	}
	st := wrapper.GetContractState()
	if st == nil {
		return &DynamicEnergy{}, nil
	}
	return &DynamicEnergy{
		Factor:      st.GetEnergyFactor(),
		Usage:       st.GetEnergyUsage(),
		UpdateCycle: st.GetUpdateCycle(),
	}, nil
}
