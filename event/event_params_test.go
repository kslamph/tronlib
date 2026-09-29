package event

// Regression tests for decodeEvent's parameter reassembly. The combine step
// used to match decoded values to declared inputs by NAME, which is not a
// valid key: Solidity emits "" for unnamed parameters, nothing in the parse
// path validates names, and duplicate names are legal in an ABI. First-match
// semantics then return the same value for every repeated name and drop the
// rest. These tests pin the positional contract instead: one value per
// declared input, in declared order, each decoded value consumed exactly once.
//
// The tests drive decodeEvent directly with hand-built EventDefs so they
// exercise the combine step independently of the registry (which is what a
// name-keyed merge breaks only inside this function); the last test replays
// the same shape through the public RegisterABIJSON + Decode path to prove
// the defect is reachable from real ABI JSON.

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/kslamph/tronlib/v2/tron"
)

// wantUintParam asserts Parameters[i] is the *big.Int want, and that its name
// is the declared input name (unnamed inputs are "", per Solidity).
func wantUintParam(t *testing.T, log *Log, i int, name string, want int64) {
	t.Helper()
	if i >= len(log.Parameters) {
		t.Fatalf("Parameters has %d entries, want at least %d", len(log.Parameters), i+1)
	}
	p := log.Parameters[i]
	if p.Name != name {
		t.Errorf("Parameters[%d].Name = %q, want %q", i, p.Name, name)
	}
	v, ok := p.Value.(*big.Int)
	if !ok {
		t.Fatalf("Parameters[%d].Value = %T (%v), want *big.Int", i, p.Value, p.Value)
	}
	if v.Cmp(big.NewInt(want)) != 0 {
		t.Errorf("Parameters[%d].Value = %s, want %d", i, v, want)
	}
}

// wantAddressParam asserts Parameters[i] holds the tron.Address built from a
// 20-byte hex payload.
func wantAddressParam(t *testing.T, log *Log, i int, name, hex20 string) {
	t.Helper()
	if i >= len(log.Parameters) {
		t.Fatalf("Parameters has %d entries, want at least %d", len(log.Parameters), i+1)
	}
	p := log.Parameters[i]
	if p.Name != name {
		t.Errorf("Parameters[%d].Name = %q, want %q", i, p.Name, name)
	}
	got, ok := p.Value.(tron.Address)
	if !ok {
		t.Fatalf("Parameters[%d].Value = %T (%v), want tron.Address", i, p.Value, p.Value)
	}
	if string(got.Bytes()) != string(addrBytes(hex20)) {
		t.Errorf("Parameters[%d].Value = %x, want %x", i, got.Bytes(), addrBytes(hex20))
	}
}

// sigTopicFor builds a placeholder first topic (decodeEvent is handed an
// already-resolved definition, so the signature hash itself is not read).
func placeholderTopic() []byte { return make([]byte, 32) }

// Two unnamed indexed inputs are the canonical failing case: both names are
// "", so name matching collapses them onto the first value.
func TestDecodeEventUnnamedIndexedInputsAreNotDuplicated(t *testing.T) {
	def := &EventDef{
		Name: "UnnamedPair",
		Inputs: []ParamDef{
			{Type: "uint256", Indexed: true},
			{Type: "uint256", Indexed: true},
		},
	}
	log, err := decodeEvent(def, [][]byte{placeholderTopic(), u256Topic(1), u256Topic(2)}, nil, "event.Decode")
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if len(log.Parameters) != len(def.Inputs) {
		t.Fatalf("len(Parameters) = %d, want %d", len(log.Parameters), len(def.Inputs))
	}
	wantUintParam(t, log, 0, "", 1)
	wantUintParam(t, log, 1, "", 2)
}

// Duplicate names among indexed inputs collapse the same way, and a name
// shared across the indexed/non-indexed boundary must not cross-contaminate.
func TestDecodeEventDuplicateNamesKeepDistinctValues(t *testing.T) {
	def := &EventDef{
		Name: "Duplicated",
		Inputs: []ParamDef{
			{Type: "uint256", Indexed: true, Name: "value"},
			{Type: "uint256", Indexed: false, Name: "value"},
			{Type: "uint256", Indexed: true, Name: "value"},
			{Type: "uint256", Indexed: false, Name: "tail"},
		},
	}
	data := append(u256Topic(2), u256Topic(4)...)
	log, err := decodeEvent(def, [][]byte{placeholderTopic(), u256Topic(1), u256Topic(3)}, data, "event.Decode")
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if len(log.Parameters) != len(def.Inputs) {
		t.Fatalf("len(Parameters) = %d, want %d", len(log.Parameters), len(def.Inputs))
	}
	wantUintParam(t, log, 0, "value", 1)
	wantUintParam(t, log, 1, "value", 2)
	wantUintParam(t, log, 2, "value", 3)
	wantUintParam(t, log, 3, "tail", 4)
}

