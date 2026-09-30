package tx

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
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
	ctxTx, err := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, []byte{0xa9, 0x05}, 0)
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
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
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
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
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
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
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

// swapContractPayload replaces a built ContractTx's wrapped contract with the
// canned TransferContract extention — the shape a caller reaches through the
// documented Extension()/Transaction() escape hatch. TransferContract is
// WIRE-COMPATIBLE with TriggerSmartContract (1: owner, 2: recipient ↔
// contract_address, 3: amount ↔ call_value), so a parameter decode without a
// contract-type check silently describes a transfer as a contract call.
func swapContractPayload(c *ContractTx) {
	c.baseTx.ext = transferExt()
}

// TestSimulateRejectsNonTriggerContract is the P1 regression: Simulate must
// check the contract TYPE before re-decoding the parameter, and must reject
// without submitting any RPC.
func TestSimulateRejectsNonTriggerContract(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, err := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var gotReq *core.TriggerSmartContract
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		gotReq = in
		return &api.TransactionExtention{Result: okResult(), EnergyUsed: 1}, nil
	}
	swapContractPayload(ctxTx)
	est, err := ctxTx.Simulate(ctx)
	if err == nil || !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Fatalf("Simulate on a TransferContract payload = %v (est %+v), want tx.invalid_argument; misdecoded request %+v", err, est, gotReq)
	}
	if est != nil {
		t.Errorf("Estimate = %v, want nil alongside the error", est)
	}
	if gotReq != nil || f.simulateCalls.Load() != 0 {
		t.Errorf("TriggerConstantContract called with %+v, want no RPC at all", gotReq)
	}
	var te *tron.Error
	if !errors.As(err, &te) {
		t.Fatalf("want a *tron.Error, got %T", err)
	}
	if te.Op != "tx.Simulate" {
		t.Errorf("Op = %q, want tx.Simulate", te.Op)
	}
	if !strings.Contains(te.Hint, "not a TriggerSmartContract") || !strings.Contains(te.Hint, "Extension()") {
		t.Errorf("Hint = %q, want it to name the expected contract type and the Extension() escape hatch", te.Hint)
	}
}

// TestEstimateEnergyRejectsNonTriggerContract is the same regression on the
// EstimateEnergy entry point.
func TestEstimateEnergyRejectsNonTriggerContract(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, err := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var gotReq *core.TriggerSmartContract
	f.EstimateEnerg = func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
		gotReq = in
		return &api.EstimateEnergyMessage{Result: okResult(), EnergyRequired: 1}, nil
	}
	swapContractPayload(ctxTx)
	est, err := ctxTx.EstimateEnergy(ctx)
	if err == nil || !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Fatalf("EstimateEnergy on a TransferContract payload = %v (est %+v), want tx.invalid_argument; misdecoded request %+v", err, est, gotReq)
	}
	if gotReq != nil || f.estimateCalls.Load() != 0 {
		t.Errorf("EstimateEnergy called with %+v, want no RPC at all", gotReq)
	}
}

// TestPreviewCostRejectsNonTriggerContract pins that the third affected
// surface — CostPreview — cannot report a cost for an operation it never
// described correctly.
func TestPreviewCostRejectsNonTriggerContract(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, err := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	swapContractPayload(ctxTx)
	preview, err := PreviewCost(ctx, cp, ctxTx, testFrom)
	if err == nil || !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Fatalf("PreviewCost on a TransferContract payload = %v (preview %+v), want tx.invalid_argument", err, preview)
	}
	if f.simulateCalls.Load() != 0 || f.accountResourceCalls.Load() != 0 || f.energyPricesCalls.Load() != 0 {
		t.Errorf("PreviewCost ran node reads (simulate=%d resource=%d prices=%d), want none",
			f.simulateCalls.Load(), f.accountResourceCalls.Load(), f.energyPricesCalls.Load())
	}
}

func TestEnergyPriceCostOfBurnsAtCurrentRate(t *testing.T) {
	p := &EnergyPrice{SunPerEnergy: 100}
	cost, err := p.CostOf(13569) // the live-run energy: 13569 × 100 = 1,356,900 SUN
	if err != nil {
		t.Fatalf("CostOf: %v", err)
	}
	if int64(cost) != 1_356_900 {
		t.Errorf("CostOf(13569 @ 100) = %d, want 1356900", int64(cost))
	}
	// Zero energy burns nothing, at any price.
	zero, err := (&EnergyPrice{SunPerEnergy: 999_999_999}).CostOf(0)
	if err != nil || int64(zero) != 0 {
		t.Errorf("CostOf(0) = %d, %v; want 0, nil", int64(zero), err)
	}
}

