package contract

import (
	"math/big"
	"strings"
	"testing"

	eCommon "github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"

	"github.com/kslamph/tronlib/v2/tron"
)

// Test fixtures: real mainnet-form addresses (the same pair tron's
// address_test pins), plus a synthetic address for ownership tests.
var (
	testMainnetAddr  = mustParseAddr("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb") // e28b3cfd4e0e909077821478e9fcb86b84be786e
	testMainnetAddr2 = mustParseAddr("TV6MuMXfmLbBqPZvBHdwFsDnQeVfnmiuSi")
)

func mustParseAddr(s string) tron.Address {
	a, err := tron.ParseAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}

// argABI is the canonical Solidity type each constructor encodes — this is
// the contract between the constructor and the encode path (encodeCall
// compares it against the method's declared input types).
func TestArgABITypeStrings(t *testing.T) {
	cases := []struct {
		name string
		arg  Arg
		want string
	}{
		{"bool", BoolArg(true), "bool"},
		{"string", StringArg("hi"), "string"},
		{"bigint", BigIntArg(big.NewInt(1)), "uint256"},
		{"address", AddressArg(testMainnetAddr), "address"},
		{"uint64", Uint64Arg(7), "uint64"},
	}
	for _, tc := range cases {
		if got := tc.arg.argABI(); got != tc.want {
			t.Errorf("%s: argABI() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestAddressArgStrips041 is the encode half of the normative 0x41 rule
// (architecture §9.1): AddressArg must produce a 20-byte EVM address, NOT the
// 21-byte TRON form (sending 0x41-prefixed + padding is the classic TRON
// ABI bug — it encodes cleanly and reads the wrong slot).
func TestAddressArgStrips041(t *testing.T) {
	for name, a := range map[string]tron.Address{
		"mainnet":  testMainnetAddr,
		"mainnet2": testMainnetAddr2,
	} {
		v, err := toEVMValue(AddressArg(a))
		if err != nil {
			t.Fatalf("%s: toEVMValue: %v", name, err)
		}
		evm, ok := v.(eCommon.Address)
		if !ok {
			t.Fatalf("%s: toEVMValue = %T, want eCommon.Address", name, v)
		}
		if len(evm.Bytes()) != 20 {
			t.Errorf("%s: encoded length = %d bytes, want 20", name, len(evm.Bytes()))
		}
		if evm.Bytes()[0] == 0x41 {
			t.Errorf("%s: encoded address still carries the 0x41 prefix", name)
		}
		// The payload must be exactly the address's 21-byte form minus 0x41.
		payload := a.Bytes()[1:]
		if string(evm.Bytes()) != string(payload) {
			t.Errorf("%s: payload mismatch: got %x, want %x", name, evm.Bytes(), payload)
		}
	}
}

// TestAddressArgRejectsNon041: an address not in 21-byte 0x41 form cannot
// be encoded (the strip has nothing valid to strip).
func TestAddressArgRejectsNon041(t *testing.T) {
	a := testMainnetAddr.Bytes()[1:] // 20 bytes, no 0x41
	bad, err := tron.AddressFromBytes(a)
	if err == nil {
		t.Fatal("AddressFromBytes accepted a 20-byte address")
	}
	_ = bad
}

// TestBigIntArgNilFailsAtEncode: BigIntArg(nil) constructs (the constructor
// signature cannot error) but fails at encode time with arg_mismatch —
// see the constructor's doc.
func TestBigIntArgNilFailsAtEncode(t *testing.T) {
	_, err := toEVMValue(BigIntArg(nil))
	if err == nil {
		t.Fatal("nil big.Int encoded without error")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("err = %v, want a nil-integer message", err)
	}
}

// TestToEVMValueShapes pins the Go shapes the geth packer receives for each
// constructor.
func TestToEVMValueShapes(t *testing.T) {
	b, err := toEVMValue(BoolArg(true))
	if err != nil || b != true {
		t.Errorf("BoolArg: (%v, %v), want (true, nil)", b, err)
	}
	s, err := toEVMValue(StringArg("x"))
	if err != nil || s != "x" {
		t.Errorf("StringArg: (%v, %v), want (x, nil)", s, err)
	}
	u, err := toEVMValue(Uint64Arg(9))
	if err != nil || u != uint64(9) {
		t.Errorf("Uint64Arg: (%v, %v), want (9, nil)", u, err)
	}
	i, err := toEVMValue(BigIntArg(big.NewInt(42)))
	if err != nil {
		t.Fatalf("BigIntArg: %v", err)
	}
	if bi, ok := i.(*big.Int); !ok || bi.Int64() != 42 {
		t.Errorf("BigIntArg: (%T %v), want *big.Int 42", i, i)
	}
}

// solTypesABI declares the methods the extended Arg constructors encode
// into: dynamic address array, dynamic bytes, fixed bytes32 and signed
// int256.
const solTypesABI = `[
  {"type":"function","name":"setOwners","stateMutability":"nonpayable","inputs":[{"name":"owners","type":"address[]"}],"outputs":[]},
  {"type":"function","name":"setData","stateMutability":"nonpayable","inputs":[{"name":"data","type":"bytes"}],"outputs":[]},
  {"type":"function","name":"setHash","stateMutability":"nonpayable","inputs":[{"name":"hash","type":"bytes32"}],"outputs":[]},
  {"type":"function","name":"setDelta","stateMutability":"nonpayable","inputs":[{"name":"delta","type":"int256"}],"outputs":[]},
  {"type":"function","name":"setAmount","stateMutability":"nonpayable","inputs":[{"name":"amount","type":"uint256"}],"outputs":[]}
]`

// TestExtendedArgABITypes: amount the new constructors report the canonical
// Solidity types the encode path compares against.
func TestExtendedArgABITypes(t *testing.T) {
	cases := []struct {
		name string
		arg  Arg
		want string
	}{
		{"address[]", AddressSliceArg([]tron.Address{testMainnetAddr}), "address[]"},
		{"bytes", BytesArg([]byte{1, 2}), "bytes"},
		{"bytes32", BytesNArg(make([]byte, 32)), "bytes32"},
		{"int256", Int256Arg(big.NewInt(-5)), "int256"},
	}
	for _, tc := range cases {
		if got := tc.arg.argABI(); got != tc.want {
			t.Errorf("%s: argABI() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestExtendedArgToEVMShapes pins the geth-facing Go shape of each new
// constructor: address[] strips 0x41 from every element, bytesN becomes a
// real [N]byte array (geth rejects a []byte for a fixed-bytes argument), and
// int256 stays a *big.Int.
func TestExtendedArgToEVMShapes(t *testing.T) {
	v, err := toEVMValue(AddressSliceArg([]tron.Address{testMainnetAddr, testMainnetAddr2}))
	if err != nil {
		t.Fatalf("AddressSliceArg: %v", err)
	}
	addrs, ok := v.([]eCommon.Address)
	if !ok || len(addrs) != 2 {
		t.Fatalf("AddressSliceArg: %T %v, want 2 eCommon.Address", v, v)
	}
	for idx, a := range []tron.Address{testMainnetAddr, testMainnetAddr2} {
		if string(addrs[idx].Bytes()) != string(a.Bytes()[1:]) {
			t.Errorf("AddressSliceArg[%d] = %x, want the 0x41-stripped form %x", idx, addrs[idx].Bytes(), a.Bytes()[1:])
		}
	}

	v, err = toEVMValue(BytesNArg([]byte{0xAA, 0xBB}))
	if err != nil {
		t.Fatalf("BytesNArg: %v", err)
	}
	arr, ok := v.([2]byte)
	if !ok || arr[0] != 0xAA || arr[1] != 0xBB {
		t.Errorf("BytesNArg: %T %v, want [2]byte {AA BB}", v, v)
	}

	v, err = toEVMValue(Int256Arg(big.NewInt(-7)))
	if err != nil {
		t.Fatalf("Int256Arg: %v", err)
	}
	if bi, ok := v.(*big.Int); !ok || bi.Int64() != -7 {
		t.Errorf("Int256Arg: %T %v, want *big.Int -7", v, v)
	}
}

// TestExtendedArgsEncodeEndToEnd drives the new constructors through the
// real encodeCall path against a matching ABI; the address[] payload must
// carry the 0x41-stripped words.
func TestExtendedArgsEncodeEndToEnd(t *testing.T) {
	i := newInstanceWithABI(t, solTypesABI)
	ctx := t.Context()

	data, err := i.encodeCall(ctx, "setOwners", []Arg{AddressSliceArg([]tron.Address{testMainnetAddr})})
	if err != nil {
		t.Fatalf("setOwners: %v", err)
	}
	// selector + offset word + length word + one padded address word.
	if len(data) != 4+32+32+32 {
		t.Fatalf("setOwners encoded length = %d, want 100", len(data))
	}
	want := make([]byte, 32)
	copy(want[12:], testMainnetAddr.Bytes()[1:])
	if string(data[len(data)-32:]) != string(want) {
		t.Errorf("setOwners address word = %x, want 0x41-stripped %x", data[len(data)-32:], want)
	}

	for _, tc := range []struct {
		method string
		arg    Arg
	}{
		{"setData", BytesArg([]byte{1, 2, 3})},
		{"setHash", BytesNArg(make([]byte, 32))},
		{"setDelta", Int256Arg(big.NewInt(-1))},
	} {
		if _, err := i.encodeCall(ctx, tc.method, []Arg{tc.arg}); err != nil {
			t.Errorf("encodeCall(%s): %v", tc.method, err)
		}
	}
}

// TestIntegerArgRangeGuard: geth's packer silently wraps out-of-range
// integers to 32 bytes (it only rejects negatives for uint), so the
// constructors must refuse them. uint256 holds [0, 2^256); int256 holds
// [-2^255, 2^255). At the encode path the refusal is contract.arg_mismatch.
func TestIntegerArgRangeGuard(t *testing.T) {
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	uintMax := new(big.Int).Sub(two256, big.NewInt(1))
	intMin := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255))
	intMax := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))

	if _, err := toEVMValue(BigIntArg(uintMax)); err != nil {
		t.Errorf("BigIntArg(2^256-1): %v, want nil (in range)", err)
	}
	for name, v := range map[string]*big.Int{
		"2^256":   two256,
		"2^256+7": new(big.Int).Add(two256, big.NewInt(7)),
	} {
		if _, err := toEVMValue(BigIntArg(v)); err == nil {
			t.Errorf("BigIntArg(%s) encoded without error; want an out-of-range refusal", name)
		}
	}
	if _, err := toEVMValue(Int256Arg(intMin)); err != nil {
		t.Errorf("Int256Arg(-2^255): %v, want nil (in range)", err)
	}
	if _, err := toEVMValue(Int256Arg(intMax)); err != nil {
		t.Errorf("Int256Arg(2^255-1): %v, want nil (in range)", err)
	}
	for name, v := range map[string]*big.Int{
		"-2^255-1": new(big.Int).Sub(intMin, big.NewInt(1)),
		"2^255":    new(big.Int).Lsh(big.NewInt(1), 255),
	} {
		if _, err := toEVMValue(Int256Arg(v)); err == nil {
			t.Errorf("Int256Arg(%s) encoded without error; want an out-of-range refusal", name)
		}
	}

	// The refusal is classified at the encode path, not a silent wrap.
	i := newInstanceWithABI(t, solTypesABI)
	if _, err := i.encodeCall(t.Context(), "setAmount", []Arg{BigIntArg(two256)}); !tron.HasCode(err, tron.CodeContractArgMismatch) {
		t.Errorf("encodeCall(uint256 2^256): err = %v, want contract.arg_mismatch", err)
	}
}

// TestEncodeCallSelector pins the full call-data shape against an
// independently computed selector: keccak256("balanceOf(address)")[:4]
// followed by the 0x41-stripped, left-padded address. This is the end-to-end
// proof of the encode half of the 0x41 rule through the real encode path.
func TestEncodeCallSelectorAndAddress(t *testing.T) {
	i := newInstanceWithTestABI(t)
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte("balanceOf(address)"))
	sel := hash.Sum(nil)[:4]

	data, err := i.encodeCall(t.Context(), "balanceOf", []Arg{AddressArg(testMainnetAddr)})
	if err != nil {
		t.Fatalf("encodeCall: %v", err)
	}
	if len(data) != 4+32 {
		t.Fatalf("encoded length = %d, want 36", len(data))
	}
	if string(data[:4]) != string(sel[:]) {
		t.Errorf("selector = %x, want %x", data[:4], sel)
	}
	arg := data[4:]
	want := make([]byte, 32)
	copy(want[12:], testMainnetAddr.Bytes()[1:]) // 0x41 stripped, right-aligned
	if string(arg) != string(want) {
		t.Errorf("address argument = %x, want 0x41-stripped padded %x", arg, want)
	}
}
