package token

// Tests for the extended TRC-20 surface: Approve (build), Allowance
// (read), and the Name/Symbol/TotalSupply metadata reads. Patterns mirror
// TestTransferBuildsContractTx / TestBalanceOf.

import (
	"context"
	"math/big"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// abiString encodes a Go string as ABI dynamic-string words: offset (32),
// length, then 32-byte-padded data.
func abiString(s string) []byte {
	out := abiWord(32)
	out = append(out, abiWord(int64(len(s)))...)
	padded := make([]byte, (len(s)+31)/32*32)
	copy(padded, s)
	return append(out, padded...)
}

func stringHandle(t *testing.T, values map[string][]byte) (*Handle, *fakeTRC20Wallet) {
	t.Helper()
	f := &fakeTRC20Wallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExt()
			switch string(in.GetData()[:4]) {
			case string(mustSelector("decimals()")):
				ext.ConstantResult = [][]byte{abiWord(6)}
			default:
				for sel, v := range values {
					if string(in.GetData()[:4]) == sel {
						ext.ConstantResult = [][]byte{v}
					}
				}
			}
			return ext, nil
		},
	}
	h, err := New(newTokenTestClient(t, f), context.Background(), testContract)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h, f
}

func TestApproveBuildsContractTx(t *testing.T) {
	var gotReq *core.TriggerSmartContract
	f := &fakeTRC20Wallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExt()
			if string(in.GetData()[:4]) == string(mustSelector("decimals()")) {
				ext.ConstantResult = [][]byte{abiWord(6)}
			}
			return ext, nil
		},
		Trigger: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			gotReq = in
			return triggerExt(), nil
		},
	}
	h, _ := newTestHandleWith(t, f)
	owner, spender := mustAddr(0x11), mustAddr(0x33)
	amt, err := h.Amount("2.5")
	if err != nil {
		t.Fatalf("Amount: %v", err)
	}
	ct, err := h.Approve(context.Background(), owner, spender, amt)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if ct == nil {
		t.Fatal("Approve returned a nil ContractTx without error")
	}
	data := gotReq.GetData()
	if len(data) != 4+64 {
		t.Fatalf("data length = %d, want 68 (selector + address + uint256)", len(data))
	}
	if string(data[:4]) != string(mustSelector("approve(address,uint256)")) {
		t.Errorf("selector = %x, want approve(address,uint256)", data[:4])
	}
	spenderWord := make([]byte, 32)
	copy(spenderWord[12:], spender.Bytes()[1:])
	if string(data[4:36]) != string(spenderWord) {
		t.Errorf("spender word = %x, want the 0x41-stripped payload", data[4:36])
	}
	if amount := new(big.Int).SetBytes(data[36:]); amount.Int64() != 2_500_000 {
		t.Errorf("amount word = %v, want 2500000 (2.5 at 6 decimals)", amount)
	}
	if gotReq.GetCallValue() != 0 {
		t.Errorf("CallValue = %d, want 0", gotReq.GetCallValue())
	}
}

func TestApproveValidation(t *testing.T) {
	h, f := newTestHandle(t)
	whole, err := h.Whole(1)
	if err != nil {
		t.Fatalf("Whole: %v", err)
	}
	if _, err := h.Approve(context.Background(), tron.Address{}, mustAddr(0x33), whole); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("Approve(zero owner): want address.invalid, got %v", err)
	}
	other := Amount{raw: big.NewInt(1), decimals: 18}
	if _, err := h.Approve(context.Background(), mustAddr(0x11), mustAddr(0x33), other); !tron.HasCode(err, tron.CodeAmountDecimalsMismatch) {
		t.Errorf("Approve(wrong scale): want amount.decimals_mismatch, got %v", err)
	}
	if f.triggerCalls != 0 {
		t.Error("rejected Approves must not reach the network")
	}
}

func TestAllowance(t *testing.T) {
	sel := string(mustSelector("allowance(address,address)"))
	h, _ := stringHandle(t, map[string][]byte{sel: abiWord(424242)})
	a, err := h.Allowance(context.Background(), mustAddr(0x11), mustAddr(0x33))
	if err != nil {
		t.Fatalf("Allowance: %v", err)
	}
	if a.Raw().Int64() != 424242 || a.Decimals() != 6 {
		t.Errorf("Allowance = %v @ %d decimals, want 424242 @ 6", a.Raw(), a.Decimals())
	}
}

func TestNameSymbolTotalSupply(t *testing.T) {
	h, _ := stringHandle(t, map[string][]byte{
		string(mustSelector("name()")):        abiString("Tether USD"),
		string(mustSelector("symbol()")):      abiString("USDT"),
		string(mustSelector("totalSupply()")): abiWord(1_000_000_000_000),
	})
	if name, err := h.Name(context.Background()); err != nil || name != "Tether USD" {
		t.Errorf("Name = %q, %v; want Tether USD, nil", name, err)
	}
	if sym, err := h.Symbol(context.Background()); err != nil || sym != "USDT" {
		t.Errorf("Symbol = %q, %v; want USDT, nil", sym, err)
	}
	sup, err := h.TotalSupply(context.Background())
	if err != nil {
		t.Fatalf("TotalSupply: %v", err)
	}
	if sup.Raw().Int64() != 1_000_000_000_000 || sup.Decimals() != 6 {
		t.Errorf("TotalSupply = %v @ %d, want 1000000000000 @ 6", sup.Raw(), sup.Decimals())
	}
}

func TestMetadataWrongTypeIsBadMetadata(t *testing.T) {
	// name() answering a uint256 instead of a string.
	h, _ := stringHandle(t, map[string][]byte{string(mustSelector("name()")): abiWord(7)})
	if _, err := h.Name(context.Background()); !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("Name(uint256): want contract.bad_metadata, got %v", err)
	}
	// totalSupply() answering truncated data.
	h2, _ := stringHandle(t, map[string][]byte{string(mustSelector("totalSupply()")): abiWord(1)[:10]})
	if _, err := h2.TotalSupply(context.Background()); !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("TotalSupply(truncated): want contract.bad_metadata, got %v", err)
	}
	// NOTE: ABI outputs are not self-describing — a wrong-but-well-formed
	// word (e.g. a string payload read as uint256) decodes silently to
	// the offset word, exactly as on Ethereum. Only undecodable answers
	// are rejected; well-formed confusion is inherent to the ABI.
}
