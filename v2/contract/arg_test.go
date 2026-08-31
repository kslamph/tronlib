package contract

import (
	"math/big"
	"strings"
	"testing"

	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/v2/tron"
	"golang.org/x/crypto/sha3"
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
// (spec §9.1): AddressArg must produce a 20-byte EVM address, NOT the
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
