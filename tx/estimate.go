package tx

import (
	"context"
	"math"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/protobuf/proto"
)

// Estimate is the result of a read-only dry run of a ContractTx against the
// node (architecture §7.2). It deliberately has NO TxID field (the B5 fix): a
// simulation never becomes a transaction, so any id it could carry would be
// fabricated.
//
// live-verified: §7.5 — Estimate.Energy (TransactionExtention.EnergyUsed,
// 13569 on the live run) matches the post-broadcast
// ResourceReceipt.EnergyUsageTotal (13569) EXACTLY; it is the accurate
// execution cost and therefore the source CostPreview uses for EnergyNeeded.
type Estimate struct {
	// ConstantResult holds the returned ABI-encoded values of the constant
	// call (one entry per returned value).
	ConstantResult [][]byte
	// Energy is the energy the call is estimated to consume
	// (TransactionExtention.EnergyUsed) — live-verified to equal the actual
	// post-broadcast EnergyUsageTotal exactly.
	Energy int64
	// Penalty is the TIP-491 dynamic-model energy surcharge the node already
	// includes in Energy (TransactionExtention.EnergyPenalty, field 8).
	Penalty int64
	// Net is always 0: TriggerConstantContract exposes no bandwidth figure.
	// Receipt.Cost reports actual bandwidth after broadcast.
	Net int64
	// Revert is the node's revert/failure message (ResMessage), when any.
	Revert string
	// Code is the v2 classification of a node-level rejection; "" means the
	// simulation ran (which does NOT mean the eventual broadcast will — an
	// execution revert surfaces here as Revert + Code receipt.reverted).
	Code tron.Code
}

// EnergyEstimate is the result of the node's EstimateEnergy RPC (architecture §7.1):
// the penalty-INCLUSIVE total energy the call is expected to consume. It has
// a single field because that is all the RPC exposes
// (api.EstimateEnergyMessage.EnergyRequired) — a Base/Penalty split here
// would be fabricated; use ContractTx.Simulate (Estimate.Penalty) for the
// split.
//
// live-verified: §7.5 — EstimateEnergy is the node's CONSERVATIVE fee-limit
// calculator: on the live run it returned 20354 for a call whose actual
// execution cost was 13569 (a 1.5× safety margin, sized so a fee limit set
// from it never runs out of energy). It is the right answer for "what fee
// limit guarantees success", NOT for "what will this cost" — use
// ContractTx.Simulate (Estimate.Energy) for the accurate cost prediction;
// CostPreview does exactly that.
type EnergyEstimate struct {
	// Energy is the total (penalty-inclusive) energy estimate.
	Energy int64
}

// HasResult reports whether the simulated call returned any ABI values. It
// replaces the `len(e.ConstantResult) > 0` idiom (architecture §7.2) and is nil-safe.
func (e *Estimate) HasResult() bool { return e != nil && len(e.ConstantResult) > 0 }

// DeployEstimate is the result of a read-only dry run of a DeployTx
// against the node: the full deployment energy — init execution plus the
// 200-per-byte code deposit — with the TIP-491 factor applied per the
// contract's (nonexistent-yet, hence zero) state. Like Estimate it
// deliberately has NO TxID field.
//
// The estimation path is the node's triggerConstantContract with an empty
// contract address and the init bytecode as data (java-tron Wallet
// synthesizes a CreateSmartContract server-side): the same VM execution
// the broadcast performs, minus persistence. Live-verified: constant 221
// == broadcast receipt 221 on Nile; deposit rate 200/byte verified
// differentially (6200 over 31 bytes).
type DeployEstimate struct {
	// Energy is the full deployment energy the broadcast will consume.
	Energy int64
	// Penalty is the TIP-491 surcharge included in Energy (zero for a
	// contract with no consumption history).
	Penalty int64
	// Revert is the node's revert/failure message, when any.
	Revert string
	// Code is the v2 classification of a node-level rejection; "" means
	// the dry run executed (which does NOT mean the broadcast will — a
	// revert surfaces here, not as an error).
	Code tron.Code
}

