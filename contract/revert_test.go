package contract

// Revert classification tests: a TRON node answers a constant call that hit
// require()/revert()/panic() with result=true and the Solidity revert
// payload (Error(string) selector 08c379a0, or Panic(uint256) selector
// 4e487b71) in ConstantResult. Unclassifed, that payload falls through to
// the ABI output decode and surfaces as a bogus contract.arg_mismatch (or
// silently misdecodes) — the bug these tests pin. Decode offline receives
// only the blob and must not classify it without the node response marker;
// callers must use tx.Simulate's in-band Revert field first.

import (
	"bytes"
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// revertHex is a live-captured reason-bearing revert answer: the node
// returned result=true with message "REVERT opcode executed" and this
// Error("OLD") payload in ConstantResult (observe() on a SunSwap V3 pool
// with insufficient observation history).
const revertHex = "08c379a0" +
	"0000000000000000000000000000000000000000000000000000000000000020" +
	"0000000000000000000000000000000000000000000000000000000000000003" +
	"4f4c440000000000000000000000000000000000000000000000000000000000"

// panicHex is Panic(0x12) — Solidity division or modulo by zero.
const panicHex = "4e487b71" +
	"0000000000000000000000000000000000000000000000000000000000000012"

func revertBlob(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad revert hex: %v", err)
	}
	return b
}

// TestCallClassifiesReasonRevert: a result=true ConstantResult carrying
// Error("OLD") must surface as contract.reverted with the reason in the
// hint — not as the decode failure contract.arg_mismatch. The blob is
// split across two ConstantResult entries to pin the concatenation the
// node's chunking must not break.
func TestCallClassifiesReasonRevert(t *testing.T) {
	blob := revertBlob(t, revertHex)
	f := &fakeWallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExtention()
			ext.Result.Message = []byte("REVERT opcode executed")
			ext.ConstantResult = [][]byte{blob[:16], blob[16:]}
			return ext, nil
		},
	}
	i, err := NewInstance(newContractTestClient(t, f), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	_, err = i.Call(context.Background(), "balanceOf", AddressArg(testMainnetAddr))
	if !tron.HasCode(err, tron.CodeContractReverted) {
		t.Fatalf("Call on a reason-bearing revert: err = %v, want contract.reverted", err)
	}
	var te *tron.Error
	if ok := asTronError(err, &te); !ok {
		t.Fatalf("err is %T, want *tron.Error", err)
	}
	if !strings.Contains(te.Hint, "OLD") {
		t.Errorf("hint %q does not carry the decoded reason OLD", te.Hint)
	}
}

// TestCallClassifiesPanicRevert: Panic(0x12) classifies as
// contract.reverted too, with the human-readable panic reason.
func TestCallClassifiesPanicRevert(t *testing.T) {
	f := &fakeWallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExtention()
			ext.Result.Message = []byte("REVERT opcode executed")
			ext.ConstantResult = [][]byte{revertBlob(t, panicHex)}
			return ext, nil
		},
	}
	i, err := NewInstance(newContractTestClient(t, f), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	_, err = i.Call(context.Background(), "balanceOf", AddressArg(testMainnetAddr))
	if !tron.HasCode(err, tron.CodeContractReverted) {
		t.Fatalf("Call on a panic revert: err = %v, want contract.reverted", err)
	}
	var te *tron.Error
	if ok := asTronError(err, &te); !ok {
		t.Fatalf("err is %T, want *tron.Error", err)
	}
	if !strings.Contains(te.Hint, "division or modulo by zero") {
		t.Errorf("hint %q does not carry the panic reason", te.Hint)
	}
}

// TestDecodeDoesNotInferRevertFromBlob: Decode receives only ABI output bytes,
// not the node response that produced them. It must keep using ordinary ABI
// decoding; otherwise a legitimate method return can be mistaken for a revert.
func TestDecodeDoesNotInferRevertFromBlob(t *testing.T) {
	i, err := NewInstance(newContractTestClient(t, &fakeWallet{}), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	if err := i.UseABI(testABI); err != nil {
		t.Fatalf("UseABI: %v", err)
	}
	_, err = i.Decode("transfer", revertBlob(t, revertHex))
	if !tron.HasCode(err, tron.CodeContractArgMismatch) {
		t.Fatalf("Decode of a revert-shaped blob: err = %v, want ordinary contract.arg_mismatch", err)
	}
}

// TestCallDoesNotMisclassifyValidRevertShapedOutput: a successful method can
// return a static array whose bytes happen to form a valid Error(string)
// payload at the start. Call must use the node's execution-failure signal,
// not the payload alone, before returning contract.reverted.
func TestCallDoesNotMisclassifyValidRevertShapedOutput(t *testing.T) {
	const arrayABI = `[{"type":"function","name":"hashes","stateMutability":"view",
	  "inputs":[],"outputs":[{"name":"","type":"bytes32[4]"}]}]`
	data := make([]byte, 128)
	copy(data, revertBlob(t, revertHex))
	f := &fakeWallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExtention()
			ext.ConstantResult = [][]byte{data}
			return ext, nil
		},
	}
	i, err := NewInstance(newContractTestClient(t, f), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	if err := i.UseABI(arrayABI); err != nil {
		t.Fatalf("UseABI: %v", err)
	}
	res, err := i.Call(context.Background(), "hashes")
	if err != nil {
		t.Fatalf("Call of a successful revert-shaped return: %v", err)
	}
	got, err := res.BytesSlice()
	if err != nil {
		t.Fatalf("BytesSlice(): %v", err)
	}
	if len(got) != 4 || !bytes.Equal(got[0], data[:32]) {
		t.Fatalf("decoded bytes32[4] = %x, want first word %x", got, data[:32])
	}
}

// TestDecodeRevertShapedBytesNotMisclassified: data that begins with the
// Error selector but does not decode as a valid Error(string) payload is
// an ordinary return value (a bytes32 that happens to start with 08c379a0)
// and must still decode normally. The selector sniff alone is not the
// classifier; the payload must unpack as a revert too.
func TestDecodeRevertShapedBytesNotMisclassified(t *testing.T) {
	const hashABI = `[{"type":"function","name":"hash","stateMutability":"view",
	  "inputs":[],"outputs":[{"name":"","type":"bytes32"}]}]`
	i, err := NewInstance(newContractTestClient(t, &fakeWallet{}), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	if err := i.UseABI(hashABI); err != nil {
		t.Fatalf("UseABI: %v", err)
	}
	// 08c379a0 + 28 bytes of 0xaa: selector-prefixed but not a valid
	// Error(string) encoding (no 32-byte-aligned string payload follows).
	data := revertBlob(t, "08c379a0"+strings.Repeat("aa", 28))
	res, err := i.Decode("hash", data)
	if err != nil {
		t.Fatalf("Decode of selector-shaped bytes32: %v, want normal decode", err)
	}
	got, err := res.Bytes()
	if err != nil {
		t.Fatalf("Bytes(): %v", err)
	}
	if hex.EncodeToString(got) != "08c379a0"+strings.Repeat("aa", 28) {
		t.Errorf("decoded bytes32 = %x, want the input verbatim", got)
	}
}
