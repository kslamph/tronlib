package contract

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
	"golang.org/x/crypto/sha3"
)

// mustSelector computes keccak256(sig)[:4] — the call selector the fake's
// TriggerConstant handler dispatches on.
func mustSelector(sig string) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(sig))
	return hash.Sum(nil)[:4]
}

// asTronError wraps errors.As for *tron.Error.
func asTronError(err error, target **tron.Error) bool { return errors.As(err, target) }

// TestNewInstanceValidation: construction rejects a nil ConnProvider and a
// zero address without touching the network.
func TestNewInstanceValidation(t *testing.T) {
	if _, err := NewInstance(nil, testContractAddress); !tron.HasCode(err, tron.CodeChainConnection) {
		t.Errorf("NewInstance(nil, ...): err = %v, want chain.connection", err)
	}
	c := newContractTestClient(t, &fakeWallet{})
	if _, err := NewInstance(c, tron.Address{}); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("NewInstance(c, zero): err = %v, want address.invalid", err)
	}
}

// TestUseABI: a valid ABI loads and is discoverable; empty/garbage ABIs are
// contract.bad_abi.
func TestUseABI(t *testing.T) {
	i := newInstanceWithTestABI(t)
	if got := i.ABI(); got != testABI {
		t.Errorf("ABI() = %q..., want the supplied JSON", got[:min(len(got), 40)])
	}
	want := []string{"balanceOf", "getOwner", "name", "noop", "transfer"}
	got := i.Methods()
	if len(got) != len(want) {
		t.Fatalf("Methods() = %v, want %v", got, want)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Errorf("Methods()[%d] = %q, want %q (sorted)", idx, got[idx], want[idx])
		}
	}

	c := newContractTestClient(t, &fakeWallet{})
	i2, _ := NewInstance(c, testContractAddress)
	for name, json := range map[string]string{
		"empty":      "   ",
		"garbage":    `{"type":"function","name":"broken","inputs":[{"type":"notatype"}]}`,
		"no-entries": `[]`,
	} {
		if err := i2.UseABI(json); !tron.HasCode(err, tron.CodeContractBadABI) {
			t.Errorf("UseABI(%s): err = %v, want contract.bad_abi", name, err)
		}
	}
}

// TestLazyABIFetch: without UseABI, the first Call performs GetContract
// exactly once and succeeds; Methods()/ABI() are populated afterwards.
func TestLazyABIFetch(t *testing.T) {
	f := &fakeWallet{}
	c := newContractTestClient(t, f)
	i, err := NewInstance(c, testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	if i.Methods() != nil || i.ABI() != "" {
		t.Error("Methods()/ABI() must be empty before any ABI load")
	}
	ctx := context.Background()
	res, err := i.Call(ctx, "noop") // no outputs — decodes to an IsNil Result
	if err != nil {
		t.Fatalf("Call(noop) with lazy fetch: %v", err)
	}
	if !res.IsNil() {
		t.Error("Call(noop) should decode to an IsNil Result")
	}
	if f.getContractCalls.Load() != 1 {
		t.Errorf("GetContract called %d times, want exactly 1 (lazily, on first use)", f.getContractCalls.Load())
	}
	if i.ABI() == "" || len(i.Methods()) != 5 {
		t.Errorf("after the lazy fetch: ABI() empty or Methods() = %v, want 5 methods", i.Methods())
	}
}

// TestLazyFetchCannedResult proves the fetched ABI actually drives decoding:
// balanceOf's uint256 ConstantResult lands in Result.BigInt().
func TestLazyFetchCannedResult(t *testing.T) {
	f := &fakeWallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExtention()
			ext.ConstantResult = [][]byte{abiUint256(777)}
			return ext, nil
		},
	}
	i, err := NewInstance(newContractTestClient(t, f), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	res, err := i.Call(context.Background(), "balanceOf", AddressArg(testMainnetAddr))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	v, err := res.BigInt()
	if err != nil || v.Int64() != 777 {
		t.Errorf("BigInt() = (%v, %v), want (777, nil)", v, err)
	}
	if f.getContractCalls.Load() != 1 {
		t.Errorf("GetContract called %d times, want 1", f.getContractCalls.Load())
	}
}