func TestEnergyPriceCostOfOverflowIsAmountOverflow(t *testing.T) {
	p := &EnergyPrice{SunPerEnergy: 100}
	_, err := p.CostOf(math.MaxInt64)
	if err == nil || !tron.HasCode(err, tron.CodeAmountOverflow) {
		t.Errorf("CostOf(MaxInt64) = %v, want amount.overflow", err)
	}
	// Negative energy is not a burnable amount.
	_, err = p.CostOf(-1)
	if err == nil || !tron.HasCode(err, tron.CodeAmountNegative) {
		t.Errorf("CostOf(-1) = %v, want amount.negative", err)
	}
}

func TestEstimateEnergyReturnsInclusiveTotal(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
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
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	f.EstimateEnerg = func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
		return &api.EstimateEnergyMessage{Result: &api.Return{Result: false, Code: api.Return_CONTRACT_VALIDATE_ERROR}}, nil
	}
	_, err := ctxTx.EstimateEnergy(ctx)
	if err == nil || !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("EstimateEnergy node reject = %v, want a mapped code error", err)
	}
}

func TestPreviewCostCarriesBandwidthEstimateNote(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: okResult(), EnergyUsed: 5000, EnergyPenalty: 0}, nil
	}
	// Free bandwidth covers the bytes, so the preview needs no balance read.
	f.AccountResource = func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		return &api.AccountResourceMessage{FreeNetLimit: 600, FreeNetUsed: 0}, nil
	}
	f.Account = func(ctx context.Context, in *core.Account) (*core.Account, error) {
		return nil, fmt.Errorf("balance read must not happen when bandwidth is covered")
	}
	cp2, err := PreviewCost(ctx, cp, ctxTx, testFrom)
	if err != nil {
		t.Fatalf("CostPreview: %v", err)
	}
	if cp2.BandwidthNote != BandwidthEstimateNote {
		t.Errorf("BandwidthNote = %q, want %q", cp2.BandwidthNote, BandwidthEstimateNote)
	}
	if cp2.Bandwidth == nil {
		t.Fatal("Bandwidth half is nil, want the covered prediction")
	}
	if cp2.Bandwidth.ToBurn != 0 || cp2.Bandwidth.Burn != 0 {
		t.Errorf("covered bandwidth = %s, want zero burn", cp2.Bandwidth.String())
	}
	if cp2.Bandwidth.NetUsage != cp2.Bandwidth.BytesNeeded {
		t.Errorf("covered NetUsage = %d, want BytesNeeded %d", cp2.Bandwidth.NetUsage, cp2.Bandwidth.BytesNeeded)
	}
	// Covered bandwidth adds nothing: the floor is the energy burn alone.
	if cp2.TotalFloor != cp2.TronToBurn {
		t.Errorf("TotalFloor = %s, want TronToBurn %s", cp2.TotalFloor, cp2.TronToBurn)
	}
	for _, want := range []string{"bandwidth preview:", "total floor", "single-signature"} {
		if !strings.Contains(cp2.String(), want) {
			t.Errorf("String() = %q, want it to contain %q", cp2.String(), want)
		}
	}
}

// TestPreviewCostBandwidthBurnPricedAndGated: no bandwidth at all, so the
// whole byte count burns at the fake's 1000 sun/byte. The zero-balance
// default account makes the preview refuse with account.insufficient_bandwidth
// (the fiction rule — the node would reject the broadcast), and funding the
// account turns the same prediction into numbers.
func TestPreviewCostBandwidthBurnPricedAndGated(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: okResult(), EnergyUsed: 5000, EnergyPenalty: 0}, nil
	}
	f.Account = func(ctx context.Context, in *core.Account) (*core.Account, error) {
		return &core.Account{Address: in.GetAddress(), Balance: 0}, nil // exists, broke
	}
	_, err := PreviewCost(ctx, cp, ctxTx, testFrom)
	if !tron.HasCode(err, tron.CodeAccountInsufficientBandwidth) {
		t.Fatalf("unfunded burn preview = %v, want account.insufficient_bandwidth", err)
	}
	f.Account = func(ctx context.Context, in *core.Account) (*core.Account, error) {
		return &core.Account{Address: in.GetAddress(), Balance: 10_000_000}, nil // 10 TRX
	}
	preview, err := PreviewCost(ctx, cp, ctxTx, testFrom)
	if err != nil {
		t.Fatalf("funded burn preview: %v", err)
	}
	bw := preview.Bandwidth
	if bw.ToBurn != bw.BytesNeeded {
		t.Errorf("ToBurn = %d, want the full byte count %d (nothing covers it)", bw.ToBurn, bw.BytesNeeded)
	}
	if want := tron.SUN(bw.ToBurn * 1000); bw.Burn != want {
		t.Errorf("Burn = %s, want ToBurn×1000 = %s", bw.Burn, want)
	}
	if bw.NetUsage != 0 {
		t.Errorf("burn-path NetUsage = %d, want 0 (the node reports 0 on the burn path)", bw.NetUsage)
	}
	if want := preview.TronToBurn + bw.Burn; preview.TotalFloor != want {
		t.Errorf("TotalFloor = %s, want TronToBurn(%s) + Burn(%s)", preview.TotalFloor, preview.TronToBurn, bw.Burn)
	}
}

