package contract

import (
	"math/big"
	"testing"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/v2/tron"
)

// collectionABI declares the collection-returning methods the R1 tests
// exercise: dynamic and fixed address arrays, a fixed bytesN array, and an
// empty address[] (the zero-length edge).
const collectionABI = `[
  {"type":"function","name":"owners","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address[]"}]},
  {"type":"function","name":"ownerPair","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address[2]"}]},
  {"type":"function","name":"hashes","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"bytes32[2]"}]}
]`

// newInstanceWithABI loads abiJSON up front (no network) so Decode tests
// exercise the real unpack/convert path.
func newInstanceWithABI(t *testing.T, abiJSON string) *Instance {
	t.Helper()
	i, err := NewInstance(newContractTestClient(t, &fakeWallet{}), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	if err := i.UseABI(abiJSON); err != nil {
		t.Fatalf("UseABI: %v", err)
	}
	return i
}

// packOutput ABI-packs one value of the declared Solidity type into the raw
// return bytes the node would hand back.
func packOutput(t *testing.T, abiType string, v any) []byte {
	t.Helper()
	at, err := eABI.NewType(abiType, "", nil)
	if err != nil {
		t.Fatalf("NewType(%s): %v", abiType, err)
	}
	b, err := eABI.Arguments{{Type: at}}.Pack(v)
	if err != nil {
		t.Fatalf("Pack(%s): %v", abiType, err)
	}
	return b
}

// evmAddr strips the 0x41 prefix from a tron address, producing the 20-byte
// form geth's address packer expects.
func evmAddr(a tron.Address) eCommon.Address {
	return eCommon.BytesToAddress(a.Bytes()[1:])
}

// TestResultCollectionAccessors: collection returns are converted element-
// wise — every address re-prepended with 0x41, every bytesN normalized to
// []byte — and read back through the slice accessors. geth hands these back
// as typed slices ([]common.Address, [2]common.Address, [2][32]byte), not the
// []any the scalar-only converter used to expect.
func TestResultCollectionAccessors(t *testing.T) {
	i := newInstanceWithABI(t, collectionABI)
	a1, a2 := mustAddr(0x11), mustAddr(0x22)

	t.Run("address[] re-prepends 0x41", func(t *testing.T) {
		res, err := i.Decode("owners", packOutput(t, "address[]", []eCommon.Address{evmAddr(a1), evmAddr(a2)}))
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		got, err := res.Addresses()
		if err != nil {
			t.Fatalf("Addresses(): %v", err)
		}
		if len(got) != 2 || got[0] != a1 || got[1] != a2 {
			t.Errorf("Addresses() = %v, want [%s %s]", got, a1, a2)
		}
	})

	t.Run("address[2] re-prepends 0x41", func(t *testing.T) {
		res, err := i.Decode("ownerPair", packOutput(t, "address[2]", [2]eCommon.Address{evmAddr(a1), evmAddr(a2)}))
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		got, err := res.Addresses()
		if err != nil {
			t.Fatalf("Addresses(): %v", err)
		}
		if len(got) != 2 || got[0] != a1 || got[1] != a2 {
			t.Errorf("Addresses() = %v, want [%s %s]", got, a1, a2)
		}
	})

	t.Run("bytes32[2] normalizes to [][]byte", func(t *testing.T) {
		var h1, h2 [32]byte
		h1[0], h2[31] = 0xAB, 0xCD
		res, err := i.Decode("hashes", packOutput(t, "bytes32[2]", [2][32]byte{h1, h2}))
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		got, err := res.BytesSlice()
		if err != nil {
			t.Fatalf("BytesSlice(): %v", err)
		}
		if len(got) != 2 || len(got[0]) != 32 || got[0][0] != 0xAB || got[1][31] != 0xCD {
			t.Errorf("BytesSlice() = %x, want two 32-byte values", got)
		}
	})

	t.Run("empty address[] yields an empty typed slice", func(t *testing.T) {
		res, err := i.Decode("owners", packOutput(t, "address[]", []eCommon.Address{}))
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		got, err := res.Addresses()
		if err != nil {
			t.Fatalf("Addresses() on empty: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("Addresses() = %v, want empty", got)
		}
	})
}

// TestResultAccessorsHappyPath: the right accessor returns the decoded
// value; wrong accessors return contract.result_type_mismatch.
func TestResultAccessorsHappyPath(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		r := &Result{val: true}
		v, err := r.Bool()
		if err != nil || v != true {
			t.Errorf("Bool() = (%v, %v), want (true, nil)", v, err)
		}
	})
	t.Run("string", func(t *testing.T) {
		r := &Result{val: "TRON"}
		v, err := r.String()
		if err != nil || v != "TRON" {
			t.Errorf("String() = (%v, %v), want (TRON, nil)", v, err)
		}
	})
	t.Run("bigint", func(t *testing.T) {
		r := &Result{val: big.NewInt(12345)}
		v, err := r.BigInt()
		if err != nil || v.Int64() != 12345 {
			t.Errorf("BigInt() = (%v, %v), want (12345, nil)", v, err)
		}
	})
	t.Run("uint64", func(t *testing.T) {
		r := &Result{val: uint64(77)}
		v, err := r.Uint64()
		if err != nil || v != 77 {
			t.Errorf("Uint64() = (%v, %v), want (77, nil)", v, err)
		}
	})
	t.Run("bytes-from-bytesN", func(t *testing.T) {
		var fixed [32]byte
		fixed[0] = 0xAB
		r := &Result{val: fixed}
		v, err := r.Bytes()
		if err != nil || len(v) != 32 || v[0] != 0xAB {
			t.Errorf("Bytes() = (%x, %v), want a 32-byte copy of the bytesN", v, err)
		}
		// A copy: mutating the result must not affect the Result's value.
		v[0] = 0x00
		if r.val.([32]byte)[0] != 0xAB {
			t.Error("Bytes() returned a view, not a copy — mutation leaked into the Result")
		}
	})
	t.Run("address-round-trip", func(t *testing.T) {
		// The decode half of the 0x41 rule: the decoded 20-byte EVM form is
		// re-prepended with 0x41 so it equals the original tron.Address.
		r := &Result{val: testMainnetAddr}
		v, err := r.Address()
		if err != nil || v != testMainnetAddr {
			t.Errorf("Address() = (%v, %v), want (%s, nil)", v, err, testMainnetAddr)
		}
	})
}

