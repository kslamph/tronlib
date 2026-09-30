package event

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
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

// --- FIX 1: full vendored builtin registry ---

// TestBuiltinTableCountAndKeys asserts the generated table keeps at least the
// 747-entry curated baseline and that every entry's map key — the full 32-byte
// signature hash — equals the hash derived from the definition itself, that the
// keys are distinct, and that each definition is actually reachable in the
// global registry by that key. Together those prove the table and the registry
// agree on the key space and that no built-in is shadowed or lost.
//
// The reconstruction covers every entry, tuple and trcToken ones included:
// the generator hashed the same literal type strings the table stores, which is
// what makes the derivation exact.
func TestBuiltinTableCountAndKeys(t *testing.T) {
	if len(builtinSig) < 747 {
		t.Fatalf("builtin table has %d entries, want at least the 747 curated entries", len(builtinSig))
	}
	derived := make(map[sigKey]string, len(builtinSig))
	for key, def := range builtinSig {
		full := sigKeyOf(def.signature())
		sig := def.Name + "(" + strings.Join(inputTypes(def), ",") + ")"
		if full != key {
			t.Errorf("builtin %s: table key %x != keccak256(sig) %x", sig, key, full)
		}
		if prev, dup := derived[full]; dup {
			t.Errorf("builtin %s and %s derive the same registry key", sig, prev)
		}
		derived[full] = sig
		if registered := globalDef(full); registered != def {
			t.Errorf("builtin %s: global registry holds %v under its derived key, want the table's definition", sig, registered)
		}
	}
}

// inputTypes lists a definition's ABI parameter types in declared order.
func inputTypes(def *Definition) []string {
	types := make([]string, len(def.Inputs))
	for i, in := range def.Inputs {
		types[i] = in.Type
	}
	return types
}

// TestDecodeBuiltinSubmitTransaction decodes a NON-TRC-20 builtin end to
// end: SubmitTransaction(uint256,address,uint256,bytes) from the generated
// table, with indexed and non-indexed params.
func TestDecodeBuiltinSubmitTransaction(t *testing.T) {
	sigTopic := keccakTopic(t, "SubmitTransaction(uint256,address,uint256,bytes)")
	key := sigKeyOf("SubmitTransaction(uint256,address,uint256,bytes)")
	if def := builtinSig[key]; def == nil || def.Name != "SubmitTransaction" {
		t.Fatal("SubmitTransaction missing from builtin table under its full signature key")
	}

	// data = uint256 value(1000) + offset(0x40) + len(5) + "hello" padded.
	data := new(bytes.Buffer)
	value := make([]byte, 32)
	new(big.Int).SetInt64(1000).FillBytes(value)
	data.Write(value)
	offset := make([]byte, 32)
	offset[31] = 0x40
	data.Write(offset)
	length := make([]byte, 32)
	length[31] = 5
	data.Write(length)
	payload := append([]byte("hello"), make([]byte, 27)...)
	data.Write(payload)

	log, err := Decode([][]byte{
		sigTopic,
		u256Topic(7), // indexed txId
		padAddr("4e83362442b8d1bec281594cea3050c8eb01311c"), // indexed to
	}, data.Bytes())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if log.EventName != "SubmitTransaction" {
		t.Fatalf("EventName = %q, want SubmitTransaction", log.EventName)
	}
	if len(log.Parameters) != 4 {
		t.Fatalf("params = %d, want 4", len(log.Parameters))
	}
	if v := log.Parameters[0].Value.(*big.Int); v.Int64() != 7 {
		t.Fatalf("txId = %s, want 7", v)
	}
	if _, ok := log.Parameters[1].Value.(tron.Address); !ok {
		t.Fatalf("to type = %T, want tron.Address", log.Parameters[1].Value)
	}
	if v := log.Parameters[2].Value.(*big.Int); v.Int64() != 1000 {
		t.Fatalf("value = %s, want 1000", v)
	}
	if b, ok := log.Parameters[3].Value.([]byte); !ok || string(b) != "hello" {
		t.Fatalf("data = %x (%T), want \"hello\"", log.Parameters[3].Value, log.Parameters[3].Value)
	}
}

// --- FIX 2: DecodeLenient ---

func TestDecodeLenientUnknownSignature(t *testing.T) {
	sig := append([]byte{0x12, 0x34, 0x56, 0x78}, make([]byte, 28)...)
	data := []byte{0xde, 0xad}
	log, err := DecodeLenient([][]byte{sig}, data)
	if err != nil {
		t.Fatalf("DecodeLenient(unknown) = %v, want nil error", err)
	}
	if log == nil || log.EventName != "" {
		t.Fatalf("EventName = %q, want empty", log.EventName)
	}
	if len(log.Parameters) != 0 {
		t.Fatalf("params = %d, want 0", len(log.Parameters))
	}
	if string(log.Topics[0]) != string(sig) || string(log.Data) != string(data) {
		t.Fatal("raw Topics/Data not preserved")
	}
}

