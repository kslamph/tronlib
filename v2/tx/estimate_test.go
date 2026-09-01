package tx

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSimulateDecodesEnergyPenaltyAndResult(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	// Echo the request into the canned ext so the built tx wraps the real
	// owner/contract/data (the default triggerExt() carries no Data).
	f.Trigger = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return baseExt(contractAny(core.Transaction_Contract_TriggerSmartContract, in), okResult()), nil
	}
	ctxTx, err := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, []byte{0xa9, 0x05}, 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var gotReq *core.TriggerSmartContract
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		gotReq = in
		return &api.TransactionExtention{
			ConstantResult: [][]byte{{0x01, 0x02}},
			Result:         &api.Return{Result: true, Message: []byte("revert reason")},
			EnergyUsed:     643,
			EnergyPenalty:  200,
		}, nil
	}
	est, err := ctxTx.Simulate(ctx)
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	if est.Energy != 643 || est.Penalty != 200 {
		t.Errorf("Energy/Penalty = %d/%d, want 643/200", est.Energy, est.Penalty)
	}
	if est.Net != 0 {
		t.Errorf("Net = %d, want 0 (TriggerConstantContract exposes no bandwidth)", est.Net)
	}
	if est.Revert != "revert reason" {
		t.Errorf("Revert = %q, want %q", est.Revert, "revert reason")
	}
	if est.Code != "" {
		t.Errorf("Code = %q, want empty on a successful simulation", est.Code)
	}
	if len(est.ConstantResult) != 1 || string(est.ConstantResult[0]) != "\x01\x02" {
		t.Errorf("ConstantResult = %v, want one entry 0102", est.ConstantResult)
	}
	if gotReq == nil || string(gotReq.GetOwnerAddress()) != string(testFrom.Bytes()) ||
		string(gotReq.GetContractAddress()) != string(testTo.Bytes()) ||
		string(gotReq.GetData()) != "\xa9\x05" {
		t.Errorf("TriggerConstantContract request = %+v, want the tx's owner/contract/data", gotReq)
	}
}

func TestSimulateNodeRejectReturnsReceiptStyleCode(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{
			Result: &api.Return{Result: false, Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("bad contract")},
		}, nil
	}
	est, err := ctxTx.Simulate(ctx)
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	if est.Code == "" {
		t.Fatal("node reject must surface a receipt-style Code, got empty")
	}
	if est.Revert != "bad contract" {
		t.Errorf("Revert = %q, want the node's message", est.Revert)
	}
}

func TestSimulateRevertMessageClassifiedReverted(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{
			Result: &api.Return{Result: false, Code: api.Return_CONTRACT_EXE_ERROR, Message: []byte("REVERT opcode executed")},
		}, nil
	}
	est, err := ctxTx.Simulate(ctx)
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	if est.Code != tron.CodeReceiptReverted {
		t.Errorf("Code = %q, want %q for a revert message", est.Code, tron.CodeReceiptReverted)
	}
}

func TestSimulateTransportErrorPropagates(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		// Block until the caller's context dies: an unambiguous transport
		// failure that Call maps to chain.timeout.
		<-ctx.Done()
		return nil, ctx.Err()
	}
	_, err := ctxTx.Simulate(ctx)
	if err == nil || !tron.HasCode(err, tron.CodeChainTimeout) {
		t.Errorf("Simulate transport failure = %v, want chain.timeout", err)
	}
}

func TestEstimateEnergyReturnsInclusiveTotal(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	var gotReq *core.TriggerSmartContract
	f.EstimateEnerg = func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
		gotReq = in
		return &api.EstimateEnergyMessage{Result: okResult(), EnergyRequired: 5000}, nil
	}
	est, err := ctxTx.EstimateEnergy(ctx)
	if err != nil {
		t.Fatalf("EstimateEnergy: %v", err)
	}
	if est.Energy != 5000 {
		t.Errorf("Energy = %d, want 5000", est.Energy)
	}
	if gotReq == nil {
		t.Error("EstimateEnergy must re-submit the tx's TriggerSmartContract shape")
	}
}

func TestEstimateEnergyNodeRejectIsError(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.EstimateEnerg = func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
		return &api.EstimateEnergyMessage{Result: &api.Return{Result: false, Code: api.Return_CONTRACT_VALIDATE_ERROR}}, nil
	}
	_, err := ctxTx.EstimateEnergy(ctx)
	if err == nil || !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("EstimateEnergy node reject = %v, want a mapped code error", err)
	}
}

