package event

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/kslamph/tronlib/v2/tron"
)

// This file pins the fix for the P1 mis-decode: the registry used to key
// definitions by the FIRST 4 BYTES of the signature hash and let a later
// registration overwrite an earlier one, so a log could be decoded against
// another contract's definition. Three changes, each tested here:
//
//   - topics[0] must be the full 32-byte signature hash, and is the key;
//   - a conflicting registration makes the signature ambiguous (event.unknown,
//     materialized raw by the lenient path) instead of silently winning;
//   - definitions can be registered per emitting address, which is the only
//     information that tells two layouts of one signature apart.
//
// Every test uses its own event names: the registry is process-wide mutable
// state, so a signature one test makes ambiguous must not be reused by another.

// Two layouts of ONE canonical signature ("Mixed(address,address,uint256)"):
// the standard TRC-20 layout, and a variant whose second parameter is not
// indexed. Indexed status is not part of the signature string, so both layouts
// hash to the same topic — which is precisely why the signature alone cannot
// select a definition.
const (
	mixedABIIndexedBoth = `[
		{"type":"event","name":"Mixed","inputs":[
			{"name":"a","type":"address","indexed":true},
			{"name":"b","type":"address","indexed":true},
			{"name":"v","type":"uint256","indexed":false}
		]}
	]`
	mixedABIIndexedFirst = `[
		{"type":"event","name":"Mixed","inputs":[
			{"name":"a","type":"address","indexed":true},
			{"name":"b","type":"address","indexed":false},
			{"name":"v","type":"uint256","indexed":false}
		]}
	]`
)

func mixedSigTopic(t *testing.T) []byte {
	t.Helper()
	return crypto.Keccak256([]byte("Mixed(address,address,uint256)"))
}

// tronAddr builds a tron.Address from a 20-byte hex payload: an emitting
// contract to register a scope against.
func tronAddr(t *testing.T, hex20 string) tron.Address {
	t.Helper()
	a, err := tron.AddressFromBytes(addrBytes(hex20))
	if err != nil {
		t.Fatalf("address %s: %v", hex20, err)
	}
	return a
}

// abiWithLayout builds a one-event ABI JSON for name over the input types
// address,address,uint256, with the indexed flags supplied by the caller.
func abiWithLayout(name string, indexed ...bool) string {
	types := []string{"address", "address", "uint256"}
	params := make([]string, 0, len(indexed))
	for i, idx := range indexed {
		params = append(params, fmt.Sprintf(`{"name":"p%d","type":"%s","indexed":%t}`, i, types[i], idx))
	}
	return fmt.Sprintf(`[{"type":"event","name":"%s","inputs":[%s]}]`, name, strings.Join(params, ","))
}

// addrWord/valueWord are the two address topics and the uint256 value used
// across the layout tests; layout A indexes both addresses, layout B only the
// first, so B's data carries the second address ahead of the value.
var (
	wordA = padAddr("1111111111111111111111111111111111111111")
	wordB = padAddr("2222222222222222222222222222222222222222")
)

func valueWord(v int64) []byte { return u256Topic(v) }