// TestPreviewCostBandwidthEstimateMatchesSignedSize pins the estimate's
// exactness for the single-signature case: the bytes PreviewCost priced
// before signing are exactly the bytes BandwidthSize measures on the signed
// transaction — the same model, one placeholder signature vs one real one.
func TestPreviewCostBandwidthEstimateMatchesSignedSize(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: okResult(), EnergyUsed: 5000, EnergyPenalty: 0}, nil
	}
	f.Account = func(ctx context.Context, in *core.Account) (*core.Account, error) {
		return &core.Account{Address: in.GetAddress(), Balance: 10_000_000}, nil
	}
	preview, err := PreviewCost(ctx, cp, ctxTx, testFrom)
	if err != nil {
		t.Fatalf("PreviewCost: %v", err)
	}
	signed, err := ctxTx.Sign(mustSigner(t, testKeyHex))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	measured, err := BandwidthSize(signed.Transaction())
	if err != nil {
		t.Fatalf("BandwidthSize(signed): %v", err)
	}
	if preview.Bandwidth.BytesNeeded != measured {
		t.Errorf("estimate BytesNeeded = %d, signed measurement = %d, want equal", preview.Bandwidth.BytesNeeded, measured)
	}
}

func TestPreviewCostThreeReadSequence(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	// Simulate returns the accurate dry-run energy (the actual VM cost).
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{
			Result:        okResult(),
			EnergyUsed:    643,
			EnergyPenalty: 200,
		}, nil
	}
	// EstimateEnergy is NOT called by PreviewCost (it is a conservative
	// fee-limit calculator, not the actual execution cost).  Setting it to a
	// different value should have no effect, and the call count must be 0.
	f.EstimateEnerg = func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
		return &api.EstimateEnergyMessage{Result: okResult(), EnergyRequired: 9999}, nil
	}
	var gotOwner *core.Account
	f.AccountResource = func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		gotOwner = in
		// Energy: 1500 available. Bandwidth: free quota covers the bytes,
		// so the fourth read's balance check stays lazy (no GetAccount).
		return &api.AccountResourceMessage{EnergyLimit: 2000, EnergyUsed: 500, FreeNetLimit: 600, FreeNetUsed: 0}, nil
	}
	// Two entries: the LATEST timestamp (1691500000000 → 420) must win.
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "1691400000000:410,1691500000000:420"}, nil
	}
	f.Account = func(ctx context.Context, in *core.Account) (*core.Account, error) {
		return nil, fmt.Errorf("balance read must not happen when both halves are covered")
	}
	cp2, err := PreviewCost(ctx, cp, ctxTx, testFrom)
	if err != nil {
		t.Fatalf("CostPreview: %v", err)
	}
	// EnergyNeeded comes from Simulate.Energy (TriggerConstantContract.EnergyUsed),
	// the accurate dry-run energy, NOT from the conservative EstimateEnergy RPC.
	if cp2.EnergyNeeded != 643 {
		t.Errorf("EnergyNeeded = %d, want 643 (from Simulate.Energy, the accurate execution cost)", cp2.EnergyNeeded)
	}
	if cp2.EnergyPenalty != 200 {
		t.Errorf("EnergyPenalty = %d, want 200 (from Simulate)", cp2.EnergyPenalty)
	}
	if cp2.EnergyBase != 443 {
		t.Errorf("EnergyBase = %d, want 443 (643 EnergyNeeded − 200 Penalty)", cp2.EnergyBase)
	}
	if cp2.EnergyAvailable != 1500 {
		t.Errorf("EnergyAvailable = %d, want 1500 (2000 limit − 500 used)", cp2.EnergyAvailable)
	}
	if cp2.EnergyToBuy != 0 {
		// 643 EnergyNeeded − 1500 available = 0 (no burn needed).
		t.Errorf("EnergyToBuy = %d, want 0 (643 needed ≤ 1500 available)", cp2.EnergyToBuy)
	}
	if cp2.SunPerEnergy != 420 {
		t.Errorf("SunPerEnergy = %d, want 420 (latest timestamp wins)", cp2.SunPerEnergy)
	}
	if cp2.TronToBurn != 0 {
		// 0 energy to buy, so 0 SUN burned.
		t.Errorf("TronToBurn = %d, want 0 (no energy to buy)", cp2.TronToBurn)
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
	// The read sequence: Simulate + GetAccountResource + GetEnergyPrices
	// (EstimateEnergy is NOT called).
	for name, n := range map[string]int32{
		"Simulate":           f.simulateCalls.Load(),
		"EstimateEnergy":     f.estimateCalls.Load(),
		"GetAccountResource": f.accountResourceCalls.Load(),
		"GetEnergyPrices":    f.energyPricesCalls.Load(),
		"GetBandwidthPrices": f.bandwidthPricesCalls.Load(),
	} {
		want := int32(1)
		if name == "EstimateEnergy" {
			want = 0
		}
		if n != want {
			t.Errorf("%s called %d times, want %d", name, n, want)
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
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return nil, status.Error(codes.DeadlineExceeded, "boom")
	}
	_, err := PreviewCost(ctx, cp, ctxTx, testFrom)
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
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "not-a-price-list"}, nil
	}
	_, err := PreviewCost(ctx, cp, ctxTx, testFrom)
	if err == nil || !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("PreviewCost with malformed prices = %v, want %q", err, tron.CodeContractBadMetadata)
	}
}

