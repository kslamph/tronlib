package contract

import (
	"fmt"
	"math/big"

	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/v2/tron"
)

// Arg is one ABI-encoded call argument. It is SEALED (spec §9): the only
// method is unexported, so types outside this package cannot implement it
// and callers cannot smuggle arbitrary values into call data — every
// argument enters through a named constructor whose encoding is reviewed.
type Arg interface {
	argABI() string
}

// The five implementations of Arg. The string each returns is the
// canonical Solidity ABI type the constructor encodes; it is the encode
// path's input (packWithSelector), not a display form.
type (
	boolArg    bool
	stringArg  string
	bigIntArg  struct{ v *big.Int }
	addressArg struct{ v tron.Address }
	uint64Arg  uint64
)

func (b boolArg) argABI() string    { return "bool" }
func (s stringArg) argABI() string  { return "string" }
func (b bigIntArg) argABI() string  { return "uint256" }
func (a addressArg) argABI() string { return "address" }
func (u uint64Arg) argABI() string  { return "uint64" }

// BoolArg encodes a bool.
func BoolArg(v bool) Arg { return boolArg(v) }

// StringArg encodes a string.
func StringArg(v string) Arg { return stringArg(v) }

// BigIntArg encodes a *big.Int as uint256 — the type of every integer
// return and argument of ERC-20/TRC-20 ABIs. A nil v yields an Arg whose
// encoding FAILS at Call/Invoke/encode time with contract.arg_mismatch
// (a constructor cannot return an error without changing the spec's
// signature; failing late keeps the failure classified and explicit).
func BigIntArg(v *big.Int) Arg { return bigIntArg{v: v} }

// AddressArg encodes a tron.Address. Per the normative 0x41 rule
// (spec §9.1) the 21-byte TRON form is stripped to the 20-byte ABI form
// before packing — geth's address packer requires exactly 20 bytes, so the
// strip happens here, once, at the only place arguments become bytes.
func AddressArg(v tron.Address) Arg { return addressArg{v: v} }

// Uint64Arg encodes a uint64 (e.g. TRC-20 amount parameters on contracts
// with fewer than 20 decimals of headroom, deadline/timestamp parameters).
func Uint64Arg(v uint64) Arg { return uint64Arg(v) }

// toEVMValue converts a sealed Arg into the Go value geth's packer expects
// for the method's declared ABI type (the declared type itself is checked
// by encodeArgs). This is the 0x41 rule's encode half.
func toEVMValue(a Arg) (any, error) {
	switch v := a.(type) {
	case boolArg:
		return bool(v), nil
	case stringArg:
		return string(v), nil
	case bigIntArg:
		if v.v == nil {
			return nil, fmt.Errorf("BigIntArg(nil): a nil integer has no ABI encoding")
		}
		return v.v, nil
	case addressArg:
		b := v.v.Bytes() // 21 bytes, 0x41-prefixed
		if len(b) != 21 || b[0] != 0x41 {
			return nil, fmt.Errorf("address is not 0x41-prefixed 21-byte form")
		}
		return eCommon.BytesToAddress(b[1:]), nil
	case uint64Arg:
		return uint64(v), nil
	default:
		return nil, fmt.Errorf("unhandled Arg implementation %T (a sealed-interface violation)", a)
	}
}