// TestResultWrongAccessorIsTypeMismatch: every accessor on a mismatched
// Result returns contract.result_type_mismatch with a Hint naming the
// actual decoded type.
func TestResultWrongAccessorIsTypeMismatch(t *testing.T) {
	values := map[string]*Result{
		"bool":    {val: true},
		"string":  {val: "hi"},
		"bigint":  {val: big.NewInt(1)},
		"uint64":  {val: uint64(2)},
		"bytes":   {val: []byte{1}},
		"address": {val: testMainnetAddr},
		"multi":   {val: []any{true, "hi"}},
	}
	for _, acc := range []string{"Bool", "String", "BigInt", "Uint64", "Bytes", "Address"} {
		for name, r := range values {
			// Skip the accessor that matches the value's real type.
			if (acc == "Bool" && name == "bool") ||
				(acc == "String" && name == "string") ||
				(acc == "BigInt" && name == "bigint") ||
				(acc == "Uint64" && name == "uint64") ||
				(acc == "Bytes" && name == "bytes") ||
				// Bytes() on a string is deliberately tolerated by the
				// implementation (guards a geth representation change; see
				// result.go) — so it is not a mismatch case.
				(acc == "Bytes" && name == "string") ||
				(acc == "Address" && name == "address") {
				continue
			}
			var err error
			switch acc {
			case "Bool":
				_, err = r.Bool()
			case "String":
				_, err = r.String()
			case "BigInt":
				_, err = r.BigInt()
			case "Uint64":
				_, err = r.Uint64()
			case "Bytes":
				_, err = r.Bytes()
			case "Address":
				_, err = r.Address()
			}
			if !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
				t.Errorf("%s().%s: err = %v, want contract.result_type_mismatch", name, acc, err)
			}
		}
	}
}

// uint8ABI declares a decimals()-style uint8 return — geth decodes the
// declared uint8 ABI type as Go uint8, which neither BigInt nor Uint64 can
// read.
const uint8ABI = `[
  {"type":"function","name":"decimals","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint8"}]}
]`

// TestResultByteAccessor: a uint8 return is read through Byte(); the wider
// integer accessors report a type mismatch for it rather than silently
// widening.
func TestResultByteAccessor(t *testing.T) {
	i := newInstanceWithABI(t, uint8ABI)
	res, err := i.Decode("decimals", packOutput(t, "uint8", uint8(6)))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	v, err := res.Byte()
	if err != nil || v != 6 {
		t.Errorf("Byte() = (%d, %v), want (6, nil)", v, err)
	}
	if _, err := res.BigInt(); !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
		t.Errorf("BigInt() on a uint8 return: err = %v, want contract.result_type_mismatch", err)
	}
	if _, err := res.Uint64(); !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
		t.Errorf("Uint64() on a uint8 return: err = %v, want contract.result_type_mismatch", err)
	}
}

// TestResultByteWrongAccessor: Byte() on a non-uint8 value is a classified
// mismatch, not a zero value.
func TestResultByteWrongAccessor(t *testing.T) {
	r := &Result{val: big.NewInt(1)}
	if _, err := r.Byte(); !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
		t.Errorf("Byte() on *big.Int: err = %v, want contract.result_type_mismatch", err)
	}
}

// TestCollectionAccessorWrongType: the slice accessors and their scalar
// counterparts reject each other's shapes with a classified mismatch.
func TestCollectionAccessorWrongType(t *testing.T) {
	addrs := &Result{val: []tron.Address{testMainnetAddr}}
	if _, err := addrs.BytesSlice(); !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
		t.Errorf("BytesSlice() on []tron.Address: err = %v, want contract.result_type_mismatch", err)
	}
	if _, err := addrs.Address(); !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
		t.Errorf("Address() on an address[] return: err = %v, want contract.result_type_mismatch", err)
	}
	b := &Result{val: [][]byte{{1}}}
	if _, err := b.Addresses(); !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
		t.Errorf("Addresses() on [][]byte: err = %v, want contract.result_type_mismatch", err)
	}
}

// TestResultIsNil: a no-output method decodes to a nil-valued Result
// (IsNil); accessors on it are a type mismatch, not a panic.
func TestResultIsNil(t *testing.T) {
	r := &Result{}
	if !r.IsNil() {
		t.Error("IsNil() = false on an empty Result")
	}
	if _, err := r.BigInt(); !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
		t.Errorf("BigInt() on empty Result: err = %v, want contract.result_type_mismatch", err)
	}
	// A nil *Result is also safe.
	var nilResult *Result
	if !nilResult.IsNil() {
		t.Error("IsNil() = false on nil *Result")
	}
	if _, err := nilResult.Bool(); !tron.HasCode(err, tron.CodeContractResultTypeMismatch) {
		t.Errorf("Bool() on nil *Result: err = %v, want contract.result_type_mismatch", err)
	}
}
