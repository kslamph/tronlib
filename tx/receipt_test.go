package tx

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// waitReceipt drives a receipt out of Wait with a canned TransactionInfo.
func waitReceipt(t *testing.T, info *core.TransactionInfo) *Receipt {
	t.Helper()
	f := &fakeWalletServer{
		TxInfo: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			return info, nil
		},
	}
	cp := newTxTestClient(t, f)
	r, err := Wait(t.Context(), cp, hex.EncodeToString(hexID()))
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	return r
}

func hexID() []byte { return repeat(0xCD, 32) }

// resourceReceipt is the fake post-execution cost (architecture §7.4).
func resourceReceipt(vm core.Transaction_ResultContractResult) *core.ResourceReceipt {
	return &core.ResourceReceipt{
		EnergyFee:          42_000,
		NetFee:             500,
		EnergyUsageTotal:   210,
		OriginEnergyUsage:  100,
		EnergyPenaltyTotal: 30,
		NetUsage:           300,
		Result:             vm,
	}
}

func TestReceiptCostFromResourceReceipt(t *testing.T) {
	info := &core.TransactionInfo{
		Id:             hexID(),
		BlockNumber:    12345,
		BlockTimeStamp: 1700000000000,
		Result:         core.TransactionInfo_SUCESS,
		Receipt:        resourceReceipt(core.Transaction_Result_SUCCESS),
	}
	r := waitReceipt(t, info)
	c := r.Cost
	if c.EnergyFee != 42_000 || c.NetFee != 500 {
		t.Errorf("fees = (%d,%d), want (42000,500)", c.EnergyFee, c.NetFee)
	}
	if c.Total != 42_500 {
		t.Errorf("Total = %d, want 42500", c.Total)
	}
	if c.Energy != 210 || c.BaseEnergy != 100 || c.Penalty != 30 || c.Bandwidth != 300 {
		t.Errorf("usage = (%d,%d,%d,%d), want (210,100,30,300)", c.Energy, c.BaseEnergy, c.Penalty, c.Bandwidth)
	}
	if r.BlockNum != 12345 {
		t.Errorf("BlockNum = %d, want 12345", r.BlockNum)
	}
	if r.BlockTime.UnixMilli() != 1700000000000 {
		t.Errorf("BlockTime = %v, want 1700000000000", r.BlockTime)
	}
	if !r.OK() {
		t.Errorf("Code = %q, want success", r.Code)
	}
}

func TestReceiptRevertAndOutOfEnergyClassification(t *testing.T) {
	revert := waitReceipt(t, &core.TransactionInfo{
		Id:      hexID(),
		Result:  core.TransactionInfo_SUCESS,
		Receipt: resourceReceipt(core.Transaction_Result_REVERT),
	})
	if revert.Code != tron.CodeReceiptReverted || revert.NodeCode != "REVERT" {
		t.Errorf("revert: Code=%q NodeCode=%q, want receipt.reverted/REVERT", revert.Code, revert.NodeCode)
	}
	if revert.OK() {
		t.Error("a revert must not be OK")
	}

	ooe := waitReceipt(t, &core.TransactionInfo{
		Id:      hexID(),
		Result:  core.TransactionInfo_SUCESS,
		Receipt: resourceReceipt(core.Transaction_Result_OUT_OF_ENERGY),
	})
	if ooe.Code != tron.CodeReceiptOutOfEnergy || ooe.NodeCode != "OUT_OF_ENERGY" {
		t.Errorf("ooe: Code=%q NodeCode=%q, want receipt.out_of_energy/OUT_OF_ENERGY", ooe.Code, ooe.NodeCode)
	}

	failed := waitReceipt(t, &core.TransactionInfo{
		Id:      hexID(),
		Result:  core.TransactionInfo_FAILED,
		Receipt: resourceReceipt(core.Transaction_Result_DEFAULT),
	})
	if failed.Code != tron.CodeReceiptFailed || failed.NodeCode != "FAILED" {
		t.Errorf("failed: Code=%q NodeCode=%q, want receipt.failed/FAILED", failed.Code, failed.NodeCode)
	}
}

func TestReceiptLogsNeverDropped(t *testing.T) {
	info := &core.TransactionInfo{
		Id:      hexID(),
		Result:  core.TransactionInfo_SUCESS,
		Receipt: resourceReceipt(core.Transaction_Result_SUCCESS),
		Log: []*core.TransactionInfo_Log{
			{ // unknown signature: materialized raw with EventName ""
				Address: testTo.Bytes(),
				Topics:  [][]byte{append(repeat(0xFE, 31), 0x01)},
				Data:    []byte{0x01, 0x02},
			},
			{ // corrupt log (no topics): still materialized, never dropped
				Address: testTo.Bytes(),
				Data:    []byte{0x03},
			},
		},
	}
	r := waitReceipt(t, info)
	if len(r.Logs) != 2 {
		t.Fatalf("Logs = %d entries, want 2 (nothing dropped)", len(r.Logs))
	}
	if r.Logs[0].EventName != "" || len(r.Logs[0].Topics) != 1 || len(r.Logs[0].Data) != 2 {
		t.Errorf("unknown log = %+v, want materialized raw", r.Logs[0])
	}
	if r.Logs[0].Address != testTo {
		t.Errorf("emitting address = %v, want %v", r.Logs[0].Address, testTo)
	}
	if r.Logs[1].EventName != "" || len(r.Logs[1].Data) != 1 {
		t.Errorf("corrupt log = %+v, want materialized raw", r.Logs[1])
	}
}

func TestIsRevertMessage(t *testing.T) {
	if !isRevertMessage("Revert opcode executed") {
		t.Error("revert message not detected")
	}
	if !isRevertMessage("REVERT") {
		t.Error("upper-case revert not detected")
	}
	if isRevertMessage("out of energy") {
		t.Error("non-revert message flagged as revert")
	}
}

func TestReceiptZeroCostOnMissingReceipt(t *testing.T) {
	info := &core.TransactionInfo{Id: hexID(), Result: core.TransactionInfo_SUCESS}
	r := waitReceipt(t, info)
	if (r.Cost != ActualCost{}) {
		t.Errorf("Cost = %+v, want zero value (no ResourceReceipt)", r.Cost)
	}
}