func TestPreviewCostThreeReadSequence(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{
			Result:        okResult(),
			EnergyUsed:    643,
			EnergyPenalty: 200,
		}, nil
	}
	f.EstimateEnerg = func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
		return &api.EstimateEnergyMessage{Result: okResult(), EnergyRequired: 5000}, nil
	}
	var gotOwner *core.Account
	f.AccountResource = func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		gotOwner = in
		return &api.AccountResourceMessage{EnergyLimit: 2000, EnergyUsed: 500}, nil
	}
	// Two entries: the LATEST timestamp (1691500000000 → 420) must win.
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "1691400000000:410,1691500000000:420"}, nil
	}
	cp2, err := PreviewCost(cp, ctx, ctxTx, testFrom)
	if err != nil {
		t.Fatalf("CostPreview: %v", err)
	}
	if cp2.EnergyNeeded != 5000 {
		t.Errorf("EnergyNeeded = %d, want 5000 (from EstimateEnergy, the inclusive total)", cp2.EnergyNeeded)
	}
	if cp2.EnergyPenalty != 200 {
		t.Errorf("EnergyPenalty = %d, want 200 (from Simulate)", cp2.EnergyPenalty)
	}
	if cp2.EnergyBase != 4800 {
		t.Errorf("EnergyBase = %d, want 4800", cp2.EnergyBase)
	}
	if cp2.EnergyAvailable != 1500 {
		t.Errorf("EnergyAvailable = %d, want 1500 (2000 limit − 500 used)", cp2.EnergyAvailable)
	}
	if cp2.EnergyToBuy != 3500 {
		t.Errorf("EnergyToBuy = %d, want 3500", cp2.EnergyToBuy)
	}
	if cp2.SunPerEnergy != 420 {
		t.Errorf("SunPerEnergy = %d, want 420 (latest timestamp wins)", cp2.SunPerEnergy)
	}
	if cp2.TronToBurn != 1_470_000 {
		// 3500 × 420 = 1,470,000 SUN (1.47 TRX). NOTE: the dispatch brief's
		// "1470000000" is an arithmetic slip; the spec §7.3 formula is the
		// direct multiply energyToBuy × sunPerEnergy.
		t.Errorf("TronToBurn = %d, want 1470000", cp2.TronToBurn)
	}
	if cp2.PricedAt.IsZero() {
		t.Error("PricedAt must be set")
	}
	if time.Since(cp2.PricedAt) > 10*time.Second {
		t.Errorf("PricedAt = %s, suspiciously stale", cp2.PricedAt)
	}
	if s := cp2.String(); s == "" || len(s) < 20 {
		t.Errorf("String() = %q, want a rendered one-line preview", s)
	}
	// The read sequence: exactly one call per RPC.
	for name, n := range map[string]int32{
		"Simulate":           f.simulateCalls.Load(),
		"EstimateEnergy":     f.estimateCalls.Load(),
		"GetAccountResource": f.accountResourceCalls.Load(),
		"GetEnergyPrices":    f.energyPricesCalls.Load(),
	} {
		if n != 1 {
			t.Errorf("%s called %d times, want exactly 1", name, n)
		}
	}
	if gotOwner == nil || string(gotOwner.GetAddress()) != string(testFrom.Bytes()) {
		t.Error("GetAccountResource must be queried for the owner address")
	}
}

func TestPreviewCostSimulateErrorPropagates(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return nil, status.Error(codes.DeadlineExceeded, "boom")
	}
	_, err := PreviewCost(cp, ctx, ctxTx, testFrom)
	if err == nil || !tron.HasCode(err, tron.CodeChainTimeout) {
		t.Errorf("PreviewCost with failing Simulate = %v, want the Simulate error", err)
	}
	if f.estimateCalls.Load() != 0 {
		t.Error("CostPreview must stop at the failed Simulate, not continue the reads")
	}
}

func TestPreviewCostMalformedPricesIsBadMetadata(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "not-a-price-list"}, nil
	}
	_, err := PreviewCost(cp, ctx, ctxTx, testFrom)
	if err == nil || !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("PreviewCost with malformed prices = %v, want %q", err, tron.CodeContractBadMetadata)
	}
}

