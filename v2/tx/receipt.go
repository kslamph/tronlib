package tx

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/event"
	"github.com/kslamph/tronlib/v2/tron"
)

// Receipt is the outcome of a broadcast or a Wait: everything the caller
// needs after the node spoke about the transaction.
//
// There is no stored Success field (spec §6.6): OK() derives from Code, so
// the two cannot contradict. Code is the v2 code ("" when the transaction
// succeeded); NodeCode preserves the raw node code — the api.Return_* name
// on the broadcast path, the VM result name (REVERT, OUT_OF_ENERGY, …) on
// the receipt path — because e.g. BANDWITH_ERROR, CONTRACT_EXE_ERROR and
// CONTRACT_VALIDATE_ERROR have three different remedies that one mapped
// code would collapse.
type Receipt struct {
	// TxID is the hex transaction id this receipt belongs to.
	TxID string
	// Code is the v2 classification; "" means success.
	Code tron.Code
	// NodeCode is the raw node code name (api.Return_* or VM result),
	// preserved because the mapped Code collapses distinct remedies.
	NodeCode string
	// BlockNum is the inclusion block, 0 if not yet included.
	BlockNum uint64
	// BlockTime is the inclusion block's timestamp.
	BlockTime time.Time
	// Cost is the actual post-execution cost (spec §7.4).
	Cost ActualCost
	// Logs are the decoded event logs. Unknown signatures materialize with
	// EventName "" and raw bytes — NEVER dropped (event.DecodeLenient).
	Logs []event.Log
	// Revert is the node's revert/failure message, when any.
	Revert string

	// solid records the provenance: receipts from the Solidity endpoint
	// (WaitForSolid) are solidified. There is no Solidified field in
	// core.TransactionInfo, so this is a provenance fact, not a parsed one.
	solid bool
}

// OK reports whether the transaction succeeded. Derived from Code; there is
// no stored Success field to contradict it.
func (r *Receipt) OK() bool { return r.Code == "" }

// Solidified reports whether this receipt comes from the Solidity endpoint
// (WaitForSolid), i.e. the block is confirmed by the super representatives —
// the finality semantic for custody and deposit-crediting use cases.
func (r *Receipt) Solidified() bool { return r.solid }

// ActualCost is the real post-execution cost from core.ResourceReceipt
// (spec §7.4). v1's BroadcastResult exposed only usage counts and discarded
// the fee fields; v2 carries both — EnergyFee and NetFee are the actual SUN
// burned — so a CostPreview→actual delta is directly observable.
type ActualCost struct {
	// EnergyFee is the real TRX burned on energy (SUN).
	EnergyFee tron.SUN
	// NetFee is the real TRX burned on bandwidth (SUN).
	NetFee tron.SUN
	// Total is EnergyFee + NetFee.
	Total tron.SUN
	// Energy is ResourceReceipt.EnergyUsageTotal.
	Energy int64
	// BaseEnergy is OriginEnergyUsage (pre-TIP-491 energy).
	BaseEnergy int64
	// Penalty is EnergyPenaltyTotal (the dynamic-model surcharge).
	Penalty int64
	// Bandwidth is NetUsage.
	Bandwidth int64
}

// receiptFromInfo parses a core.TransactionInfo (the poll answer of Wait,
// WaitForSolid and Broadcast's reconciliation) into a Receipt.
func receiptFromInfo(info *core.TransactionInfo, solid bool) *Receipt {
	r := &Receipt{
		TxID:      hex.EncodeToString(info.GetId()),
		BlockNum:  uint64(info.GetBlockNumber()),
		BlockTime: time.UnixMilli(info.GetBlockTimeStamp()),
		Revert:    string(info.GetResMessage()),
		solid:     solid,
	}
	r.Code, r.NodeCode = receiptCode(info)
	r.Cost = actualCostFrom(info.GetReceipt())
	for _, l := range info.GetLog() {
		r.Logs = append(r.Logs, decodeReceiptLog(l))
	}
	return r
}

// receiptCode classifies a TransactionInfo onto the v2 receipt codes. The VM
// result (ResourceReceipt.Result) is the sharper signal: REVERT means the
// contract deliberately reverted, OUT_OF_ENERGY means the fee limit or
// staked energy was exhausted; everything else that is not a success is
// receipt.failed.
func receiptCode(info *core.TransactionInfo) (tron.Code, string) {
	infoOK := info.GetResult() == core.TransactionInfo_SUCESS
	vm := info.GetReceipt().GetResult()
	switch {
	case infoOK && (vm == core.Transaction_Result_SUCCESS || vm == core.Transaction_Result_DEFAULT):
		return "", vmNodeCode(info)
	case vm == core.Transaction_Result_REVERT:
		return tron.CodeReceiptReverted, vmNodeCode(info)
	case vm == core.Transaction_Result_OUT_OF_ENERGY:
		return tron.CodeReceiptOutOfEnergy, vmNodeCode(info)
	default:
		return tron.CodeReceiptFailed, vmNodeCode(info)
	}
}

// vmNodeCode renders the raw node code for a receipt: the VM result name
// (REVERT, OUT_OF_ENERGY, …) when the receipt carries one, else the
// TransactionInfo result name (SUCESS/FAILED — the proto's spelling).
func vmNodeCode(info *core.TransactionInfo) string {
	if vm := info.GetReceipt().GetResult(); vm != core.Transaction_Result_DEFAULT {
		return vm.String()
	}
	return info.GetResult().String()
}

// actualCostFrom parses the resource receipt (spec §7.4).
func actualCostFrom(rr *core.ResourceReceipt) ActualCost {
	if rr == nil {
		return ActualCost{}
	}
	energyFee := tron.SUN(rr.GetEnergyFee())
	netFee := tron.SUN(rr.GetNetFee())
	return ActualCost{
		EnergyFee:  energyFee,
		NetFee:     netFee,
		Total:      energyFee + netFee,
		Energy:     rr.GetEnergyUsageTotal(),
		BaseEnergy: rr.GetOriginEnergyUsage(),
		Penalty:    rr.GetEnergyPenaltyTotal(),
		Bandwidth:  rr.GetNetUsage(),
	}
}

// decodeReceiptLog decodes one wire log via event.DecodeLenient and attaches
// the emitting contract. Unknown signatures materialize with EventName "" and
// raw bytes; a known-but-corrupt log is also materialized raw rather than
// dropped — a receipt must never lose a log entry it was given.
func decodeReceiptLog(l *core.TransactionInfo_Log) event.Log {
	el, err := event.DecodeLenient(l.GetTopics(), l.GetData())
	if err != nil || el == nil {
		el = &event.Log{Topics: l.GetTopics(), Data: l.GetData(), EventName: ""}
	}
	if a, err := tron.AddressFromBytes(l.GetAddress()); err == nil {
		el.Address = a
	}
	return *el
}

// revertIndicator is the marker TRON nodes put in a constant-call failure
// message when the contract executed a REVERT.
const revertIndicator = "REVERT"

// isRevertMessage reports whether a node failure message indicates a
// contract-level revert (as opposed to a transaction-shape validation
// failure).
func isRevertMessage(msg string) bool {
	return strings.Contains(strings.ToUpper(msg), revertIndicator)
}