// TestAddressScopedRegistrationDecodesEachEmittersLayout is the P1 scenario at
// registry level: two contracts share a canonical signature and disagree about
// which parameters are indexed. Registered per address, both decode; neither
// ever borrows the other's layout.
func TestAddressScopedRegistrationDecodesEachEmittersLayout(t *testing.T) {
	const name = "Scoped"
	sig := keccakTopic(t, name+"(address,address,uint256)")
	contractA := tronAddr(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	contractB := tronAddr(t, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	value := valueWord(1000)

	if err := RegisterABIJSONForAddress(contractA, abiWithLayout(name, true, true, false)); err != nil {
		t.Fatalf("RegisterABIJSONForAddress(A): %v", err)
	}
	if err := RegisterABIJSONForAddress(contractB, abiWithLayout(name, true, false, false)); err != nil {
		t.Fatalf("RegisterABIJSONForAddress(B): %v", err)
	}

	logA, err := DecodeFor(contractA, [][]byte{sig, wordA, wordB}, value)
	if err != nil {
		t.Fatalf("DecodeFor(A): %v", err)
	}
	logB, err := DecodeFor(contractB, [][]byte{sig, wordA}, append(append([]byte{}, wordB...), value...))
	if err != nil {
		t.Fatalf("DecodeFor(B): %v", err)
	}

	if logA.EventName != name || logB.EventName != name {
		t.Fatalf("event names = %q, %q, want %q", logA.EventName, logB.EventName, name)
	}
	if logA.Address != contractA || logB.Address != contractB {
		t.Errorf("DecodeFor must record the emitting address: got %v and %v", logA.Address, logB.Address)
	}
	if len(logA.Parameters) != 3 || len(logB.Parameters) != 3 {
		t.Fatalf("parameters = %d and %d, want 3 each", len(logA.Parameters), len(logB.Parameters))
	}
	// The two layouts differ only in where p1 comes from; both must report the
	// same value for it, taken from the right place.
	for label, lg := range map[string]*Log{"A": logA, "B": logB} {
		p0, ok := lg.Parameters[0].Value.(tron.Address)
		if !ok || string(p0.Bytes()) != string(addrBytes("1111111111111111111111111111111111111111")) {
			t.Errorf("%s p0 = %#v (%T), want the first address", label, lg.Parameters[0].Value, lg.Parameters[0].Value)
		}
		p1, ok := lg.Parameters[1].Value.(tron.Address)
		if !ok || string(p1.Bytes()) != string(addrBytes("2222222222222222222222222222222222222222")) {
			t.Errorf("%s p1 = %#v (%T), want the second address", label, lg.Parameters[1].Value, lg.Parameters[1].Value)
		}
		v, ok := lg.Parameters[2].Value.(*big.Int)
		if !ok || v.Int64() != 1000 {
			t.Errorf("%s p2 = %#v (%T), want 1000", label, lg.Parameters[2].Value, lg.Parameters[2].Value)
		}
	}

	// Scope is a preference, not a filter: an address that registered nothing
	// falls back to the global registry, which knows neither layout here, and is
	// refused rather than guessed.
	stranger := tronAddr(t, "cccccccccccccccccccccccccccccccccccccccc")
	if _, err := DecodeFor(stranger, [][]byte{sig, wordA, wordB}, value); !tron.HasCode(err, tron.CodeEventUnknown) {
		t.Errorf("DecodeFor(unregistered address): err = %v, want event.unknown", err)
	}
	if _, err := Decode([][]byte{sig, wordA, wordB}, value); !tron.HasCode(err, tron.CodeEventUnknown) {
		t.Errorf("Decode (global, signature registered only per address): err = %v, want event.unknown", err)
	}
	// The scoped registrations must not have leaked into the global registry.
	if sigName, ok := DecodeEventSignature(sig); ok || sigName != "" {
		t.Errorf("DecodeEventSignature = (%q, %v), want unknown: scoped registration leaked globally", sigName, ok)
	}
}

// TestAddressScopedRegistrationResolvesAnAmbiguousGlobalSignature is the
// recovery path the ambiguity error promises: once two global registrations
// collide, only the emitting address can pick a layout.
func TestAddressScopedRegistrationResolvesAnAmbiguousGlobalSignature(t *testing.T) {
	const name = "Ambiguous"
	sig := keccakTopic(t, name+"(address,address,uint256)")
	value := valueWord(5)

	if err := RegisterABIJSON(abiWithLayout(name, true, true, false)); err != nil {
		t.Fatalf("RegisterABIJSON(layout A): %v", err)
	}
	if err := RegisterABIJSON(abiWithLayout(name, true, false, false)); err != nil {
		t.Fatalf("RegisterABIJSON(layout B): %v", err)
	}

	scopeAddr := tronAddr(t, "dddddddddddddddddddddddddddddddddddddddd")
	if err := RegisterABIJSONForAddress(scopeAddr, abiWithLayout(name, true, true, false)); err != nil {
		t.Fatalf("RegisterABIJSONForAddress: %v", err)
	}
	lg, err := DecodeFor(scopeAddr, [][]byte{sig, wordA, wordB}, value)
	if err != nil {
		t.Fatalf("an address-scoped definition must decode over an ambiguous global signature: %v", err)
	}
	if lg.EventName != name || len(lg.Parameters) != 3 {
		t.Errorf("scoped decode = %+v, want %q with 3 parameters", lg, name)
	}
	// The global conflict still refuses everyone else.
	if _, err := DecodeFor(tronAddr(t, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"), [][]byte{sig, wordA, wordB}, value); !tron.HasCode(err, tron.CodeEventUnknown) {
		t.Errorf("unrelated address: err = %v, want event.unknown", err)
	}
	if _, err := Decode([][]byte{sig, wordA, wordB}, value); !tron.HasCode(err, tron.CodeEventUnknown) {
		t.Errorf("global decode: err = %v, want event.unknown", err)
	}
}

// TestRegisterABIJSONConflictingLayoutsRefusesToGuess is the regression for the
// silent-overwrite path: after a conflicting registration the log that decoded
// correctly before must not decode at all. Reporting the surviving definition's
// fields against the other layout is how wrong values reached a caller;
// event.unknown is the honest answer until the caller scopes the registration.
func TestRegisterABIJSONConflictingLayoutsRefusesToGuess(t *testing.T) {
	if err := RegisterABIJSON(mixedABIIndexedBoth); err != nil {
		t.Fatalf("RegisterABIJSON(layout A): %v", err)
	}

	topicsA := [][]byte{mixedSigTopic(t), wordA, wordB}
	dataA := valueWord(7)
	topicsB := [][]byte{mixedSigTopic(t), wordA}
	dataB := append(append([]byte{}, wordB...), valueWord(7)...)

	logA, err := Decode(topicsA, dataA)
	if err != nil {
		t.Fatalf("decode before the conflicting registration: %v", err)
	}
	if len(logA.Parameters) != 3 {
		t.Fatalf("parameters = %d, want 3", len(logA.Parameters))
	}

	if err := RegisterABIJSON(mixedABIIndexedFirst); err != nil {
		t.Fatalf("RegisterABIJSON(layout B): %v", err)
	}

	for label, tc := range map[string]struct {
		topics [][]byte
		data   []byte
	}{
		"layout A (two indexed topics)": {topicsA, dataA},
		"layout B (one indexed topic)":  {topicsB, dataB},
	} {
		_, err := Decode(tc.topics, tc.data)
		if err == nil {
			t.Errorf("%s: Decode succeeded after a conflicting registration, want event.unknown", label)
			continue
		}
		if !tron.HasCode(err, tron.CodeEventUnknown) {
			t.Errorf("%s: err = %v, want event.unknown (refuse to guess, never guess wrong)", label, err)
		}
		if e, ok := err.(*tron.Error); ok && !strings.Contains(e.Hint, "RegisterABIJSONForAddress") {
			t.Errorf("%s: hint %q must point at address-scoped registration", label, e.Hint)
		}
	}

	// The lenient path keeps the log, unnamed, with raw bytes preserved.
	raw, err := DecodeLenient(topicsA, dataA)
	if err != nil {
		t.Fatalf("DecodeLenient after conflict: %v", err)
	}
	if raw.EventName != "" || len(raw.Parameters) != 0 {
		t.Errorf("ambiguous log materialized as %+v, want EventName \"\" and no parameters", raw)
	}
	if !bytes.Equal(raw.Topics[1], wordA) || !bytes.Equal(raw.Data, dataA) {
		t.Errorf("ambiguous log lost raw bytes: topics=%x data=%x", raw.Topics, raw.Data)
	}
	// DecodeFor for an address that registered nothing reports the same
	// ambiguity rather than falling through to a guess.
	if _, err := DecodeLenientFor(tronAddr(t, "9999999999999999999999999999999999999999"), topicsA, dataA); err != nil {
		t.Errorf("DecodeLenientFor must materialize an ambiguous global signature, got %v", err)
	}
}

// TestIdenticalReRegistrationIsANoOp guards the other half of the rule: the same
// ABI registered again — through either entry point, globally or scoped — stays
// decodable instead of being flagged as a conflict.
func TestIdenticalReRegistrationIsANoOp(t *testing.T) {
	const name = "Repeated"
	abi := abiWithLayout(name, true, false, false)
	sig := keccakTopic(t, name+"(address,address,uint256)")
	topics := [][]byte{sig, wordA}
	data := append(append([]byte{}, wordB...), valueWord(9)...)

	if err := RegisterABIJSON(abi); err != nil {
		t.Fatalf("first RegisterABIJSON: %v", err)
	}
	first, err := Decode(topics, data)
	if err != nil {
		t.Fatalf("Decode after the first registration: %v", err)
	}
	if first.EventName != name || len(first.Parameters) != 3 {
		t.Fatalf("first decode = %+v, want %q with 3 parameters", first, name)
	}

	parsed, err := newSimpleABIParser().parseABI(abi)
	if err != nil {
		t.Fatalf("parseABI: %v", err)
	}
	for label, reg := range map[string]func() error{
		"RegisterABIJSON again":              func() error { return RegisterABIJSON(abi) },
		"RegisterABIObject of the same ABI":  func() error { return RegisterABIObject(parsed) },
		"RegisterABIJSONForAddress again":    func() error { return RegisterABIJSONForAddress(scopedAddr(t), abi) },
		"RegisterABIObjectForAddress again":  func() error { return RegisterABIObjectForAddress(scopedAddr(t), parsed) },
		"RegisterABIJSONForAddress third":    func() error { return RegisterABIJSONForAddress(scopedAddr(t), abi) },
		"RegisterABIObjectForAddress third ": func() error { return RegisterABIObjectForAddress(scopedAddr(t), parsed) },
	} {
		if err := reg(); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	}

	if _, err := Decode(topics, data); err != nil {
		t.Fatalf("Decode after identical re-registrations: %v", err)
	}
	if _, err := DecodeFor(scopedAddr(t), topics, data); err != nil {
		t.Fatalf("DecodeFor after identical re-registrations: %v", err)
	}
}

func scopedAddr(t *testing.T) tron.Address {
	t.Helper()
	return tronAddr(t, "ffffffffffffffffffffffffffffffffffffffff")
}

// Evt055828 and Evt042226 hash to signatures that share their first 4 bytes
// (c6853953) and differ in the remaining 28 — found by brute-forcing keccak256
// over "EvtNNNNNN(address,uint256)". Under prefix keying the later registration
// took the shared slot, so a log of the FIRST signature decoded against the
// second contract's definition and reported the wrong event name for
// correctly-shaped bytes.
const (
	prefixCollidingA = "Evt055828" // c68539536cd45d55afffe2722736fb46b647248f9ef773d168b85fa9d49c2859
	prefixCollidingB = "Evt042226" // c68539538516ddd74c78dc3c422891e05f349f244e4221661916c1b085ad06a3
)

func TestSignaturesSharingAPrefixDoNotShareADefinition(t *testing.T) {
	abiA := abiWithPrefixCollisionFixture(prefixCollidingA)
	abiB := abiWithPrefixCollisionFixture(prefixCollidingB)
	if err := RegisterABIJSON(abiA); err != nil {
		t.Fatalf("RegisterABIJSON(%s): %v", prefixCollidingA, err)
	}
	if err := RegisterABIJSON(abiB); err != nil {
		t.Fatalf("RegisterABIJSON(%s): %v", prefixCollidingB, err)
	}

	topicA := mustHex("c68539536cd45d55afffe2722736fb46b647248f9ef773d168b85fa9d49c2859")
	topicB := mustHex("c68539538516ddd74c78dc3c422891e05f349f244e4221661916c1b085ad06a3")
	// Keep the fixture honest: same 4-byte prefix, different full hash. If
	// either stops holding, the test below proves nothing.
	if !bytes.Equal(crypto.Keccak256([]byte(prefixCollidingA+"(address,uint256)")), topicA) ||
		!bytes.Equal(crypto.Keccak256([]byte(prefixCollidingB+"(address,uint256)")), topicB) {
		t.Fatal("prefix-collision fixtures no longer match their signatures")
	}
	if !bytes.Equal(topicA[:4], topicB[:4]) {
		t.Fatal("prefix-collision fixtures no longer share a 4-byte prefix")
	}

	value := valueWord(7)
	for label, tc := range map[string]struct {
		topic []byte
		want  string
	}{
		"A": {topicA, prefixCollidingA},
		"B": {topicB, prefixCollidingB},
	} {
		lg, err := Decode([][]byte{tc.topic, wordA}, value)
		if err != nil {
			t.Fatalf("%s: Decode: %v", label, err)
		}
		if lg.EventName != tc.want {
			t.Errorf("%s: EventName = %q, want %q — a prefix collision resolved to the wrong definition", label, lg.EventName, tc.want)
		}
		if sig, ok := DecodeEventSignature(tc.topic); !ok || sig != tc.want+"(address,uint256)" {
			t.Errorf("%s: DecodeEventSignature = (%q, %v), want the topic's own signature", label, sig, ok)
		}
	}
}

func abiWithPrefixCollisionFixture(name string) string {
	return fmt.Sprintf(`[{"type":"event","name":"%s","inputs":[
		{"name":"who","type":"address","indexed":true},
		{"name":"amount","type":"uint256","indexed":false}]}]`, name)
}

// TestDecodeRequiresFullSignatureTopic is the regression for 4-byte keying: a
// truncated topic used to BE a registry key, so a prefix could select a
// definition that the full hash contradicts.
func TestDecodeRequiresFullSignatureTopic(t *testing.T) {
	if err := RegisterABIJSON(mixedABIIndexedBoth); err != nil {
		t.Fatalf("RegisterABIJSON: %v", err)
	}
	full := mixedSigTopic(t)
	for _, n := range []int{0, 1, 4, 31} {
		topic := make([]byte, n)
		copy(topic, full[:n])
		_, err := Decode([][]byte{topic, wordA, wordB}, valueWord(7))
		if !tron.HasCode(err, tron.CodeContractArgMismatch) {
			t.Errorf("%d-byte signature topic: err = %v, want contract.arg_mismatch", n, err)
		}
	}
	// Longer than 32 is malformed for the same reason: it is not a keccak-256
	// digest, so it can never be a registry key.
	long := append(append([]byte{}, full...), 0x01)
	if _, err := Decode([][]byte{long, wordA, wordB}, valueWord(7)); !tron.HasCode(err, tron.CodeContractArgMismatch) {
		t.Errorf("33-byte signature topic: err = %v, want contract.arg_mismatch", err)
	}
	// The same shape rule applies to DecodeFor.
	if _, err := DecodeFor(tronAddr(t, "1212121212121212121212121212121212121212"), [][]byte{full[:4], wordA, wordB}, valueWord(7)); !tron.HasCode(err, tron.CodeContractArgMismatch) {
		t.Errorf("DecodeFor with a 4-byte topic: err = %v, want contract.arg_mismatch", err)
	}
}

// TestBuiltinTransferDecodesForAnyUnregisteredContract pins that address
// scoping does not cost the zero-config guarantee: a TRC-20 Transfer from a
// contract that registered nothing still resolves through the built-in
// registry — and a different Transfer layout scoped to some OTHER contract
// cannot disturb that answer.
func TestBuiltinTransferDecodesForAnyUnregisteredContract(t *testing.T) {
	BuiltinTRC20()
	sig := mustHex(transferFixture.sig)

	odd := tronAddr(t, "0000000000000000000000000000000000000001")
	oddABI := `[{"type":"event","name":"Transfer","inputs":[
		{"name":"from","type":"address","indexed":true},
		{"name":"to","type":"address","indexed":false},
		{"name":"value","type":"uint256","indexed":false}]}]`
	if err := RegisterABIJSONForAddress(odd, oddABI); err != nil {
		t.Fatalf("RegisterABIJSONForAddress: %v", err)
	}

	token := tronAddr(t, "a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	lg, err := DecodeLenientFor(token, [][]byte{sig, transferFixture.from, transferFixture.to}, transferFixture.value)
	if err != nil {
		t.Fatalf("DecodeLenientFor(builtin Transfer, unregistered contract): %v", err)
	}
	if lg.EventName != "Transfer" || len(lg.Parameters) != 3 {
		t.Fatalf("builtin Transfer decoded as %+v, want EventName Transfer with 3 parameters", lg)
	}
	if v, ok := lg.Parameters[2].Value.(*big.Int); !ok || v.Int64() != 1000 {
		t.Errorf("value = %#v (%T), want 1000", lg.Parameters[2].Value, lg.Parameters[2].Value)
	}
	if lg.Address != token {
		t.Errorf("Address = %v, want the emitting contract %v", lg.Address, token)
	}

	// The scoped layout is what its own contract means: the same topic with
	// only one indexed topic decodes there, and nowhere else.
	if _, err := DecodeFor(odd, [][]byte{sig, transferFixture.from}, append(append([]byte{}, transferFixture.to...), transferFixture.value...)); err != nil {
		t.Errorf("DecodeFor(odd contract, its own layout): %v", err)
	}
	if _, err := DecodeFor(token, [][]byte{sig, transferFixture.from}, append(append([]byte{}, transferFixture.to...), transferFixture.value...)); err == nil {
		t.Error("DecodeFor(TRC-20 contract, one indexed topic): want an error, the odd layout is not this contract's")
	}
}

// TestRegisterABIForAddressRejectsUnsetAddress: the zero Address means "the
// global registry", so accepting it as a scope would let a caller who believes
// they registered privately change every contract's decoding.
func TestRegisterABIForAddressRejectsUnsetAddress(t *testing.T) {
	const name = "UnsetScope"
	abi := abiWithLayout(name, true, true, false)
	parsed, err := newSimpleABIParser().parseABI(abi)
	if err != nil {
		t.Fatalf("parseABI: %v", err)
	}
	addr := tronAddr(t, "aabbccddee00112233445566778899aabbccddee")

	for label, reg := range map[string]func() error{
		"RegisterABIJSONForAddress(zero)":      func() error { return RegisterABIJSONForAddress(tron.Address{}, abi) },
		"RegisterABIObjectForAddress(zero)":    func() error { return RegisterABIObjectForAddress(tron.Address{}, parsed) },
		"RegisterABIObjectForAddress(nil ABI)": func() error { return RegisterABIObjectForAddress(addr, nil) },
		"RegisterABIJSONForAddress(bad JSON)":  func() error { return RegisterABIJSONForAddress(addr, "{not json") },
	} {
		err := reg()
		if err == nil {
			t.Errorf("%s: err = nil, want an error", label)
			continue
		}
		if strings.Contains(label, "zero") && !tron.HasCode(err, tron.CodeAddressInvalid) {
			t.Errorf("%s: err = %v, want address.invalid", label, err)
		}
	}

	// Rejection must not have registered anything under the zero address's
	// scope or globally: the signature stays unknown.
	if _, err := Decode([][]byte{keccakTopic(t, name+"(address,address,uint256)"), wordA, wordB}, valueWord(1)); !tron.HasCode(err, tron.CodeEventUnknown) {
		t.Errorf("rejected registration still took effect: err = %v", err)
	}
}

// TestDecodeEventSignatureRequiresFullTopic pins the narrowing that follows
// from full-hash keying: a bare selector is no longer enough to name an event.
func TestDecodeEventSignatureRequiresFullTopic(t *testing.T) {
	topic := keccakTopic(t, "Approval(address,address,uint256)")
	if sig, ok := DecodeEventSignature(topic); !ok || sig != "Approval(address,address,uint256)" {
		t.Fatalf("full topic: (%q, %v), want the canonical signature", sig, ok)
	}
	for label, in := range map[string][]byte{
		"4-byte selector": topic[:4],
		"31-byte topic":   topic[:31],
		"33-byte topic":   append(append([]byte{}, topic...), 0x01),
	} {
		if sig, ok := DecodeEventSignature(in); ok || sig != "" {
			t.Errorf("%s: (%q, %v), want (\"\", false)", label, sig, ok)
		}
	}
}