func TestPreviewCostEmptyPricesIsBadMetadata(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{}, nil
	}
	_, err := PreviewCost(ctx, cp, ctxTx, testFrom)
	if err == nil || !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("PreviewCost with empty prices = %v, want %q", err, tron.CodeContractBadMetadata)
	}
}

func TestPreviewCostBurnOverflowIsAmountOverflow(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ctx := t.Context()
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	// Simulate returns the accurate energy; use MaxInt64 to trigger overflow.
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: okResult(), EnergyUsed: math.MaxInt64, EnergyPenalty: 0}, nil
	}
	f.AccountResource = func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		return &api.AccountResourceMessage{}, nil // nothing available → buy MaxInt64
	}
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "1691500000000:3"}, nil
	}
	_, err := PreviewCost(ctx, cp, ctxTx, testFrom)
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
	_, err := PreviewCost(t.Context(), cp, nil, testFrom)
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
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	// Simulate is the accurate energy source: 5000 needed − 1500 available.
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: okResult(), EnergyUsed: 5000, EnergyPenalty: 0}, nil
	}
	f.AccountResource = func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		return &api.AccountResourceMessage{EnergyLimit: 2000, EnergyUsed: 500}, nil
	}
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "1691500000000:420"}, nil
	}
	low, err := ctxTx.WithFeeLimit(tron.TRX(1)) // 1_000_000 SUN < 1_470_000 SUN burn
	if err != nil {
		t.Fatalf("WithFeeLimit: %v", err)
	}
	_, err = PreviewCost(ctx, cp, low, testFrom)
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
	ctxTx, _ := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, nil, 0)
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: okResult(), EnergyUsed: 5000, EnergyPenalty: 0}, nil
	}
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		return &api.PricesResponseMessage{Prices: "1691500000000:420"}, nil
	}
	f.AccountResource = func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
		return &api.AccountResourceMessage{EnergyLimit: 2000, EnergyUsed: 500, FreeNetLimit: 600, FreeNetUsed: 0}, nil
	}
	high, err := ctxTx.WithFeeLimit(tron.TRX(5)) // 5_000_000 SUN > 1_470_000 SUN burn
	if err != nil {
		t.Fatalf("WithFeeLimit: %v", err)
	}
	preview, err := PreviewCost(ctx, cp, high, testFrom)
	if err != nil {
		t.Fatalf("PreviewCost with 5-TRX fee limit: %v", err)
	}
	if preview.TronToBurn != 1_470_000 {
		t.Errorf("TronToBurn = %d, want 1470000", preview.TronToBurn)
	}
}

func TestEstimateHasResult(t *testing.T) {
	if (&Estimate{}).HasResult() {
		t.Error("(&Estimate{}).HasResult() = true, want false")
	}
	if !(&Estimate{ConstantResult: [][]byte{{0x01}}}).HasResult() {
		t.Error("populated Estimate.HasResult() = false, want true")
	}
	var e *Estimate
	if e.HasResult() {
		t.Error("nil Estimate.HasResult() = true, want false (nil-safe)")
	}
}