// Unnamed non-indexed parameters mixed with named ones, decoded through the
// geth ABI unpack path: the data word order must survive the merge.
func TestDecodeEventUnnamedNonIndexedMixedWithNamed(t *testing.T) {
	def := &EventDef{
		Name: "MixedUnnamed",
		Inputs: []ParamDef{
			{Type: "address", Indexed: true, Name: "owner"},
			{Type: "uint256", Indexed: false},
			{Type: "address", Indexed: false, Name: "to"},
			{Type: "uint256", Indexed: false},
		},
	}
	data := append(u256Topic(7), padAddr("2222222222222222222222222222222222222222")...)
	data = append(data, u256Topic(9)...)

	log, err := decodeEvent(def, [][]byte{placeholderTopic(), padAddr("1111111111111111111111111111111111111111")}, data, "event.Decode")
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if len(log.Parameters) != len(def.Inputs) {
		t.Fatalf("len(Parameters) = %d, want %d", len(log.Parameters), len(def.Inputs))
	}
	wantAddressParam(t, log, 0, "owner", "1111111111111111111111111111111111111111")
	wantUintParam(t, log, 1, "", 7)
	wantAddressParam(t, log, 2, "to", "2222222222222222222222222222222222222222")
	wantUintParam(t, log, 3, "", 9)
}

// Unnamed inputs on both sides of the indexed boundary at once.
func TestDecodeEventUnnamedOnBothSides(t *testing.T) {
	def := &EventDef{
		Name: "AllUnnamed",
		Inputs: []ParamDef{
			{Type: "uint256", Indexed: true},
			{Type: "uint256", Indexed: false},
			{Type: "uint256", Indexed: true},
			{Type: "uint256", Indexed: false},
		},
	}
	data := append(u256Topic(20), u256Topic(40)...)
	log, err := decodeEvent(def, [][]byte{placeholderTopic(), u256Topic(10), u256Topic(30)}, data, "event.Decode")
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if len(log.Parameters) != len(def.Inputs) {
		t.Fatalf("len(Parameters) = %d, want %d", len(log.Parameters), len(def.Inputs))
	}
	for i, want := range []int64{10, 20, 30, 40} {
		wantUintParam(t, log, i, "", want)
	}
}

// Fully named inputs, interleaved across topics and data, must still come
// back in declared order — the positional merge must not reorder them.
func TestDecodeEventNamedInputsKeepDeclaredOrder(t *testing.T) {
	def := &EventDef{
		Name: "Ordered",
		Inputs: []ParamDef{
			{Type: "uint256", Indexed: true, Name: "a"},
			{Type: "uint256", Indexed: false, Name: "b"},
			{Type: "address", Indexed: true, Name: "c"},
			{Type: "uint256", Indexed: false, Name: "d"},
			{Type: "uint256", Indexed: true, Name: "e"},
		},
	}
	data := append(u256Topic(2), u256Topic(4)...)
	log, err := decodeEvent(def, [][]byte{
		placeholderTopic(),
		u256Topic(1),
		padAddr("3333333333333333333333333333333333333333"),
		u256Topic(5),
	}, data, "event.Decode")
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if len(log.Parameters) != len(def.Inputs) {
		t.Fatalf("len(Parameters) = %d, want %d", len(log.Parameters), len(def.Inputs))
	}
	wantUintParam(t, log, 0, "a", 1)
	wantUintParam(t, log, 1, "b", 2)
	wantAddressParam(t, log, 2, "c", "3333333333333333333333333333333333333333")
	wantUintParam(t, log, 3, "d", 4)
	wantUintParam(t, log, 4, "e", 5)
}

