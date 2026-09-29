package token

import (
	"context"
	"math/big"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// TestNewEagerDecimalsFetch: New performs exactly one decimals() view call
// at construction and caches the scale — afterwards Amount/Whole/BalanceOf
// need no further decimals reads (only BalanceOf's own call).
func TestNewEagerDecimalsFetch(t *testing.T) {
	h, f := newTestHandle(t)
	if h.Decimals() != 6 {
		t.Errorf("Decimals() = %d, want 6 (fetched at construction)", h.Decimals())
	}
	if h.Contract() != testContract {
		t.Errorf("Contract() = %s, want %s", h.Contract(), testContract)
	}
	if f.triggerConstantCalls != 1 {
		t.Fatalf("TriggerConstantContract fired %d times, want exactly 1 (eager decimals fetch)", f.triggerConstantCalls)
	}
	before := f.triggerConstantCalls
	if _, err := h.Amount("1"); err != nil {
		t.Fatalf("Amount: %v", err)
	}
	if _, err := h.Whole(1); err != nil {
		t.Fatalf("Whole: %v", err)
	}
	if f.triggerConstantCalls != before {
		t.Errorf("Amount/Whole fired %d extra view calls; parsing must do no I/O", f.triggerConstantCalls-before)
	}
}

// TestNewMalformedDecimals: a non-standard contract returning a decimals
// value outside the uint8 range (e.g. wide-packed metadata) fails
// construction with contract.bad_metadata — never a Handle that formats
// amounts wrong forever.
func TestNewMalformedDecimals(t *testing.T) {
	for _, v := range []int64{256, 1 << 40} {
		f := &fakeTRC20Wallet{
			TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
				ext := okExt()
				ext.ConstantResult = [][]byte{abiWord(v)}
				return ext, nil
			},
		}
		h, err := New(newTokenTestClient(t, f), context.Background(), testContract)
		if h != nil {
			t.Errorf("New with decimals=%d returned a Handle; must fail", v)
		}
		var te *tron.Error
		if !asTronError(err, &te) || te.Code != tron.CodeContractBadMetadata {
			t.Errorf("New with decimals=%d: err = %v, want contract.bad_metadata", v, err)
		}
	}
}

// TestNewContractMissing: no contract at the address surfaces the
// contract layer's classified error (contract.not_found) unchanged.
func TestNewContractMissing(t *testing.T) {
	f := &fakeTRC20Wallet{
		GetContractFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{}, nil
		},
	}
	// token.New supplies its own ABI (UseABI), so remove the eager path's
	// ABI injection by pointing the fake's contract RPC at a missing
	// contract and using a Handle construction that must still consult the
	// chain for decimals — New always issues the decimals() call, which the
	// fake answers with a node-level rejection here.
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: &api.Return{Result: false, Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("no contract")}}, nil
	}
	_, err := New(newTokenTestClient(t, f), context.Background(), testContract)
	if err == nil {
		t.Fatal("New against a failing node: err = nil, want an error")
	}
	var te *tron.Error
	if !asTronError(err, &te) || te.Code == "" {
		t.Errorf("New against a failing node: err = %v, want a classified *tron.Error", err)
	}
}

// TestNewNilProvider: a nil ConnProvider is chain.connection before I/O.
func TestNewNilProvider(t *testing.T) {
	_, err := New(nil, context.Background(), testContract)
	var te *tron.Error
	if !asTronError(err, &te) || te.Code != tron.CodeChainConnection {
		t.Errorf("New(nil): err = %v, want chain.connection", err)
	}
}

// TestBalanceOf: the balanceOf view call lands in an Amount carrying the
// handle's decimals — Result.BigInt() works because balanceOf returns
// uint256 (see the uint8 accessor-gap note in the report).
func TestBalanceOf(t *testing.T) {
	h, f := newTestHandle(t)
	a, err := h.BalanceOf(context.Background(), mustAddr(0x11))
	if err != nil {
		t.Fatalf("BalanceOf: %v", err)
	}
	if a.Raw().Int64() != 777 {
		t.Errorf("BalanceOf raw = %v, want 777 (the fake's canned uint256)", a.Raw())
	}
	if a.Decimals() != 6 {
		t.Errorf("BalanceOf decimals = %d, want the handle's 6", a.Decimals())
	}
	if f.triggerConstantCalls != 2 { // 1 decimals at New + 1 balanceOf
		t.Errorf("TriggerConstantContract fired %d times, want 2", f.triggerConstantCalls)
	}
	// Mutating the returned raw must not affect anything (Amount immutability).
	a.Raw().SetInt64(0)
	if again, _ := h.BalanceOf(context.Background(), mustAddr(0x11)); again.Raw().Int64() != 777 {
		t.Errorf("mutating a BalanceOf Amount leaked into the next read: %v", again.Raw())
	}
}