// TestLazyFetchNoABI: the contract exists on-chain but publishes no ABI —
// contract.no_abi.
func TestLazyFetchNoABI(t *testing.T) {
	f := &fakeWallet{
		GetContractFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{Bytecode: []byte{0x60, 0x80}}, nil // exists, no ABI entries
		},
	}
	i, _ := NewInstance(newContractTestClient(t, f), testContractAddress)
	_, err := i.Call(context.Background(), "name")
	if !tron.HasCode(err, tron.CodeContractNoABI) {
		t.Errorf("Call with no on-chain ABI: err = %v, want contract.no_abi", err)
	}
}

// TestLazyFetchNotFound: nothing at the address — contract.not_found.
func TestLazyFetchNotFound(t *testing.T) {
	f := &fakeWallet{
		GetContractFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{}, nil
		},
	}
	i, _ := NewInstance(newContractTestClient(t, f), testContractAddress)
	_, err := i.Call(context.Background(), "name")
	if !tron.HasCode(err, tron.CodeContractNotFound) {
		t.Errorf("Call for a missing contract: err = %v, want contract.not_found", err)
	}
}

// TestLazyFetchBadABI: the on-chain ABI does not parse — contract.bad_abi.
func TestLazyFetchBadABI(t *testing.T) {
	f := &fakeWallet{
		GetContractFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{
				Abi: &core.SmartContract_ABI{Entrys: []*core.SmartContract_ABI_Entry{
					{Name: "broken", Type: core.SmartContract_ABI_Entry_Function,
						Inputs: []*core.SmartContract_ABI_Entry_Param{{Type: "notatype"}}},
				}},
			}, nil
		},
	}
	i, _ := NewInstance(newContractTestClient(t, f), testContractAddress)
	_, err := i.Call(context.Background(), "broken")
	if !tron.HasCode(err, tron.CodeContractBadABI) {
		t.Errorf("Call with unparseable on-chain ABI: err = %v, want contract.bad_abi", err)
	}
}

// TestCallRequestShape pins the TriggerConstantContract request the Call
// path sends: the contract's address, the encoded data, and the null owner
// (see the package doc — view calls spend nothing but the RPC requires an
// owner_address).
func TestCallRequestShape(t *testing.T) {
	var gotReq *core.TriggerSmartContract
	f := &fakeWallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			gotReq = in
			ext := okExtention()
			ext.ConstantResult = [][]byte{abiUint256(777)}
			return ext, nil
		},
	}
	i := newInstanceWithTestABIOn(t, f)
	res, err := i.Call(context.Background(), "balanceOf", AddressArg(testMainnetAddr))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if v, _ := res.BigInt(); v.Int64() != 777 {
		t.Errorf("BigInt() = %v, want 777", v)
	}
	if gotReq == nil {
		t.Fatal("TriggerConstantContract was not reached")
	}
	if string(gotReq.GetContractAddress()) != string(testContractAddress.Bytes()) {
		t.Errorf("ContractAddress = %x, want %x", gotReq.GetContractAddress(), testContractAddress.Bytes())
	}
	if gotReq.GetCallValue() != 0 {
		t.Errorf("CallValue = %d, want 0 (a read cannot spend)", gotReq.GetCallValue())
	}
	owner := gotReq.GetOwnerAddress()
	if len(owner) != 21 || owner[0] != 0x41 {
		t.Fatalf("OwnerAddress = %x, want a 21-byte 0x41-prefixed form", owner)
	}
	for _, b := range owner[1:] {
		if b != 0 {
			t.Errorf("OwnerAddress payload = %x, want the null address", owner)
			break
		}
	}
}