func TestPreviewCostEmptyPricesIsBadMetadata(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{}, nil
	}
	_, err := PreviewCost(cp, ctx, ctxTx, testFrom)
	if err == nil || !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("PreviewCost with empty prices = %v, want %q", err, tron.CodeContractBadMetadata)
	}
}

func TestPreviewCostBurnOverflowIsAmountOverflow(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.EstimateEnerg = func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
		return &api.EstimateEnergyMessage{Result: okResult(), EnergyRequired: math.MaxInt64}, nil
	}
	f.AccountResource = func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		return &api.AccountResourceMessage{}, nil // nothing available → buy MaxInt64
	}
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "1691500000000:3"}, nil
	}
	_, err := PreviewCost(cp, ctx, ctxTx, testFrom)
	if err == nil || !tron.HasCode(err, tron.CodeAmountOverflow) {
		t.Errorf("PreviewCost overflow = %v, want %q", err, tron.CodeAmountOverflow)
	}
	var te *tron.Error
	if !errors.As(err, &te) {
		t.Fatalf("want a *tron.Error, got %T", err)
	}
}

func TestPreviewCostNilTxRejected(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	_, err := PreviewCost(cp, t.Context(), nil, testFrom)
	if err == nil || !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("PreviewCost(nil) = %v, want tx.invalid_argument", err)
	}
}

// TestPreviewCostFeeLimitGateTooLow pins the §6.4 floor-check: the computed
// burn (3500 energy × 420 sun = 1,470,000 SUN) exceeds a 1-TRX fee limit, so
// PreviewCost must return tx.fee_limit_too_low with Next=fix_transaction —
// the gate the code table defined but no path emitted before this fix.
func TestPreviewCostFeeLimitGateTooLow(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: okResult(), EnergyUsed: 5, EnergyPenalty: 0}, nil
	}
	f.EstimateEnerg = func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
		return &api.EstimateEnergyMessage{Result: okResult(), EnergyRequired: 5000}, nil
	}
	f.AccountResource = func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		return &api.AccountResourceMessage{EnergyLimit: 2000, EnergyUsed: 500}, nil
	}
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "1691500000000:420"}, nil
	}
	low := ctxTx.WithFeeLimit(tron.TRX(1)) // 1_000_000 SUN < 1_470_000 SUN burn
	_, err := PreviewCost(cp, ctx, low, testFrom)
	if err == nil || !tron.HasCode(err, tron.CodeTxFeeLimitTooLow) {
		t.Errorf("PreviewCost with 1-TRX fee limit vs 1.47-TRX burn = %v, want %q", err, tron.CodeTxFeeLimitTooLow)
	}
	var te *tron.Error
	if !errors.As(err, &te) {
		t.Fatalf("want a *tron.Error, got %T", err)
	}
	if te.Op != "tx.CostPreview" {
		t.Errorf("Op = %q, want tx.CostPreview", te.Op)
	}
	if te.Action() != tron.ActionFixTransaction {
		t.Errorf("Action() = %v, want fix_transaction (rebuild with higher fee limit, re-sign, re-broadcast)", te.Action())
	}
}

// TestPreviewCostFeeLimitGatePasses: the same preview with a 5-TRX fee limit
// (5,000,000 SUN > 1,470,000 SUN burn) succeeds — the gate is a floor-check,
// not a rejection of fee limits in general.
func TestPreviewCostFeeLimitGatePasses(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: okResult(), EnergyUsed: 5, EnergyPenalty: 0}, nil
	}
	f.EstimateEnerg = func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
		return &api.EstimateEnergyMessage{Result: okResult(), EnergyRequired: 5000}, nil
	}
	f.AccountResource = func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		return &api.AccountResourceMessage{EnergyLimit: 2000, EnergyUsed: 500}, nil
	}
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "1691500000000:420"}, nil
	}
	high := ctxTx.WithFeeLimit(tron.TRX(5)) // 5_000_000 SUN > 1_470_000 SUN burn
	preview, err := PreviewCost(cp, ctx, high, testFrom)
	if err != nil {
		t.Fatalf("PreviewCost with 5-TRX fee limit: %v", err)
	}
	if preview.TronToBurn != 1_470_000 {
		t.Errorf("TronToBurn = %d, want 1470000", preview.TronToBurn)
	}
}
