package tx

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func signedNative(t *testing.T, f *fakeWalletServer) (*NativeTx, string) {
	t.Helper()
	cp := newTxTestClient(t, f)
	ntx, err := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	signed, err := ntx.Sign(mustSigner(t, testKeyHex))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed, signed.ID()
}

func TestBroadcastSuccessReceipt(t *testing.T) {
	f := &fakeWalletServer{}
	ntx, txid := signedNative(t, f)
	r, err := Broadcast(newTxTestClient(t, f), t.Context(), ntx)
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if r.TxID != txid {
		t.Errorf("Receipt.TxID = %q, want %q", r.TxID, txid)
	}
	if !r.OK() {
		t.Errorf("Receipt.Code = %q, want \"\" (success)", r.Code)
	}
	if r.NodeCode != "SUCCESS" {
		t.Errorf("NodeCode = %q, want SUCCESS", r.NodeCode)
	}
	// A successful broadcast is final: no reconciliation poll happens.
	if n := f.txInfoCalls.Load(); n != 0 {
		t.Errorf("GetTransactionInfoById called %d times on success, want 0", n)
	}
}

func TestBroadcastNodeRejectIsAReturnNotAnError(t *testing.T) {
	f := &fakeWalletServer{
		Broadcast: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return &api.Return{
				Result:  false,
				Code:    api.Return_CONTRACT_VALIDATE_ERROR,
				Message: []byte("contract validate error"),
			}, nil
		},
	}
	ntx, _ := signedNative(t, f)
	r, err := Broadcast(newTxTestClient(t, f), t.Context(), ntx)
	if err != nil {
		t.Fatalf("Broadcast: %v (a node rejection is an answer, not an error)", err)
	}
	if r.OK() {
		t.Errorf("Receipt.Code = %q, want the mapped rejection code", r.Code)
	}
	if r.Code != tron.CodeTxInvalidArgument {
		t.Errorf("Receipt.Code = %q, want tx.invalid_argument (CONTRACT_VALIDATE_ERROR)", r.Code)
	}
	if r.NodeCode != "CONTRACT_VALIDATE_ERROR" {
		t.Errorf("NodeCode = %q, want CONTRACT_VALIDATE_ERROR", r.NodeCode)
	}
}

func TestBroadcastPreflightChecks(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	ctx := t.Context()

	if _, err := Broadcast(cp, ctx, nil); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("nil tx: err = %v, want tx.invalid_argument", err)
	}

	unsigned, _ := BuildTransfer(cp, ctx, testFrom, testTo, 1)
	if _, err := Broadcast(cp, ctx, unsigned); !tron.HasCode(err, tron.CodeTxNoSigner) {
		t.Errorf("unsigned: err = %v, want tx.no_signer", err)
	}

	// expiration is checked before signer: mutate raw_data via the escape
	// hatch and keep the tx unsigned.
	expired, _ := BuildTransfer(cp, ctx, testFrom, testTo, 1)
	expired.Transaction().GetRawData().Expiration = time.Now().Add(-time.Minute).UnixMilli()
	if _, err := Broadcast(cp, ctx, expired); !tron.HasCode(err, tron.CodeTxExpired) {
		t.Errorf("expired: err = %v, want tx.expired", err)
	}

	// a transaction stripped of its contract message is rejected
	contractless, _ := BuildTransfer(cp, ctx, testFrom, testTo, 1)
	contractless.Transaction().GetRawData().Contract = nil
	if _, err := Broadcast(cp, ctx, contractless); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("no contract message: err = %v, want tx.invalid_argument", err)
	}
}

// THE §6.4 TEST: on an ambiguous chain.timeout the broadcast did ONE
// reconciliation poll; if the transaction was found the real receipt is
// returned, if not the caller gets chain.unconfirmed + TxID + Next=Wait —
// and NOTHING is retried or resent (resending/rebuilding is the double-spend
// path; see package doc).
func TestBroadcastTimeoutThenFoundReturnsReceipt(t *testing.T) {
	f := &fakeWalletServer{
		Broadcast: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			// first attempt: ambiguous transport failure
			return nil, status.Error(codes.DeadlineExceeded, "fake broadcast deadline")
		},
		TxInfo: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			// the reconciliation poll finds the transaction landed
			return foundInfo(in.Value), nil
		},
	}
	ntx, txid := signedNative(t, f)
	r, err := Broadcast(newTxTestClient(t, f), t.Context(), ntx)
	if err != nil {
		t.Fatalf("Broadcast after timeout+found: %v", err)
	}
	if r.TxID != txid {
		t.Errorf("Receipt.TxID = %q, want %q (the polled receipt, not an error)", r.TxID, txid)
	}
	if !r.OK() {
		t.Errorf("Receipt.Code = %q, want success", r.Code)
	}
	// exactly ONE broadcast and ONE reconciliation poll — no retry loop
	if n := f.broadcastCalls.Load(); n != 1 {
		t.Errorf("BroadcastTransaction called %d times, want 1 (no resend)", n)
	}
	if n := f.txInfoCalls.Load(); n != 1 {
		t.Errorf("GetTransactionInfoById called %d times, want 1 (single reconciliation poll)", n)
	}
}