// TestCallDecodesEachOutputType drives one Call per output type through the
// fake's ConstantResult and checks the decoded accessor value.
func TestCallDecodesEachOutputType(t *testing.T) {
	addrPayload := testMainnetAddr.Bytes()[1:]
	f := &fakeWallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExtention()
			switch sig := string(in.GetData()[:4]); {
			case sig == string(mustSelector("name()")):
				// string: offset word + length word + "TRON" padded
				ext.ConstantResult = [][]byte{abiUint256(0x20), abiUint256(4), append([]byte("TRON"), make([]byte, 28)...)}
			case string(in.GetData()[:4]) == string(mustSelector("getOwner()")):
				ext.ConstantResult = [][]byte{abiAddress(addrPayload)}
			case string(in.GetData()[:4]) == string(mustSelector("balanceOf(address)")):
				ext.ConstantResult = [][]byte{abiUint256(777)}
			default:
				ext.ConstantResult = [][]byte{abiBool(true)}
			}
			return ext, nil
		},
	}
	i := newInstanceWithTestABIOn(t, f)
	ctx := context.Background()

	t.Run("uint256", func(t *testing.T) {
		res, err := i.Call(ctx, "balanceOf", AddressArg(testMainnetAddr))
		if err != nil {
			t.Fatalf("Call: %v", err)
		}
		if v, err := res.BigInt(); err != nil || v.Int64() != 777 {
			t.Errorf("BigInt() = (%v, %v), want (777, nil)", v, err)
		}
	})
	t.Run("bool", func(t *testing.T) {
		// transfer returns bool; the fake's default branch answers abiBool(true).
		res, err := i.Call(ctx, "transfer", AddressArg(testMainnetAddr), BigIntArg(big.NewInt(1)))
		if err != nil {
			t.Fatalf("Call: %v", err)
		}
		if v, err := res.Bool(); err != nil || v != true {
			t.Errorf("Bool() = (%v, %v), want (true, nil)", v, err)
		}
	})
	t.Run("string", func(t *testing.T) {
		res, err := i.Call(ctx, "name")
		if err != nil {
			t.Fatalf("Call: %v", err)
		}
		if v, err := res.String(); err != nil || v != "TRON" {
			t.Errorf("String() = (%q, %v), want (TRON, nil)", v, err)
		}
	})
	t.Run("address", func(t *testing.T) {
		res, err := i.Call(ctx, "getOwner")
		if err != nil {
			t.Fatalf("Call: %v", err)
		}
		v, err := res.Address()
		if err != nil || v != testMainnetAddr {
			t.Errorf("Address() = (%s, %v), want (%s, nil) — the 0x41 round-trip through the node path", v, err, testMainnetAddr)
		}
	})
}

// TestCallTypeMismatchOnWrongAccessor: a uint256 return read as bool is
// contract.result_type_mismatch — through the full Call path.
func TestCallTypeMismatchOnWrongAccessor(t *testing.T) {
	f := &fakeWallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExtention()
			ext.ConstantResult = [][]byte{abiUint256(777)}
			return ext, nil
		},
	}
	i := newInstanceWithTestABIOn(t, f)
	res, err := i.Call(context.Background(), "balanceOf", AddressArg(testMainnetAddr))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if _, err := res.Bool(); !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
		t.Errorf("Bool() on uint256: err = %v, want contract.result_type_mismatch", err)
	}
}

// TestCallMethodUnknown: a method not in the ABI is contract.method_unknown.
func TestCallMethodUnknown(t *testing.T) {
	i := newInstanceWithTestABI(t)
	_, err := i.Call(context.Background(), "doesNotExist")
	if !tron.HasCode(err, tron.CodeContractMethodUnknown) {
		t.Errorf("Call(unknown): err = %v, want contract.method_unknown", err)
	}
}