func TestDecodeLenientKnownEventMatchesDecode(t *testing.T) {
	decoded, err := Decode([][]byte{
		mustHex(transferFixture.sig),
		transferFixture.from,
		transferFixture.to,
	}, transferFixture.value)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	lenient, err := DecodeLenient([][]byte{
		mustHex(transferFixture.sig),
		transferFixture.from,
		transferFixture.to,
	}, transferFixture.value)
	if err != nil {
		t.Fatalf("DecodeLenient: %v", err)
	}
	if lenient.EventName != decoded.EventName || len(lenient.Parameters) != len(decoded.Parameters) {
		t.Fatalf("DecodeLenient = %+v, want identical to Decode %+v", lenient, decoded)
	}
	if v := lenient.Parameters[2].Value.(*big.Int); v.Cmp(big.NewInt(1000)) != 0 {
		t.Fatalf("value = %s, want 1000", v)
	}
}

func TestDecodeLenientCorruptKnownEventErrors(t *testing.T) {
	// Approval (known) with short data must stay an error, not materialize
	// as an unnamed log.
	_, err := DecodeLenient([][]byte{
		keccakTopic(t, "Approval(address,address,uint256)"),
		padAddr("1111111111111111111111111111111111111111"),
		padAddr("2222222222222222222222222222222222222222"),
	}, []byte{0x01, 0x02})
	if err == nil {
		t.Fatal("expected error for corrupt known event")
	}
	if tron.HasCode(err, tron.CodeEventUnknown) {
		t.Fatalf("corrupt known event must not be event.unknown, got: %v", err)
	}
}

func TestDecodeLenientMalformedShapeErrors(t *testing.T) {
	if _, err := DecodeLenient(nil, nil); err == nil {
		t.Fatal("expected error for nil topics")
	}
	if _, err := DecodeLenient([][]byte{{0x01, 0x02}}, nil); err == nil {
		t.Fatal("expected error for short signature topic")
	}
}

// --- RIDE-1: signed int topics decode two's-complement ---

// TestDecodeIndexedNegativeInt: an int256 topic with the high bit set must
// decode to the negative two's-complement value. Diverges from v1, which
// decoded such topics as huge positives (big.Int.SetBytes is unsigned).
func TestDecodeIndexedNegativeInt(t *testing.T) {
	const abiJSON = `[
		{"type":"event","name":"SignedUp","inputs":[
			{"name":"negOne","type":"int256","indexed":true},
			{"name":"minInt256","type":"int256","indexed":true},
			{"name":"pos","type":"int256","indexed":true}
		]}
	]`
	if err := RegisterABIJSON(abiJSON); err != nil {
		t.Fatalf("RegisterABIJSON: %v", err)
	}
	log, err := Decode([][]byte{
		keccakTopic(t, "SignedUp(int256,int256,int256)"),
		bytes.Repeat([]byte{0xff}, 32),            // -1
		append([]byte{0x80}, make([]byte, 31)...), // -2^255
		u256Topic(42), // +42
	}, nil)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if v := log.Parameters[0].Value.(*big.Int); v.Cmp(big.NewInt(-1)) != 0 {
		t.Fatalf("negOne = %s, want -1", v)
	}
	wantMin := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255))
	if v := log.Parameters[1].Value.(*big.Int); v.Cmp(wantMin) != 0 {
		t.Fatalf("minInt256 = %s, want %s", v, wantMin)
	}
	if v := log.Parameters[2].Value.(*big.Int); v.Int64() != 42 {
		t.Fatalf("pos = %s, want 42", v)
	}
}

// --- RIDE-2: DecodeEventSignature ---

func TestDecodeEventSignature(t *testing.T) {
	sig, ok := DecodeEventSignature(mustHex(transferFixture.sig))
	if !ok || sig != "Transfer(address,address,uint256)" {
		t.Fatalf("got (%q, %v), want (Transfer(address,address,uint256), true)", sig, ok)
	}
	sig, ok = DecodeEventSignature(keccakTopic(t, "SubmitTransaction(uint256,address,uint256,bytes)"))
	if !ok || sig != "SubmitTransaction(uint256,address,uint256,bytes)" {
		t.Fatalf("got (%q, %v), want builtin canonical signature", sig, ok)
	}
	unknown := append([]byte{0x12, 0x34, 0x56, 0x78}, make([]byte, 28)...)
	if s, ok := DecodeEventSignature(unknown); ok || s != "" {
		t.Fatalf("unknown sig = (%q, %v), want (\"\", false)", s, ok)
	}
	if s, ok := DecodeEventSignature([]byte{0x01, 0x02}); ok || s != "" {
		t.Fatalf("short sig = (%q, %v), want (\"\", false)", s, ok)
	}
}

// u256Topic builds a 32-byte left-padded topic for a small non-negative int.
func u256Topic(v int64) []byte {
	out := make([]byte, 32)
	new(big.Int).SetInt64(v).FillBytes(out)
	return out
}