// The count invariant across shapes, including no inputs and inputs whose
// types fall through decodeTopicValue's default branch.
func TestDecodeEventParameterCountInvariant(t *testing.T) {
	cases := []struct {
		name     string
		def      *EventDef
		topics   [][]byte
		data     []byte
		wantLen  int
		wantVals []int64
	}{
		{
			name:    "no inputs",
			def:     &EventDef{Name: "NoInputs"},
			topics:  [][]byte{placeholderTopic()},
			wantLen: 0,
		},
		{
			name: "three unnamed indexed",
			def: &EventDef{Name: "T", Inputs: []ParamDef{
				{Type: "uint256", Indexed: true},
				{Type: "uint256", Indexed: true},
				{Type: "uint256", Indexed: true},
			}},
			topics:   [][]byte{placeholderTopic(), u256Topic(1), u256Topic(2), u256Topic(3)},
			wantLen:  3,
			wantVals: []int64{1, 2, 3},
		},
		{
			name: "four unnamed non-indexed",
			def: &EventDef{Name: "U", Inputs: []ParamDef{
				{Type: "uint256", Indexed: false},
				{Type: "uint256", Indexed: false},
				{Type: "uint256", Indexed: false},
				{Type: "uint256", Indexed: false},
			}},
			data:     append(append(append(u256Topic(1), u256Topic(2)...), u256Topic(3)...), u256Topic(4)...),
			wantLen:  4,
			wantVals: []int64{1, 2, 3, 4},
		},
		{
			name: "two indexed, two non-indexed, unnamed",
			def: &EventDef{Name: "W", Inputs: []ParamDef{
				{Type: "uint256", Indexed: false, Name: "x"},
				{Type: "uint256", Indexed: true},
				{Type: "uint256", Indexed: false, Name: "x"},
				{Type: "uint256", Indexed: true},
			}},
			topics:   [][]byte{placeholderTopic(), u256Topic(2), u256Topic(4)},
			data:     append(u256Topic(1), u256Topic(3)...),
			wantLen:  4,
			wantVals: []int64{1, 2, 3, 4},
		},
		{
			name: "unnamed hash types",
			def: &EventDef{Name: "V", Inputs: []ParamDef{
				{Type: "bytes32", Indexed: true},
				{Type: "bytes32", Indexed: true},
			}},
			topics:  [][]byte{placeholderTopic(), mustHex("0101010101010101010101010101010101010101010101010101010101010101"), mustHex("0202020202020202020202020202020202020202020202020202020202020202")},
			wantLen: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			log, err := decodeEvent(tc.def, tc.topics, tc.data, "event.Decode")
			if err != nil {
				t.Fatalf("decodeEvent: %v", err)
			}
			if len(log.Parameters) != tc.wantLen {
				t.Fatalf("len(Parameters) = %d, want %d", len(log.Parameters), tc.wantLen)
			}
			if len(log.Parameters) != len(tc.def.Inputs) {
				t.Fatalf("len(Parameters) = %d, len(Inputs) = %d, must be equal", len(log.Parameters), len(tc.def.Inputs))
			}
			for i, want := range tc.wantVals {
				wantUintParam(t, log, i, tc.def.Inputs[i].Name, want)
			}
		})
	}
}

// Unnamed indexed bytes32 inputs (no uint values involved) must also stay
// distinct: both topics decode through the same type branch, so a name-keyed
// merge returns the first digest twice.
func TestDecodeEventUnnamedBytes32IndexedDistinct(t *testing.T) {
	const digestA = "0101010101010101010101010101010101010101010101010101010101010101"
	const digestB = "0202020202020202020202020202020202020202020202020202020202020202"
	def := &EventDef{
		Name: "BytesPair",
		Inputs: []ParamDef{
			{Type: "bytes32", Indexed: true},
			{Type: "bytes32", Indexed: true},
		},
	}
	log, err := decodeEvent(def, [][]byte{placeholderTopic(), mustHex(digestA), mustHex(digestB)}, nil, "event.Decode")
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if len(log.Parameters) != len(def.Inputs) {
		t.Fatalf("len(Parameters) = %d, want %d", len(log.Parameters), len(def.Inputs))
	}
	for i, want := range []string{digestA, digestB} {
		got, ok := log.Parameters[i].Value.([]byte)
		if !ok || hex.EncodeToString(got) != want {
			t.Errorf("Parameters[%d].Value = %T (%v), want %s", i, log.Parameters[i].Value, log.Parameters[i].Value, want)
		}
	}
}

// End-to-end through the public path: unnamed inputs in real ABI JSON are
// registered without complaint and must decode positionally.
func TestDecodeUnnamedInputsFromABIJSON(t *testing.T) {
	const abiJSON = `[
		{"type":"event","name":"RegistryProbe","inputs":[
			{"type":"uint256","indexed":true},
			{"type":"uint256","indexed":true},
			{"type":"uint256","indexed":false}
		]}
	]`
	if err := RegisterABIJSON(abiJSON); err != nil {
		t.Fatalf("RegisterABIJSON: %v", err)
	}
	sig, ok := DecodeEventSignature(keccakTopic(t, "RegistryProbe(uint256,uint256,uint256)"))
	if !ok || sig != "RegistryProbe(uint256,uint256,uint256)" {
		t.Fatalf("registered signature = (%q, %v)", sig, ok)
	}

	log, err := Decode([][]byte{
		keccakTopic(t, "RegistryProbe(uint256,uint256,uint256)"),
		u256Topic(1),
		u256Topic(2),
	}, u256Topic(3))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if log.EventName != "RegistryProbe" {
		t.Fatalf("EventName = %q, want RegistryProbe", log.EventName)
	}
	if len(log.Parameters) != 3 {
		t.Fatalf("len(Parameters) = %d, want 3", len(log.Parameters))
	}
	wantUintParam(t, log, 0, "", 1)
	wantUintParam(t, log, 1, "", 2)
	wantUintParam(t, log, 2, "", 3)
}
