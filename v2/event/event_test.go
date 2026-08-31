package event

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/tronlib/v2/tron"
)

// keccakTopic hashes a canonical event signature string into a 32-byte topic.
func keccakTopic(t *testing.T, sig string) []byte {
	t.Helper()
	h := crypto.Keccak256([]byte(sig))
	return h
}

// padAddr builds a 32-byte topic from a 20-byte address payload (left-padded).
func padAddr(hex20 string) []byte {
	b, _ := hex.DecodeString(hex20)
	out := make([]byte, 32)
	copy(out[12:], b)
	return out
}

// fixture ported from v1 pkg/eventdecoder/decoder_test.go
// (TestRegisterAndDecodeTRC20): synthetic TRC-20 Transfer log, value = 1000.
var transferFixture = struct {
	sig   string
	from  []byte
	to    []byte
	value []byte
}{
	sig:   "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
	from:  padAddr("a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
	to:    padAddr("4e83362442b8d1bec281594cea3050c8eb01311c"),
	value: mustHex("00000000000000000000000000000000000000000000000000000000000003e8"),
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func addrBytes(hex20 string) []byte {
	b, _ := hex.DecodeString(hex20)
	return append([]byte{0x41}, b...)
}

func TestDecodeTRC20Transfer(t *testing.T) {
	BuiltinTRC20()

	log, err := Decode([][]byte{
		mustHex(transferFixture.sig),
		transferFixture.from,
		transferFixture.to,
	}, transferFixture.value)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if log.EventName != "Transfer" {
		t.Fatalf("EventName = %q, want Transfer", log.EventName)
	}
	if len(log.Parameters) != 3 {
		t.Fatalf("params = %d, want 3", len(log.Parameters))
	}
	from, ok := log.Parameters[0].Value.(tron.Address)
	if !ok {
		t.Fatalf("from param type = %T, want tron.Address", log.Parameters[0].Value)
	}
	if string(from.Bytes()) != string(addrBytes("a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")) {
		t.Fatalf("from address mismatch: %x", from.Bytes())
	}
	to, ok := log.Parameters[1].Value.(tron.Address)
	if !ok {
		t.Fatalf("to param type = %T, want tron.Address", log.Parameters[1].Value)
	}
	if string(to.Bytes()) != string(addrBytes("4e83362442b8d1bec281594cea3050c8eb01311c")) {
		t.Fatalf("to address mismatch: %x", to.Bytes())
	}
	val, ok := log.Parameters[2].Value.(*big.Int)
	if !ok {
		t.Fatalf("value param type = %T, want *big.Int", log.Parameters[2].Value)
	}
	if val.Cmp(big.NewInt(1000)) != 0 {
		t.Fatalf("value = %s, want 1000", val)
	}
	if log.Parameters[0].Name != "from" || log.Parameters[1].Name != "to" || log.Parameters[2].Name != "value" {
		t.Fatalf("param names = %v %v %v", log.Parameters[0].Name, log.Parameters[1].Name, log.Parameters[2].Name)
	}
}

func TestDecodeUnknownSignature(t *testing.T) {
	BuiltinTRC20()

	// 0x12345678 prefix matches no registered definition.
	sig := append([]byte{0x12, 0x34, 0x56, 0x78}, make([]byte, 28)...)
	_, err := Decode([][]byte{sig}, nil)
	if err == nil {
		t.Fatal("expected error for unknown signature, got nil")
	}
	if !tron.HasCode(err, tron.CodeEventUnknown) {
		t.Fatalf("error code = %v, want event.unknown", err)
	}
}

func TestRegisterABIJSONAndDecodeCustomEvent(t *testing.T) {
	const abiJSON = `[
		{"type":"event","name":"Payout","inputs":[
			{"name":"recipient","type":"address","indexed":true},
			{"name":"amount","type":"uint256","indexed":false}
		]}
	]`
	if err := RegisterABIJSON(abiJSON); err != nil {
		t.Fatalf("RegisterABIJSON: %v", err)
	}

	data := make([]byte, 32)
	new(big.Int).SetInt64(42).FillBytes(data)
	log, err := Decode([][]byte{
		keccakTopic(t, "Payout(address,uint256)"),
		padAddr("1111111111111111111111111111111111111111"),
	}, data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if log.EventName != "Payout" {
		t.Fatalf("EventName = %q, want Payout", log.EventName)
	}
	if len(log.Parameters) != 2 {
		t.Fatalf("params = %d, want 2", len(log.Parameters))
	}
	if _, ok := log.Parameters[0].Value.(tron.Address); !ok {
		t.Fatalf("recipient type = %T, want tron.Address", log.Parameters[0].Value)
	}
	if v := log.Parameters[1].Value.(*big.Int); v.Int64() != 42 {
		t.Fatalf("amount = %s, want 42", v)
	}
}

func TestRegisterABIObject(t *testing.T) {
	parsed, err := newSimpleABIParser().parseABI(`[
		{"type":"event","name":"Ping","inputs":[{"name":"n","type":"uint256","indexed":false}]}
	]`)
	if err != nil {
		t.Fatalf("parseABI: %v", err)
	}
	if err := RegisterABIObject(parsed); err != nil {
		t.Fatalf("RegisterABIObject: %v", err)
	}
	if err := RegisterABIObject(nil); err == nil {
		t.Fatal("expected error for nil ABI")
	}
}

func TestDecodeMalformedDataLength(t *testing.T) {
	BuiltinTRC20()

	// Approval has one non-indexed uint256; short data must be a decode
	// error, NOT event.unknown (the signature IS known).
	sig := keccakTopic(t, "Approval(address,address,uint256)")
	_, err := Decode([][]byte{
		sig,
		padAddr("1111111111111111111111111111111111111111"),
		padAddr("2222222222222222222222222222222222222222"),
	}, []byte{0x01, 0x02}) // 2 bytes, want 32
	if err == nil {
		t.Fatal("expected decode error for short data, got nil")
	}
	if tron.HasCode(err, tron.CodeEventUnknown) {
		t.Fatalf("short data must not be event.unknown, got: %v", err)
	}
}

func TestDecodeInputValidation(t *testing.T) {
	if _, err := Decode(nil, nil); err == nil {
		t.Fatal("expected error for no topics")
	}
	if _, err := Decode([][]byte{{0x01, 0x02}}, nil); err == nil {
		t.Fatal("expected error for short signature topic")
	}
}

func TestDecodeIndexedBoolAndBytes32(t *testing.T) {
	const abiJSON = `[
		{"type":"event","name":"Flagged","inputs":[
			{"name":"on","type":"bool","indexed":true},
			{"name":"digest","type":"bytes32","indexed":false}
		]}
	]`
	if err := RegisterABIJSON(abiJSON); err != nil {
		t.Fatalf("RegisterABIJSON: %v", err)
	}
	digest := mustHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	data := append([]byte{}, digest...)
	log, err := Decode([][]byte{
		keccakTopic(t, "Flagged(bool,bytes32)"),
		append(make([]byte, 31), 0x01),
	}, data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if v := log.Parameters[0].Value.(bool); !v {
		t.Fatalf("on = %v, want true", log.Parameters[0].Value)
	}
	b, ok := log.Parameters[1].Value.([]byte)
	if !ok || string(b) != string(digest) {
		t.Fatalf("digest = %x (%T), want %x", log.Parameters[1].Value, log.Parameters[1].Value, digest)
	}
}

// guard: the ABI type parsing used for data decoding must reject garbage.
func TestRegisterABIJSONBadJSON(t *testing.T) {
	if err := RegisterABIJSON("{not json"); err == nil {
		t.Fatal("expected error for bad JSON")
	}
}
