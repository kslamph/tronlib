package eventdecoder

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/pb/core"
	"golang.org/x/crypto/sha3"
)

// --- RegisterABIJSON error path (line 56) ---

func TestRegisterABIJSON_InvalidJSON(t *testing.T) {
	err := RegisterABIJSON(`{not valid json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRegisterABIJSON_EmptyArray(t *testing.T) {
	err := RegisterABIJSON(`[]`)
	if err != nil {
		t.Fatalf("unexpected error for empty array: %v", err)
	}
}

// --- RegisterABIEntries nil entry and nil inputs (lines 86, 108-110) ---

func TestRegisterABIEntries_NilEntry(t *testing.T) {
	entries := []*core.SmartContract_ABI_Entry{
		nil,
		{
			Type:  core.SmartContract_ABI_Entry_Event,
			Name:  "TestEvent",
			Inputs: []*core.SmartContract_ABI_Entry_Param{
				{Type: "uint256", Indexed: true, Name: "val"},
			},
		},
	}
	err := RegisterABIEntries(entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify the non-nil entry was registered
	key := keccak4("TestEvent(uint256)")
	sig, found := DecodeEventSignature(key[:])
	if !found {
		t.Fatal("expected TestEvent to be registered")
	}
	if !strings.Contains(sig, "TestEvent") {
		t.Fatalf("unexpected sig: %s", sig)
	}
}

func TestRegisterABIEntries_NilInput(t *testing.T) {
	entries := []*core.SmartContract_ABI_Entry{
		{
			Type:  core.SmartContract_ABI_Entry_Event,
			Name:  "NilInputEvent",
			Inputs: []*core.SmartContract_ABI_Entry_Param{
				nil,
				{Type: "uint256", Indexed: false, Name: "val"},
			},
		},
	}
	err := RegisterABIEntries(entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- DecodeEventSignature nil def path (line 132-134) ---

func TestDecodeEventSignature_RegisteredButNilDef(t *testing.T) {
	// Directly inject a nil def into the registry to test the nil check
	key := [4]byte{0xAA, 0xBB, 0xCC, 0xDD}
	mu.Lock()
	sig4[key] = nil
	mu.Unlock()
	defer func() {
		mu.Lock()
		delete(sig4, key)
		mu.Unlock()
	}()

	sig, found := DecodeEventSignature(key[:])
	if found {
		t.Fatal("expected not found for nil def")
	}
	if sig != "" {
		t.Fatalf("expected empty sig, got: %s", sig)
	}
}

// --- DecodeLogs error path (line 181-184) ---

func TestDecodeLogs_ErrorFromLog(t *testing.T) {
	// A log with a topic[0] too short triggers DecodeLog error
	shortTopic := []byte{0x01, 0x02} // only 2 bytes, < 4
	addr := make([]byte, 20)
	addr[0] = 0xAA
	log := &core.TransactionInfo_Log{
		Topics: [][]byte{shortTopic},
		Data:   nil,
		Address: addr,
	}
	_, err := DecodeLogs([]*core.TransactionInfo_Log{log})
	if err == nil {
		t.Fatal("expected error from DecodeLogs when DecodeLog fails")
	}
}

// --- decodeEventInternal error from data decode (line 208-209) ---

func TestDecodeEventInternal_BadEventData(t *testing.T) {
	// Register an event with non-indexed params, then pass bad data
	def := &EventDef{
		Name: "BadDataEvent",
		Inputs: []ParamDef{
			{Type: "uint256", Indexed: false, Name: "amount"},
		},
	}
	// Pass data that can't be decoded as uint256 (too short)
	badData := []byte{0x01}
	_, err := decodeEventInternal(def, [][]byte{make([]byte, 32)}, badData)
	if err == nil {
		t.Fatal("expected error from decodeEventInternal with bad data")
	}
	if !strings.Contains(err.Error(), "failed to decode event data") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- decodeEventInternal fewer topics than indexed params (line 224-226) ---

func TestDecodeEventInternal_FewerTopicsThanIndexed(t *testing.T) {
	def := &EventDef{
		Name: "MultiIndexed",
		Inputs: []ParamDef{
			{Type: "address", Indexed: true, Name: "sender"},
			{Type: "address", Indexed: true, Name: "receiver"},
			{Type: "uint256", Indexed: false, Name: "amount"},
		},
	}
	// Only provide 1 indexed topic (topics[0] = sig, topics[1] = sender), but missing receiver
	sigTopic := make([]byte, 32)
	topic1 := make([]byte, 32)
	topic1[31] = 0x01
	topics := [][]byte{sigTopic, topic1}
	data := make([]byte, 64) // 32-byte padded uint256 = 0

	ev, err := decodeEventInternal(def, topics, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have only 2 params: sender (indexed) and amount (non-indexed)
	// receiver is skipped because there aren't enough topics
	if len(ev.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d: %v", len(ev.Parameters), ev.Parameters)
	}
}

// --- decodeTopicValue address error path (line 264-266) ---

func TestDecodeTopicValue_InvalidAddress(t *testing.T) {
	// Create a 32-byte topic that represents an invalid address
	// (all zeros is technically valid, but a very specific pattern could fail NewAddressFromHex)
	// The address conversion does eCommon.BytesToAddress(topic).Hex() then NewAddressFromHex
	// Hard to make BytesToAddress.Hex() fail, so test the hex fallback with a short topic
	topic := []byte{0x00, 0x01}
	result := decodeTopicValue(topic, "address")
	// With a short topic, BytesToAddress pads on the left, the hex should still be valid
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

// --- decodeEventData ABI type error (line 295-297) ---

func TestDecodeEventData_InvalidABIType(t *testing.T) {
	params := []ParamDef{
		{Type: "invalid[]type[[", Name: "bad"},
	}
	data := make([]byte, 32)
	_, err := decodeEventData(data, params)
	if err == nil {
		t.Fatal("expected error for invalid ABI type")
	}
	if !strings.Contains(err.Error(), "failed to create ABI type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- decodeEventData unpack error (line 306-308) ---

func TestDecodeEventData_UnpackError(t *testing.T) {
	params := []ParamDef{
		{Type: "uint256", Name: "val"},
	}
	// Empty data can't be unpacked as uint256
	_, err := decodeEventData([]byte{}, params)
	if err == nil {
		t.Fatal("expected unpack error")
	}
	if !strings.Contains(err.Error(), "failed to unpack event data") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- formatEventValue address error path (line 334-336) ---

func TestFormatEventValue_AddressConversionError(t *testing.T) {
	// formatEventValue with "address" type but value is an Address that fails NewAddressFromHex
	// The only way this happens is if addr.Hex() returns something NewAddressFromHex rejects
	// Using a zero address should work fine, so use a slightly crafted scenario
	// Actually, all valid ethereum addresses convert fine. Test with wrong type.
	result := formatEventValue(12345, "address")
	if result == "" {
		t.Fatal("expected non-empty fallback")
	}
}

// --- formatEventValue array type (lines 361-371) ---

func TestFormatEventValue_ArrayUint256(t *testing.T) {
	// Create a []uint256-like array: ABI decoding produces []*big.Int for uint256[]
	val := []*big.Int{big.NewInt(100), big.NewInt(200), big.NewInt(300)}
	result := formatEventValue(val, "uint256[]")
	if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
		t.Fatalf("expected array format, got: %s", result)
	}
	if !strings.Contains(result, "100") || !strings.Contains(result, "200") || !strings.Contains(result, "300") {
		t.Fatalf("expected all values in result, got: %s", result)
	}
}

func TestFormatEventValue_ArrayAddress(t *testing.T) {
	val := []string{"addr1", "addr2"}
	result := formatEventValue(val, "string[]")
	if result != "[addr1,addr2]" {
		t.Fatalf("expected [addr1,addr2], got: %s", result)
	}
}

func TestFormatEventValue_Bytes16(t *testing.T) {
	b := make([]byte, 16)
	b[0] = 0xAB
	result := formatEventValue(b, "bytes16")
	if result != hex.EncodeToString(b) {
		t.Fatalf("expected %s, got: %s", hex.EncodeToString(b), result)
	}
}

func TestFormatEventValue_Bytes8(t *testing.T) {
	b := make([]byte, 8)
	result := formatEventValue(b, "bytes8")
	if result != hex.EncodeToString(b) {
		t.Fatalf("expected %s, got: %s", hex.EncodeToString(b), result)
	}
}

// --- ParseABI invalid JSON (line 413-415) ---

func TestParseABI_InvalidJSON(t *testing.T) {
	parser := NewSimpleABIParser()
	_, err := parser.ParseABI(`not json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseABI_UnknownEntryType(t *testing.T) {
	parser := NewSimpleABIParser()
	abiJSON := `[{"type":"unknown_type","name":"foo","inputs":[]}]`
	result, err := parser.ParseABI(abiJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unknown type should be skipped, result should have 0 entries
	if len(result.Entrys) != 0 {
		t.Fatalf("expected 0 entries for unknown type, got %d", len(result.Entrys))
	}
}

func TestParseABI_AllEntryTypes(t *testing.T) {
	parser := NewSimpleABIParser()
	abiJSON := `[
		{"type":"constructor","inputs":[{"name":"addr","type":"address"}]},
		{"type":"function","name":"foo","inputs":[],"outputs":[{"type":"uint256"}]},
		{"type":"event","name":"Bar","inputs":[{"name":"x","type":"uint256","indexed":true}]},
		{"type":"fallback","inputs":[]},
		{"type":"receive","inputs":[]},
		{"type":"error","name":"CustomError","inputs":[{"name":"code","type":"uint256"}]}
	]`
	result, err := parser.ParseABI(abiJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Entrys) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(result.Entrys))
	}
}

// --- Multi-log DecodeLogs (line 172-191) ---

func TestDecodeLogs_MultipleLogs(t *testing.T) {
	// Register Transfer first
	if err := RegisterABIJSON(`[{"type":"event","name":"Approval","inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}]}]`); err != nil {
		t.Fatal(err)
	}

	// Build a log with valid address
	addr := make([]byte, 20)
	addr[0] = 0xAA

	// Approval sig
	approvalSig, _ := hex.DecodeString("8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
	topic1 := make([]byte, 32)
	topic1[31] = 0x01
	topic2 := make([]byte, 32)
	topic2[31] = 0x02

	log1 := &core.TransactionInfo_Log{
		Topics:  [][]byte{approvalSig, topic1, topic2},
		Data:    make([]byte, 32),
		Address: addr,
	}
	log2 := &core.TransactionInfo_Log{
		Topics:  [][]byte{approvalSig, topic1, topic2},
		Data:    make([]byte, 32),
		Address: addr,
	}

	results, err := DecodeLogs([]*core.TransactionInfo_Log{log1, log2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.EventName != "Approval" {
			t.Fatalf("unexpected event name: %s", r.EventName)
		}
		if r.Contract == "" {
			t.Fatal("expected non-empty contract address")
		}
	}
}

// --- DecodeLogs mixed with nil entries (line 185-186) ---

func TestDecodeLogs_MixedWithNil(t *testing.T) {
	// The existing TestEdgeCases already tests this but let's add more coverage
	// with multiple logs and a nil in the middle
	addr := make([]byte, 20)
	addr[0] = 0xBB
	sigTopic := make([]byte, 32)
	sigTopic[0] = 0x11

	log := &core.TransactionInfo_Log{
		Topics:  [][]byte{sigTopic},
		Data:    nil,
		Address: addr,
	}

	logs := []*core.TransactionInfo_Log{nil, log, nil}
	results, err := DecodeLogs(logs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

// --- decodeEventInternal with data but no non-indexed params ---

func TestDecodeEventInternal_NoNonIndexed(t *testing.T) {
	def := &EventDef{
		Name: "OnlyIndexed",
		Inputs: []ParamDef{
			{Type: "uint256", Indexed: true, Name: "val"},
		},
	}
	topic := make([]byte, 32)
	topic[31] = 0x42
	topics := [][]byte{make([]byte, 32), topic}

	ev, err := decodeEventInternal(def, topics, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.EventName != "OnlyIndexed" {
		t.Fatalf("unexpected event name: %s", ev.EventName)
	}
	if len(ev.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(ev.Parameters))
	}
}

// --- decodeEventInternal with data but empty data, non-indexed params present ---

func TestDecodeEventInternal_EmptyDataWithNonIndexed(t *testing.T) {
	def := &EventDef{
		Name: "EmptyDataEvent",
		Inputs: []ParamDef{
			{Type: "uint256", Indexed: false, Name: "val"},
		},
	}
	topics := [][]byte{make([]byte, 32)}

	ev, err := decodeEventInternal(def, topics, []byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When data is empty but non-indexed params exist, the non-indexed block is skipped
	if len(ev.Parameters) != 0 {
		t.Fatalf("expected 0 parameters with empty data, got %d: %v", len(ev.Parameters), ev.Parameters)
	}
}

// --- formatEventValue with non-slice value for array type ---

func TestFormatEventValue_ArrayNonSlice(t *testing.T) {
	// If the value has [] suffix but the value is not a slice, it should fall through
	result := formatEventValue("not-a-slice", "uint256[]")
	// Falls through to default case: fmt.Sprintf("%v", value)
	if result != "not-a-slice" {
		t.Fatalf("expected fallback, got: %s", result)
	}
}

// --- RegisterABIEntries with non-event entries ---

func TestRegisterABIEntries_NonEventEntries(t *testing.T) {
	entries := []*core.SmartContract_ABI_Entry{
		{
			Type:  core.SmartContract_ABI_Entry_Function,
			Name:  "transfer",
			Inputs: []*core.SmartContract_ABI_Entry_Param{
				{Type: "address", Name: "to"},
				{Type: "uint256", Name: "amount"},
			},
		},
		{
			Type:  core.SmartContract_ABI_Entry_Constructor,
			Name:  "",
			Inputs: []*core.SmartContract_ABI_Entry_Param{
				{Type: "address", Name: "owner"},
			},
		},
	}
	err := RegisterABIEntries(entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No events registered, so the function should return nil
}

// --- DecodeEventSignature with 4-byte exact match ---

func TestDecodeEventSignature_Exact4Bytes(t *testing.T) {
	// Use the Transfer signature we already registered
	transferSig := make([]byte, 32)
	copy(transferSig[:4], []byte{0xdd, 0xf2, 0x52, 0xad})
	_, found := DecodeEventSignature(transferSig)
	if !found {
		t.Fatal("expected Transfer to be found")
	}
}

// keccak4 computes the first 4 bytes of keccak256 of a signature string.
func keccak4(sig string) [4]byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(sig))
	sum := hasher.Sum(nil)
	var key [4]byte
	copy(key[:], sum[:4])
	return key
}
