package main

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tx"
)

func mustTriggerTx(t *testing.T, typ core.Transaction_Contract_ContractType, param []byte) *core.Transaction {
	t.Helper()
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{Type: typ, Parameter: &anypb.Any{Value: param}},
			},
		},
	}
}

func TestExtractTriggerRoundTrip(t *testing.T) {
	want := &core.TriggerSmartContract{
		OwnerAddress:    []byte{0x41, 0x11},
		ContractAddress: []byte{0x41, 0x22},
		Data:            []byte{0xa9, 0x05},
	}
	raw, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := extractTrigger(mustTriggerTx(t, core.Transaction_Contract_TriggerSmartContract, raw))
	if err != nil {
		t.Fatalf("extractTrigger: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("extractTrigger = %v, want %v", got, want)
	}
}

func TestExtractTriggerRejects(t *testing.T) {
	raw, _ := proto.Marshal(&core.TriggerSmartContract{})
	cases := []struct {
		name string
		tx   *core.Transaction
	}{
		{"wrong type is not unmarshalled blind (F1 rule)", mustTriggerTx(t, core.Transaction_Contract_TransferContract, raw)},
		{"garbage bytes", mustTriggerTx(t, core.Transaction_Contract_TriggerSmartContract, []byte{0xff, 0xff})},
		{"empty transaction", &core.Transaction{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := extractTrigger(tc.tx); err == nil {
				t.Fatal("want an error, got nil")
			}
		})
	}
	// Multi-contract transaction.
	multi := mustTriggerTx(t, core.Transaction_Contract_TriggerSmartContract, raw)
	multi.RawData.Contract = append(multi.RawData.Contract, multi.RawData.Contract[0])
	if _, err := extractTrigger(multi); err == nil {
		t.Fatal("multi-contract: want an error, got nil")
	}
}

func TestReplayBandwidthVerdictShapes(t *testing.T) {
	// Burn: 345 bytes × 1000 = 345,000 — the Nile live-run shape.
	if detail, ok := replayBandwidthVerdict(345, 1000, &core.ResourceReceipt{NetUsage: 0, NetFee: 345_000}); !ok {
		t.Fatalf("burn shape must pass: %s", detail)
	}
	// Covered: usage equals the byte count, no fee.
	if detail, ok := replayBandwidthVerdict(281, 1000, &core.ResourceReceipt{NetUsage: 281}); !ok {
		t.Fatalf("covered shape must pass: %s", detail)
	}
	// Off-by-one on either side fails.
	if detail, ok := replayBandwidthVerdict(345, 1000, &core.ResourceReceipt{NetFee: 345_001}); ok {
		t.Fatalf("wrong fee must not pass: %s", detail)
	}
	if detail, ok := replayBandwidthVerdict(281, 1000, &core.ResourceReceipt{NetUsage: 282}); ok {
		t.Fatalf("wrong usage must not pass: %s", detail)
	}
	// Creation-scale fee on non-100 bytes is reported, not passed.
	if detail, ok := replayBandwidthVerdict(281, 1000, &core.ResourceReceipt{NetFee: 100_000}); ok {
		t.Fatalf("creation-fee shape must not pass as burn: %s", detail)
	}
	// The exact-100-byte ambiguity is named.
	detail, ok := replayBandwidthVerdict(100, 1000, &core.ResourceReceipt{NetFee: 100_000})
	if ok {
		t.Fatalf("ambiguous shape must not pass: %s", detail)
	}
	if !strings.Contains(detail, "ambiguous") {
		t.Fatalf("detail should name the ambiguity: %s", detail)
	}
}

func TestReplayVerdictExactOnly(t *testing.T) {
	pass, ok := replayVerdict(
		&tx.Estimate{Energy: 64285, Penalty: 49635},
		&core.ResourceReceipt{EnergyUsageTotal: 64285, EnergyPenaltyTotal: 49635},
	)
	if !ok {
		t.Fatalf("exact match must pass: %s", pass)
	}
	for _, tc := range []struct {
		name string
		est  *tx.Estimate
		rec  *core.ResourceReceipt
	}{
		{"energy differs by one", &tx.Estimate{Energy: 64286, Penalty: 49635}, &core.ResourceReceipt{EnergyUsageTotal: 64285, EnergyPenaltyTotal: 49635}},
		{"penalty differs by one", &tx.Estimate{Energy: 64285, Penalty: 49634}, &core.ResourceReceipt{EnergyUsageTotal: 64285, EnergyPenaltyTotal: 49635}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if detail, ok := replayVerdict(tc.est, tc.rec); ok {
				t.Fatalf("off-by-one must not pass: %s", detail)
			}
		})
	}
}