// TestCallArgCountMismatch: the wrong number of arguments is
// contract.arg_mismatch (and no RPC fires).
func TestCallArgCountMismatch(t *testing.T) {
	f := &fakeWallet{}
	i := newInstanceWithTestABIOn(t, f)
	_, err := i.Call(context.Background(), "balanceOf") // takes 1 address
	if !tron.HasCode(err, tron.CodeContractArgMismatch) {
		t.Errorf("Call with 0 args: err = %v, want contract.arg_mismatch", err)
	}
	_, err = i.Call(context.Background(), "name", AddressArg(testMainnetAddr)) // takes 0
	if !tron.HasCode(err, tron.CodeContractArgMismatch) {
		t.Errorf("Call with 1 extra arg: err = %v, want contract.arg_mismatch", err)
	}
	if f.triggerConstantCalls.Load() != 0 {
		t.Errorf("TriggerConstantContract fired %d times on encode failures, want 0", f.triggerConstantCalls.Load())
	}
}

// TestCallArgTypeMismatch: an address passed where uint256 is declared is
// contract.arg_mismatch (the Arg seal carries the declared type).
func TestCallArgTypeMismatch(t *testing.T) {
	i := newInstanceWithTestABI(t)
	_, err := i.Call(context.Background(), "balanceOf", BigIntArg(big.NewInt(1)))
	if !tron.HasCode(err, tron.CodeContractArgMismatch) {
		t.Errorf("Call with a uint256 Arg for an address input: err = %v, want contract.arg_mismatch", err)
	}
}

// TestCallNodeReject: a node-level failure (Result.Result == false, e.g. a
// reverted constant call) surfaces as a classified tron.Error.
func TestCallNodeReject(t *testing.T) {
	f := &fakeWallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: &api.Return{Result: false, Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("revert")}}, nil
		},
	}
	i := newInstanceWithTestABIOn(t, f)
	_, err := i.Call(context.Background(), "balanceOf", AddressArg(testMainnetAddr))
	if err == nil {
		t.Fatal("Call on a rejected node result: err = nil, want a classified error")
	}
	var te *tron.Error
	if !asTronError(err, &te) {
		t.Fatalf("err = %T (%v), want *tron.Error", err, err)
	}
	if te.Code == "" {
		t.Errorf("unclassified node rejection: %v", err)
	}
}

// TestCallAtBlockRefused: the Wallet API has no block-anchored constant
// call, so CallAtBlock refuses with a classified error rather than silently
// reading head state (see its doc).
func TestCallAtBlockRefused(t *testing.T) {
	i := newInstanceWithTestABI(t)
	res, err := i.CallAtBlock(context.Background(), 42_000_000, "balanceOf", AddressArg(testMainnetAddr))
	if res != nil {
		t.Errorf("CallAtBlock returned a Result (%v); it must refuse", res)
	}
	if err == nil {
		t.Fatal("CallAtBlock: err = nil, want a refusal error")
	}
	var te *tron.Error
	if !asTronError(err, &te) || te.Code == "" {
		t.Errorf("CallAtBlock error is not a classified *tron.Error: %v", err)
	}
}

// TestInvokeReturnsContractTx: Invoke encodes and hands off to the tx
// builder, returning *tx.ContractTx — the DAG direction contract → tx —
// and the returned transaction's Kind is KindContract.
func TestInvokeReturnsContractTx(t *testing.T) {
	var gotReq *core.TriggerSmartContract
	f := &fakeWallet{
		Trigger: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			gotReq = in
			return triggerExt(), nil
		},
	}
	i := newInstanceWithTestABIOn(t, f)
	owner := mustAddr(0x11)
	value := tron.SUN(0)
	ct, err := i.Invoke(context.Background(), owner, value, "transfer", AddressArg(testMainnetAddr), BigIntArg(big.NewInt(100)))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	// The static return type IS the DAG-direction proof: Invoke's signature
	// hands back the tx layer's concrete ContractTx (a compile-time fact,
	// re-checked here for readability).
	var _ *tx.ContractTx = ct
	if ct.Kind() != tx.KindContract {
		t.Errorf("Kind() = %v, want KindContract", ct.Kind())
	}
	if gotReq == nil {
		t.Fatal("the tx builder never reached TriggerContract")
	}
	if string(gotReq.GetOwnerAddress()) != string(owner.Bytes()) {
		t.Errorf("TriggerContract owner = %x, want %x", gotReq.GetOwnerAddress(), owner.Bytes())
	}
	if string(gotReq.GetContractAddress()) != string(testContractAddress.Bytes()) {
		t.Errorf("TriggerContract contract = %x, want %x", gotReq.GetContractAddress(), testContractAddress.Bytes())
	}
	if len(gotReq.GetData()) == 0 {
		t.Error("TriggerContract data is empty; the encoded call data was not forwarded")
	}
	if int64(value) != int64(gotReq.GetCallValue()) {
		t.Errorf("TriggerContract CallValue = %d, want %d", gotReq.GetCallValue(), value)
	}
}

