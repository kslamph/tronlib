package contract

import (
	"math/big"
	"testing"

	"github.com/kslamph/tronlib/v2/tron"
)

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
