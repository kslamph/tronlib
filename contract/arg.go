package contract

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"

	eCommon "github.com/ethereum/go-ethereum/common"

	"github.com/kslamph/tronlib/v2/tron"
)

// Arg is one ABI-encoded call argument. It is SEALED: the only
// method is unexported, so types outside this package cannot implement it
// and callers cannot smuggle arbitrary values into call data — every
// argument enters through a named constructor whose encoding is reviewed.
type Arg interface {
	argABI() string
}

// The implementations of Arg. The string each returns is the
// canonical Solidity ABI type the constructor encodes; it is the encode
// path's input (packWithSelector), not a display form.
type (
	boolArg         bool
	stringArg       string
	bigIntArg       struct{ v *big.Int }
	addressArg      struct{ v tron.Address }
	uint64Arg       uint64
	addressSliceArg []tron.Address
	bytesArg        []byte
	bytesNArg       []byte
	int256Arg       struct{ v *big.Int }
)

func (b boolArg) argABI() string         { return "bool" }
func (s stringArg) argABI() string       { return "string" }
func (b bigIntArg) argABI() string       { return "uint256" }
func (a addressArg) argABI() string      { return "address" }
func (u uint64Arg) argABI() string       { return "uint64" }
func (a addressSliceArg) argABI() string { return "address[]" }
func (b bytesArg) argABI() string        { return "bytes" }
func (b bytesNArg) argABI() string       { return "bytes" + strconv.Itoa(len(b)) }
func (i int256Arg) argABI() string       { return "int256" }

// BoolArg encodes a bool.
func BoolArg(v bool) Arg { return boolArg(v) }

// StringArg encodes a string.
func StringArg(v string) Arg { return stringArg(v) }

// BigIntArg encodes a *big.Int as uint256 — the type of every integer
// return and argument of ERC-20/TRC-20 ABIs. A nil v yields an Arg whose
// encoding FAILS at Call/Invoke/encode time with contract.arg_mismatch
// (a constructor cannot return an error without changing the constructor's
// signature; failing late keeps the failure classified and explicit).
func BigIntArg(v *big.Int) Arg { return bigIntArg{v: v} }

// AddressArg encodes a tron.Address. Per the 0x41 rule,
// the 21-byte TRON form is stripped to the 20-byte ABI form
// before packing — geth's address packer requires exactly 20 bytes, so the
// strip happens here, once, at the only place arguments become bytes.
func AddressArg(v tron.Address) Arg { return addressArg{v: v} }

// Uint64Arg encodes a uint64 (e.g. TRC-20 amount parameters on contracts
// with fewer than 20 decimals of headroom, deadline/timestamp parameters).
func Uint64Arg(v uint64) Arg { return uint64Arg(v) }

// AddressSliceArg encodes a []tron.Address as Solidity address[] — e.g. a
// batch transfer's recipients. Each element is stripped to its 20-byte ABI
// form (the 0x41 rule) and every element must be a valid 0x41-prefixed
// address; a malformed element fails at encode time with
// contract.arg_mismatch.
func AddressSliceArg(v []tron.Address) Arg { return addressSliceArg(v) }

// BytesArg encodes a []byte as Solidity bytes (dynamic length).
func BytesArg(v []byte) Arg { return bytesArg(v) }

// BytesNArg encodes a []byte as Solidity bytesN, where N is len(v) — so
// BytesNArg(make([]byte, 32)) is bytes32. geth requires a real [N]byte for
// a fixed-bytes argument (it rejects a slice), which this constructor
// builds. len(v) must be 1..32 (bytes1..bytes32); 0 or >32 fails at encode
// time with contract.arg_mismatch.
func BytesNArg(v []byte) Arg { return bytesNArg(v) }

// Int256Arg encodes a *big.Int as Solidity int256 (the SIGNED counterpart
// of BigIntArg, whose uint256 cannot express negatives). A nil v fails at
// encode time with contract.arg_mismatch, like BigIntArg(nil).
func Int256Arg(v *big.Int) Arg { return int256Arg{v: v} }

// ABI integer bounds. geth's packer does NOT enforce them — it only rejects
// negatives for uint and silently wraps every other out-of-range value into
// 32 bytes — so the constructors refuse here instead: uint256 holds
// [0, 2^256); int256 holds [-2^255, 2^255).
var (
	abiUint256Max = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	abiInt256Min  = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255))
	abiInt256Max  = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
)

// evmAddress validates that a is TRON's 21-byte 0x41-prefixed form and
// strips the prefix to the 20-byte address geth's ABI packer requires — the
// encode half of the 0x41 rule, shared by the scalar and slice constructors
// so the rule lives in exactly one place.
func evmAddress(a tron.Address) (eCommon.Address, error) {
	b := a.Bytes()
	if len(b) != 21 || b[0] != 0x41 {
		return eCommon.Address{}, fmt.Errorf("address is not 0x41-prefixed 21-byte form")
	}
	return eCommon.BytesToAddress(b[1:]), nil
}

// toEVMValue converts a sealed Arg into the Go value geth's packer expects
// for the method's declared ABI type (the declared type itself is checked
// by encodeArgs). This is the 0x41 rule's encode half.
func toEVMValue(a Arg) (any, error) { //nolint:gocyclo // one flat type switch over the ABI value space; splitting it would scatter the conversion table
	switch v := a.(type) {
	case boolArg:
		return bool(v), nil
	case stringArg:
		return string(v), nil
	case bigIntArg:
		if v.v == nil {
			return nil, fmt.Errorf("BigIntArg(nil): a nil integer has no ABI encoding")
		}
		if v.v.Sign() < 0 {
			return nil, fmt.Errorf("BigIntArg: negative value %s has no uint256 encoding (use Int256Arg for signed values)", v.v)
		}
		if v.v.Cmp(abiUint256Max) > 0 {
			return nil, fmt.Errorf("BigIntArg: value needs %d bits; uint256 holds at most 256", v.v.BitLen())
		}
		return v.v, nil
	case addressArg:
		return evmAddress(v.v)
	case uint64Arg:
		return uint64(v), nil
	case addressSliceArg:
		out := make([]eCommon.Address, len(v))
		for idx, a := range v {
			addr, err := evmAddress(a)
			if err != nil {
				return nil, fmt.Errorf("address[%d]: %w", idx, err)
			}
			out[idx] = addr
		}
		return out, nil
	case bytesArg:
		return []byte(v), nil
	case bytesNArg:
		if len(v) < 1 || len(v) > 32 {
			return nil, fmt.Errorf("bytesN length %d out of range (bytes1..bytes32)", len(v))
		}
		arr := reflect.New(reflect.ArrayOf(len(v), reflect.TypeOf(byte(0)))).Elem()
		reflect.Copy(arr, reflect.ValueOf([]byte(v)))
		return arr.Interface(), nil
	case int256Arg:
		if v.v == nil {
			return nil, fmt.Errorf("Int256Arg(nil): a nil integer has no ABI encoding")
		}
		if v.v.Cmp(abiInt256Min) < 0 || v.v.Cmp(abiInt256Max) > 0 {
			return nil, fmt.Errorf("Int256Arg: value is outside int256's range [-2^255, 2^255-1]")
		}
		return v.v, nil
	default:
		return nil, fmt.Errorf("unhandled Arg implementation %T (a sealed-interface violation)", a)
	}
}