// TestInvokeValidation: a zero owner is address.invalid before any RPC.
func TestInvokeValidation(t *testing.T) {
	f := &fakeWallet{}
	i := newInstanceWithTestABIOn(t, f)
	_, err := i.Invoke(context.Background(), tron.Address{}, 0, "transfer", AddressArg(testMainnetAddr), BigIntArg(big.NewInt(1)))
	if !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("Invoke(zero owner): err = %v, want address.invalid", err)
	}
	if f.triggerCalls.Load() != 0 {
		t.Error("Invoke with a zero owner must not reach the network")
	}
}

// TestDecodeRoundTrip (spec §14 step 8's partner): encode via the Invoke
// path's encoder, hand the data to Decode, and get the values back.
// Decode targets OUTPUTS, so this drives Decode(method, outputBytes)
// against canned ConstantResult blobs — the same blob Call decodes.
func TestDecodeRoundTrip(t *testing.T) {
	i := newInstanceWithTestABI(t)

	t.Run("uint256", func(t *testing.T) {
		res, err := i.Decode("balanceOf", abiUint256(555))
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if v, err := res.BigInt(); err != nil || v.Int64() != 555 {
			t.Errorf("BigInt() = (%v, %v), want (555, nil)", v, err)
		}
	})
	t.Run("bool", func(t *testing.T) {
		res, err := i.Decode("noop", nil) // no outputs -> IsNil Result
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if !res.IsNil() {
			t.Error("Decode of a no-output method: IsNil() = false")
		}
	})
	t.Run("address", func(t *testing.T) {
		res, err := i.Decode("getOwner", abiAddress(testMainnetAddr.Bytes()[1:]))
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		v, err := res.Address()
		if err != nil || v != testMainnetAddr {
			t.Errorf("Address() = (%s, %v), want (%s, nil) — the decode half of the 0x41 rule via Decode", v, err, testMainnetAddr)
		}
	})
	t.Run("bad-data", func(t *testing.T) {
		_, err := i.Decode("balanceOf", []byte{0x01}) // too short for uint256
		if !tron.HasCode(err, tron.CodeContractArgMismatch) {
			t.Errorf("Decode of undecodable data: err = %v, want contract.arg_mismatch", err)
		}
	})
	t.Run("unknown-method", func(t *testing.T) {
		_, err := i.Decode("doesNotExist", nil)
		if !tron.HasCode(err, tron.CodeContractMethodUnknown) {
			t.Errorf("Decode of an unknown method: err = %v, want contract.method_unknown", err)
		}
	})
}

// TestDecodeContextPropagatesContext: Decode hardcodes a background
// context, so a lazy ABI fetch cannot be bounded. DecodeContext passes the
// caller's context through to that fetch — a cancelled context must surface
// as an error, not silently fetch on Background.
func TestDecodeContextPropagatesContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &fakeWallet{GetContractFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return &core.SmartContract{Abi: mustPbABI(testABI)}, nil
		}
	}}
	i, err := NewInstance(newContractTestClient(t, f), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	_, err = i.DecodeContext(ctx, "balanceOf", abiUint256(1))
	if err == nil {
		t.Fatal("DecodeContext with a cancelled context returned no error (context not propagated)")
	}
	if !tron.HasCode(err, tron.CodeChainTimeout) && !tron.HasCode(err, tron.CodeChainConnection) {
		t.Errorf("DecodeContext: err = %v, want chain.timeout (or chain.connection)", err)
	}
}