// Estimate dry-runs the deployment read-only via the node's deploy
// estimation path and returns the full energy the broadcast will consume
// (research 2026-09-28 — the "no simulation path" premise in architecture §6.1
// is superseded: the path exists, it just takes bytecode instead of a
// built call). It exists ONLY on *DeployTx. The request reuses the built
// transaction's own owner, bytecode and call value, so what is estimated
// is what will be broadcast. A node-level rejection is returned in
// Code/Revert, not as an error; transport failures are *tron.Error.
//
// fee_limit sizing from the estimate follows the documented strategies
// (estimate × SunPerEnergy, plus headroom for the TIP-491 upper bound).
func (t *DeployTx) Estimate(ctx context.Context) (*DeployEstimate, error) {
	const op = "tx.DeployTx.Estimate"
	if t == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	raw := t.raw()
	if err := requireOneContract(raw, op); err != nil {
		return nil, err
	}
	if c := raw.GetContract()[0]; c.GetType() != core.Transaction_Contract_CreateSmartContract {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "the wrapped contract is not a CreateSmartContract; was the extention replaced via Extension()?"}
	}
	var create core.CreateSmartContract
	if err := proto.Unmarshal(raw.GetContract()[0].GetParameter().GetValue(), &create); err != nil {
		return nil, &tron.Error{
			Code: tron.CodeTxInvalidArgument, Op: op, Cause: err,
			Hint: "the wrapped contract parameter does not decode as CreateSmartContract; was the extention replaced via Extension()?",
		}
	}
	// Empty contract address selects the node's deploy-estimation path:
	// it synthesizes the CreateSmartContract server-side and runs the
	// same VM execution the broadcast would.
	req := &core.TriggerSmartContract{
		OwnerAddress: create.GetOwnerAddress(),
		Data:         create.GetNewContract().GetBytecode(),
		CallValue:    create.GetNewContract().GetCallValue(),
	}
	ext, err := rpc.TriggerConstantContract(t.cp, ctx, req)
	if err != nil {
		return nil, err
	}
	est := &DeployEstimate{
		Energy:  ext.GetEnergyUsed(),
		Penalty: ext.GetEnergyPenalty(),
		Revert:  string(ext.GetResult().GetMessage()),
	}
	if ret := ext.GetResult(); !ret.GetResult() {
		if isRevertMessage(est.Revert) {
			est.Code = tron.CodeReceiptReverted
		} else {
			est.Code = mappedReturnCode(ret, op)
		}
	}
	return est, nil
}

// EffectiveFactor derives the TIP-491 surcharge factor the node applied to
// this simulation, from the node's own two numbers:
//
//	FactorDecimal*Energy/(Energy-Penalty) - FactorDecimal
//
// It is the second independent read of the same factor DynamicEnergyOf
// reports (the first). The aggregate derivation floors at or below the
// stored factor — derived > stored is impossible under the per-opcode
// formula, so it contradicts the model rather than approximating it.
// A penalty-free simulation derives exactly 0.
//
// ok is false when the estimate carries no information to derive from:
// a nil estimate, non-positive energy, a negative penalty, or a penalty
// at or above the energy (a degenerate node answer — penalty scales with
// base cost, so it cannot meet or exceed the total).
func (e *Estimate) EffectiveFactor() (factor int64, ok bool) {
	if e == nil || e.Energy <= 0 || e.Penalty < 0 || e.Penalty >= e.Energy {
		return 0, false
	}
	if e.Penalty == 0 {
		return 0, true
	}
	base := e.Energy - e.Penalty
	if e.Energy > math.MaxInt64/FactorDecimal {
		return 0, false
	}
	return FactorDecimal*e.Energy/base - FactorDecimal, true
}