func TestBroadcastTimeoutThenNotFoundUnconfirmed(t *testing.T) {
	f := &fakeWalletServer{
		Broadcast: func(ctx context.Context, in *core.Transaction) (*api.Return, error) {
			return nil, status.Error(codes.DeadlineExceeded, "fake broadcast deadline")
		},
		// TxInfo nil -> default handler returns empty info = not found
	}
	ntx, txid := signedNative(t, f)
	_, err := Broadcast(newTxTestClient(t, f), t.Context(), ntx)
	if !tron.HasCode(err, tron.CodeChainUnconfirmed) {
		t.Fatalf("err = %v, want chain.unconfirmed (the double-spend-fix answer)", err)
	}
	var te *tron.Error
	if !errors.As(err, &te) {
		t.Fatalf("err is not *tron.Error")
	}
	if te.TxID != txid {
		t.Errorf("te.TxID = %q, want %q", te.TxID, txid)
	}
	if te.Next != tron.ActionWait {
		t.Errorf("te.Next = %v, want ActionWait", te.Next)
	}
	// one broadcast, one poll, no retries
	if n := f.broadcastCalls.Load(); n != 1 {
		t.Errorf("BroadcastTransaction called %d times, want 1", n)
	}
	if n := f.txInfoCalls.Load(); n != 1 {
		t.Errorf("GetTransactionInfoById called %d times, want 1 (single poll)", n)
	}
}

func TestWaitReturnsFullNodeReceipt(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ntx, _ := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	f.TxInfo = func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
		return foundInfo(in.Value), nil
	}
	r, err := Wait(cp, t.Context(), ntx.ID())
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if r.TxID != ntx.ID() {
		t.Errorf("TxID = %q, want %q", r.TxID, ntx.ID())
	}
	if r.Solidified() {
		t.Error("Wait receipt must not report Solidified")
	}
}

func TestWaitForSolidUsesSolidityEndpoint(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ntx, _ := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	f.TxInfoSolidity = func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
		return foundInfo(in.Value), nil
	}
	r, err := WaitForSolid(cp, t.Context(), ntx.ID())
	if err != nil {
		t.Fatalf("WaitForSolid: %v", err)
	}
	if !r.Solidified() {
		t.Error("WaitForSolid receipt must report Solidified")
	}
	if n := f.txInfoCalls.Load(); n != 0 {
		t.Errorf("WaitForSolid hit the FullNode endpoint %d times, want 0", n)
	}
}

func TestWaitPollsUntilFound(t *testing.T) {
	f := &fakeWalletServer{}
	cp := newTxTestClient(t, f)
	ntx, _ := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	var polls int
	f.TxInfo = func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
		polls++
		if polls < 3 {
			return &core.TransactionInfo{}, nil // not yet
		}
		return foundInfo(in.Value), nil
	}
	r, err := Wait(cp, t.Context(), ntx.ID())
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if polls != 3 {
		t.Errorf("polls = %d, want 3", polls)
	}
	if r.TxID != ntx.ID() {
		t.Errorf("TxID = %q, want %q", r.TxID, ntx.ID())
	}
}

func TestWaitInvalidTxid(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	if _, err := Wait(cp, t.Context(), "not-hex"); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("err = %v, want tx.invalid_argument", err)
	}
	if _, err := Wait(cp, t.Context(), ""); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("empty txid: err = %v, want tx.invalid_argument", err)
	}
}

func TestWaitContextTimeout(t *testing.T) {
	f := &fakeWalletServer{} // default: not found
	cp := newTxTestClient(t, f)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, err := Wait(cp, ctx, hex.EncodeToString(repeat(0xAB, 32)))
	if !tron.HasCode(err, tron.CodeChainTimeout) {
		t.Fatalf("err = %v, want chain.timeout", err)
	}
}