// TestDecodeContextHappyPath: with the ABI already loaded, DecodeContext
// decodes like Decode but under the caller's context.
func TestDecodeContextHappyPath(t *testing.T) {
	i := newInstanceWithTestABI(t)
	res, err := i.DecodeContext(t.Context(), "balanceOf", abiUint256(9))
	if err != nil {
		t.Fatalf("DecodeContext: %v", err)
	}
	if v, err := res.BigInt(); err != nil || v.Int64() != 9 {
		t.Errorf("BigInt() = (%v, %v), want (9, nil)", v, err)
	}
}

// TestLazyLoadErrorOpIsLoadABI: parseAndRegisterABI is shared by UseABI and
// the lazy network fetch, so it must label its errors with the operation
// that actually ran — otherwise a failed lazy fetch misreports itself as
// UseABI, sending callers to the wrong code path.
func TestLazyLoadErrorOpIsLoadABI(t *testing.T) {
	bad := &core.SmartContract{
		Bytecode: []byte{0x00},
		Abi: &core.SmartContract_ABI{Entrys: []*core.SmartContract_ABI_Entry{{
			Type:   core.SmartContract_ABI_Entry_Function,
			Name:   "broken",
			Inputs: []*core.SmartContract_ABI_Entry_Param{{Type: "notatype"}},
		}}},
	}
	f := &fakeWallet{GetContractFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
		return bad, nil
	}}
	i, err := NewInstance(newContractTestClient(t, f), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	_, err = i.Call(context.Background(), "balanceOf")
	if !tron.HasCode(err, tron.CodeContractBadABI) {
		t.Fatalf("Call with a bad on-chain ABI: err = %v, want contract.bad_abi", err)
	}
	var te *tron.Error
	if !asTronError(err, &te) {
		t.Fatalf("err = %v, want a *tron.Error", err)
	}
	if te.Op != "contract.loadABI" {
		t.Errorf("Op = %q, want contract.loadABI (the operation that ran)", te.Op)
	}
}

// TestEncodeDecodeInvokePath: the full call-data round-trip — encode with
// the instance's encoder (what Invoke sends), then re-parse it: the
// selector matches the ABI method and the packed arguments unpack to the
// original values.
func TestEncodeDecodeInvokePath(t *testing.T) {
	i := newInstanceWithTestABI(t)
	data, err := i.encodeCall(context.Background(), "transfer", []Arg{AddressArg(testMainnetAddr), BigIntArg(big.NewInt(42))})
	if err != nil {
		t.Fatalf("encodeCall: %v", err)
	}
	if len(data) != 4+64 {
		t.Fatalf("encoded length = %d, want 68 (selector + address + uint256)", len(data))
	}
	// The second word is the 0x41-stripped address; the third is the amount.
	want := make([]byte, 32)
	copy(want[12:], testMainnetAddr.Bytes()[1:])
	if string(data[4+12:4+32]) != string(want[12:]) {
		t.Errorf("address word = %x, want the 0x41-stripped payload", data[4:36])
	}
	amount := new(big.Int).SetBytes(data[36:])
	if amount.Int64() != 42 {
		t.Errorf("amount word = %v, want 42", amount)
	}
}