// Simulate dry-runs the contract call read-only via the node's
// TriggerConstantContract (no fee_limit is spent, nothing is broadcast) and
// returns the decoded constant results, the energy/penalty split and any
// revert message (architecture §7.2). It exists ONLY on *ContractTx (the F1 fix) —
// calling it on any other kind is a compile error, pinned in
// v2/internal/compilecheck. A node-level rejection is returned in
// Estimate.Code/Revert, not as an error; transport failures are *tron.Error.
func (t *ContractTx) Simulate(ctx context.Context) (*Estimate, error) {
	const op = "tx.Simulate"
	req, err := triggerParam(&t.baseTx, op)
	if err != nil {
		return nil, err
	}
	ext, err := rpc.TriggerConstantContract(t.cp, ctx, req)
	if err != nil {
		return nil, err
	}
	est := &Estimate{
		ConstantResult: ext.GetConstantResult(),
		Energy:         ext.GetEnergyUsed(),
		Penalty:        ext.GetEnergyPenalty(),
		// TransactionExtention carries no message of its own; the node's
		// revert/diagnostic text rides on the embedded Result (Return.Message).
		Revert: string(ext.GetResult().GetMessage()),
	}
	if ret := ext.GetResult(); !ret.GetResult() {
		// The node rejected the simulated call. A revert marker in the
		// failure message means the contract deliberately reverted —
		// receipt-style, the same classification a real broadcast receipt
		// would carry — otherwise the raw Return maps through rpc's table.
		if isRevertMessage(est.Revert) {
			est.Code = tron.CodeReceiptReverted
		} else {
			est.Code = mappedReturnCode(ret, op)
		}
	}
	return est, nil
}

// EstimateEnergy asks the node's EstimateEnergy RPC for the penalty-inclusive
// total energy of the call (architecture §7.1). Like Simulate it exists ONLY on
// *ContractTx. A node-level rejection surfaces as a *tron.Error (the RPC's
// Return mapped through the v2 table), unlike Simulate's in-band Code.
//
// NOTE (live-verified §7.5): the node's EstimateEnergy RPC returns a
// CONSERVATIVE upper bound for fee-limit setting (1.5× actual on the live
// run), NOT the accurate execution cost. Use it when you need a fee limit
// that guarantees the call completes without OUT_OF_ENERGY; for an accurate
// cost prediction use Simulate (Estimate.Energy), which is what CostPreview
// does.
func (t *ContractTx) EstimateEnergy(ctx context.Context) (*EnergyEstimate, error) {
	const op = "tx.EstimateEnergy"
	req, err := triggerParam(&t.baseTx, op)
	if err != nil {
		return nil, err
	}
	msg, err := rpc.EstimateEnergy(t.cp, ctx, req)
	if err != nil {
		return nil, err
	}
	if ret := msg.GetResult(); !ret.GetResult() {
		return nil, &tron.Error{Code: mappedReturnCode(ret, op), Op: op, Hint: "the node rejected the EstimateEnergy request"}
	}
	return &EnergyEstimate{Energy: msg.GetEnergyRequired()}, nil
}

// triggerParam decodes the wrapped contract message of a ContractTx as the
// TriggerSmartContract request shape Simulate and EstimateEnergy re-submit.
func triggerParam(b *baseTx, op string) (*core.TriggerSmartContract, error) {
	raw := b.raw()
	if raw == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction has no raw data; build it with the tx.Build* functions"}
	}
	if err := requireOneContract(raw, op); err != nil {
		return nil, err
	}
	req := new(core.TriggerSmartContract)
	if err := proto.Unmarshal(raw.GetContract()[0].GetParameter().GetValue(), req); err != nil {
		return nil, &tron.Error{
			Code:  tron.CodeTxInvalidArgument,
			Op:    op,
			Cause: err,
			Hint:  "the wrapped contract parameter does not decode as TriggerSmartContract; was the extention replaced via Extension()?",
		}
	}
	return req, nil
}