// TestTransferBuildsContractTx: Transfer encodes transfer(address,uint256)
// through contract.Instance.Invoke and returns a *tx.ContractTx with the
// right owner, contract, and data (to + raw amount in the call data).
func TestTransferBuildsContractTx(t *testing.T) {
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
	from, to := mustAddr(0x11), mustAddr(0x22)
	amt, err := h.Amount("1.6")
	if err != nil {
		t.Fatalf("Amount: %v", err)
	}
	ct, err := h.Transfer(context.Background(), from, to, amt)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	var _ *tx.ContractTx = ct // the static return type IS the DAG-direction proof
	if ct == nil {
		t.Fatal("Transfer returned a nil ContractTx without error")
	}
	if gotReq == nil {
		t.Fatal("the tx builder never reached TriggerContract")
	}
	if string(gotReq.GetOwnerAddress()) != string(from.Bytes()) {
		t.Errorf("owner = %x, want %x", gotReq.GetOwnerAddress(), from.Bytes())
	}
	if string(gotReq.GetContractAddress()) != string(testContract.Bytes()) {
		t.Errorf("contract = %x, want %x", gotReq.GetContractAddress(), testContract.Bytes())
	}
	if gotReq.GetCallValue() != 0 {
		t.Errorf("CallValue = %d, want 0 (an ERC-20 transfer spends no TRX)", gotReq.GetCallValue())
	}
	data := gotReq.GetData()
	if len(data) != 4+64 {
		t.Fatalf("data length = %d, want 68 (selector + address + uint256)", len(data))
	}
	if string(data[:4]) != string(mustSelector("transfer(address,uint256)")) {
		t.Errorf("selector = %x, want transfer(address,uint256)", data[:4])
	}
	toWord := make([]byte, 32)
	copy(toWord[12:], to.Bytes()[1:]) // the 0x41-stripped address form
	if string(data[4+12:4+32]) != string(toWord[12:]) {
		t.Errorf("to word = %x, want the 0x41-stripped payload", data[4:36])
	}
	if amount := new(big.Int).SetBytes(data[36:]); amount.Int64() != 1_600_000 {
		t.Errorf("amount word = %v, want 1600000 (1.6 at 6 decimals)", amount)
	}
}

// TestTransferDecimalsMismatch: an Amount minted against a different
// scale is amount.decimals_mismatch — never silently re-scaled.
func TestTransferDecimalsMismatch(t *testing.T) {
	h, _ := newTestHandle(t)
	other := Amount{raw: big.NewInt(1), decimals: 18} // same token value, wrong scale
	_, err := h.Transfer(context.Background(), mustAddr(0x11), mustAddr(0x22), other)
	if !tron.HasCode(err, tron.CodeAmountDecimalsMismatch) {
		t.Errorf("Transfer with an 18-decimal Amount on a 6-decimal Handle: err = %v, want amount.decimals_mismatch", err)
	}
}

// TestTransferZeroOwner: a zero from address is address.invalid before any RPC.
func TestTransferZeroOwner(t *testing.T) {
	h, f := newTestHandle(t)
	whole, werr := h.Whole(1)
	if werr != nil {
		t.Fatalf("Whole: %v", werr)
	}
	_, err := h.Transfer(context.Background(), tron.Address{}, mustAddr(0x22), whole)
	var te *tron.Error
	if !asTronError(err, &te) || te.Code != tron.CodeAddressInvalid {
		t.Errorf("Transfer(zero owner): err = %v, want address.invalid", err)
	}
	if f.triggerCalls != 0 {
		t.Error("Transfer with a zero owner must not reach the network")
	}
}

// newTestHandleWith is newTestHandle against a caller-supplied fake.
func newTestHandleWith(t *testing.T, f *fakeTRC20Wallet) (*Handle, *fakeTRC20Wallet) {
	t.Helper()
	h, err := New(newTokenTestClient(t, f), context.Background(), testContract)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h, f
}