// newInstanceWithTestABIOn is newInstanceWithTestABI against a specific
// fake (so tests can install handlers before the client exists).
func newInstanceWithTestABIOn(t *testing.T, f *fakeWallet) *Instance {
	t.Helper()
	i, err := NewInstance(newContractTestClient(t, f), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	if err := i.UseABI(testABI); err != nil {
		t.Fatalf("UseABI: %v", err)
	}
	return i
}

// TestInstanceManagementOps: UpdateSetting, UpdateEnergyLimit and ClearABI
// delegate to the tx management builders bound to the Instance's address,
// returning bandwidth-only NativeTx.
func TestInstanceManagementOps(t *testing.T) {
	ctx := t.Context()
	var gotSetting *core.UpdateSettingContract
	var gotLimit *core.UpdateEnergyLimitContract
	var gotClear *core.ClearABIContract
	f := &fakeWallet{
		UpdateSettingFn: func(ctx context.Context, in *core.UpdateSettingContract) (*api.TransactionExtention, error) {
			gotSetting = in
			return manageExt(), nil
		},
		UpdateEnergyFn: func(ctx context.Context, in *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
			gotLimit = in
			return manageExt(), nil
		},
		ClearABIFn: func(ctx context.Context, in *core.ClearABIContract) (*api.TransactionExtention, error) {
			gotClear = in
			return manageExt(), nil
		},
	}
	i := newInstanceWithTestABIOn(t, f)
	owner := mustAddr(0x11)

	nt, err := i.UpdateSetting(ctx, owner, 30)
	if err != nil {
		t.Fatalf("UpdateSetting: %v", err)
	}
	var _ *tx.NativeTx = nt
	if string(gotSetting.GetContractAddress()) != string(testContractAddress.Bytes()) {
		t.Errorf("contract = %x, want the Instance address %x", gotSetting.GetContractAddress(), testContractAddress.Bytes())
	}
	if gotSetting.GetConsumeUserResourcePercent() != 30 {
		t.Errorf("percent = %d, want 30", gotSetting.GetConsumeUserResourcePercent())
	}
	if _, err := i.UpdateSetting(ctx, owner, 101); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("percent 101: want tx.invalid_argument, got %v", err)
	}

	if _, err := i.UpdateEnergyLimit(ctx, owner, 5_000_000); err != nil {
		t.Fatalf("UpdateEnergyLimit: %v", err)
	}
	if gotLimit.GetOriginEnergyLimit() != 5_000_000 {
		t.Errorf("limit = %d, want 5000000", gotLimit.GetOriginEnergyLimit())
	}

	if _, err := i.ClearABI(ctx, owner); err != nil {
		t.Fatalf("ClearABI: %v", err)
	}
	if string(gotClear.GetOwnerAddress()) != string(owner.Bytes()) {
		t.Errorf("owner = %x, want %x", gotClear.GetOwnerAddress(), owner.Bytes())
	}
}

// TestInstanceDynamicEnergy: the Instance read delegates to tx's
// GetContractInfo path — state fields map through, a nil wrapper is
// contract.not_found, and an absent state is a fresh (zero) contract.
func TestInstanceDynamicEnergy(t *testing.T) {
	ctx := t.Context()

	f := &fakeWallet{
		GetContractInfoFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
			return &core.SmartContractDataWrapper{
				SmartContract: &core.SmartContract{},
				ContractState: &core.ContractState{EnergyUsage: 6000000000, EnergyFactor: 2000, UpdateCycle: 999},
			}, nil
		},
	}
	i := newInstanceWithTestABIOn(t, f)
	got, err := i.DynamicEnergy(ctx)
	if err != nil {
		t.Fatalf("DynamicEnergy: %v", err)
	}
	if got.Factor != 2000 || got.Usage != 6000000000 || got.UpdateCycle != 999 {
		t.Fatalf("DynamicEnergy = %+v, want factor 2000 usage 6000000000 cycle 999", got)
	}
	var _ *tx.DynamicEnergy = got

	missing := &fakeWallet{
		GetContractInfoFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
			return nil, nil
		},
	}
	if _, err := newInstanceWithTestABIOn(t, missing).DynamicEnergy(ctx); !tron.HasCode(err, tron.CodeContractNotFound) {
		t.Fatalf("missing contract: want contract.not_found, got %v", err)
	}

	fresh, err := newInstanceWithTestABIOn(t, &fakeWallet{}).DynamicEnergy(ctx)
	if err != nil {
		t.Fatalf("fresh contract: %v", err)
	}
	if *fresh != (tx.DynamicEnergy{}) {
		t.Fatalf("fresh contract = %+v, want zero state", fresh)
	}
}
