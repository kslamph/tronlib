package contract

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/kslamph/tronlib/v2/tron"
)

// Result is one decoded ABI return value with typed accessors (spec §9).
// A mistaken accessor is a classified error (contract.result_type_mismatch)
// rather than a zero value or a panic, so an agent reading the wrong field
// learns the actual ABI type from the Hint instead of silently reading a
// wrong number.
type Result struct {
	// val is the decoded ABI value in its natural Go shape: bool, string,
	// *big.Int (uintN/intN of any width except uint8), tron.Address (0x41
	// re-prepended), uint8 (see Byte), uint64, []byte (dynamic bytes),
	// [N]byte (bytesN), a typed slice for collection returns ([]tron.Address,
	// [][]byte, []*big.Int, ...), or []any for multi-output methods (whose
	// singular accessors then fail — see the package doc). nil means the
	// method declared no outputs (IsNil).
	val any
}

// value is nil-safe access to the decoded value (a nil *Result decodes
// nothing — accessors on it report a nil value, they do not panic).
func (r *Result) value() any {
	if r == nil {
		return nil
	}
	return r.val
}

// resultError builds the wrong-accessor error for r. It names the actual
// decoded Go type so the fix is mechanical.
func (r *Result) resultError(accessor string) *tron.Error {
	v := r.value()
	return &tron.Error{
		Code: tron.CodeContractResultTypeMismatch,
		Op:   "contract.Result." + accessor,
		Hint: fmt.Sprintf("the ABI return value is %T, not %s; use the matching accessor", v, accessor),
	}
}

// IsNil reports whether the method declared no outputs (or the decoded
// value is the ABI's null value).
func (r *Result) IsNil() bool {
	return r == nil || r.val == nil
}

// Bool reads a bool return.
func (r *Result) Bool() (bool, error) {
	v, ok := r.value().(bool)
	if !ok {
		return false, r.resultError("Bool")
	}
	return v, nil
}

// String reads a string return.
func (r *Result) String() (string, error) {
	v, ok := r.value().(string)
	if !ok {
		return "", r.resultError("String")
	}
	return v, nil
}

// BigInt reads an integer return (uintN/intN of any width decode as
// *big.Int — the ERC-20 balance/allowance shape). The one exception is the
// declared uint8 type, which geth decodes as Go uint8 and must be read with
// Byte; this accessor reports contract.result_type_mismatch for it.
func (r *Result) BigInt() (*big.Int, error) {
	v, ok := r.value().(*big.Int)
	if !ok {
		return nil, r.resultError("BigInt")
	}
	return v, nil
}

// Address reads an address return. The 0x41 rule's decode half (spec
// §9.1, normative): the 20-byte ABI form is re-prepended with TRON's 0x41,
// so the returned Address round-trips against AddressArg.
func (r *Result) Address() (tron.Address, error) {
	v, ok := r.value().(tron.Address)
	if !ok {
		return tron.Address{}, r.resultError("Address")
	}
	return v, nil
}

// Byte reads a uint8 return (geth decodes the declared uint8 ABI type as Go
// uint8, e.g. a TRC-20 decimals() on some contracts). Wider integers decode
// as *big.Int — use BigInt for those.
func (r *Result) Byte() (uint8, error) {
	v, ok := r.value().(uint8)
	if !ok {
		return 0, r.resultError("Byte")
	}
	return v, nil
}

// Uint64 reads a uint64 return (geth decodes the declared uint64 ABI type
// as Go uint64; wider integers decode as *big.Int — use BigInt for those,
// and read a uint8 with Byte).
func (r *Result) Uint64() (uint64, error) {
	v, ok := r.value().(uint64)
	if !ok {
		return 0, r.resultError("Uint64")
	}
	return v, nil
}

// Bytes reads a bytes return. Dynamic bytes decode as []byte; fixed
// bytesN decode as [N]byte and are normalized to a []byte slice (a copy —
// mutating the result does not affect the Result).
func (r *Result) Bytes() ([]byte, error) {
	switch v := r.value().(type) {
	case []byte:
		return v, nil
	case string: // unreachable today; guards a geth representation change
		return []byte(v), nil
	}
	rv := reflect.ValueOf(r.value())
	if rv.Kind() == reflect.Array && rv.Type().Elem().Kind() == reflect.Uint8 {
		out := make([]byte, rv.Len())
		reflect.Copy(reflect.ValueOf(out), rv)
		return out, nil
	}
	return nil, r.resultError("Bytes")
}

// Addresses reads an address[] or address[N] return. Each element is
// re-prepended with 0x41 exactly like Address, so the returned addresses
// round-trip against AddressArg. A scalar address return is a
// contract.result_type_mismatch here (use Address).
func (r *Result) Addresses() ([]tron.Address, error) {
	v, ok := r.value().([]tron.Address)
	if !ok {
		return nil, r.resultError("Addresses")
	}
	return v, nil
}

// BytesSlice reads a bytesN[] or bytes[] return as one []byte per element,
// with each bytesN element normalized to a fresh []byte copy (see Bytes).
// A scalar bytes return is a contract.result_type_mismatch here (use Bytes).
func (r *Result) BytesSlice() ([][]byte, error) {
	v, ok := r.value().([][]byte)
	if !ok {
		return nil, r.resultError("BytesSlice")
	}
	return v, nil
}

// tronAddressFromEVM converts a 20-byte EVM address into a tron.Address by
// prepending TRON's 0x41 network prefix — the decode half of the 0x41 rule.
func tronAddressFromEVM(b [20]byte) (tron.Address, error) {
	full := make([]byte, 21)
	full[0] = 0x41
	copy(full[1:], b[:])
	return tron.AddressFromBytes(full)
}
