package eventdecoder

import (
	"encoding/hex"
	"testing"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/trc20"
)

func TestRegisterAndDecodeTRC20(t *testing.T) {
	// register ERC20 ABI
	if err := RegisterABIJSON(trc20.ERC20ABI); err != nil {
		t.Fatalf("register ABI: %v", err)
	}

	// Build a synthetic Transfer event
	// keccak("Transfer(address,address,uint256)") first topic
	sigTopic, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	fromTopic, _ := hex.DecodeString("000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	toTopic, _ := hex.DecodeString("0000000000000000000000004e83362442b8d1bec281594cea3050c8eb01311c")
	amountData, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000003e8") // 1000

	topics := [][]byte{sigTopic, fromTopic, toTopic}
	data := amountData

	ev, err := DecodeLog(topics, data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ev.EventName != "Transfer" {
		t.Fatalf("unexpected event name: %s", ev.EventName)
	}
	if len(ev.Parameters) != 3 {
		t.Fatalf("unexpected param count: %d", len(ev.Parameters))
	}
}

func TestDecodeUnknownSignature(t *testing.T) {
	// unknown 4-byte
	sigTopic, _ := hex.DecodeString("1234567800000000000000000000000000000000000000000000000000000000")
	ev, err := DecodeLog([][]byte{sigTopic}, nil)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ev.EventName == "" {
		t.Fatalf("expected unknown event name, got empty")
	}
}

func TestDecodeLogsBatch(t *testing.T) {
	// minimal sanity: use empty logs
	res, err := DecodeLogs([]*core.TransactionInfo_Log{})
	if err != nil {
		t.Fatalf("decode logs: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected empty result")
	}
}

// Test all public APIs of the eventdecoder package
func TestPublicAPIs(t *testing.T) {
	// Test RegisterABIObject with empty ABI (should not fail)
	abiObj := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{},
	}
	err := RegisterABIObject(abiObj)
	if err != nil {
		t.Fatalf("RegisterABIObject failed: %v", err)
	}

	// Test RegisterABIEntries with empty entries
	err = RegisterABIEntries([]*core.SmartContract_ABI_Entry{})
	if err != nil {
		t.Fatalf("RegisterABIEntries should not fail with empty entries: %v", err)
	}

	// Test DecodeEventSignature with TRC20 Transfer signature
	sigTopic, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef") // Transfer signature
	sigStr, found := DecodeEventSignature(sigTopic)
	if !found {
		t.Fatalf("DecodeEventSignature should find TRC20 Transfer signature")
	}
	if sigStr != "Transfer(address,address,uint256)" {
		t.Fatalf("unexpected signature string: %s", sigStr)
	}

	// Test DecodeLogs with a mock log (empty for now)
	logs := []*core.TransactionInfo_Log{}
	results, err := DecodeLogs(logs)
	if err != nil {
		t.Fatalf("DecodeLogs failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

// Test all supported decoded types
func TestDecodedTypes(t *testing.T) {
	// Test a simple builtin event - use Transfer event which we know works
	// Signature: 0xddf252ad (Transfer(address,address,uint256))
	sigTopic, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	// Test that the signature is registered
	sigStr, found := DecodeEventSignature(sigTopic)
	if !found {
		t.Fatalf("Transfer signature not found in builtin registry")
	}
	expectedSig := "Transfer(address,address,uint256)"
	if sigStr != expectedSig {
		t.Fatalf("Unexpected signature string: %s, expected: %s", sigStr, expectedSig)
	}

	// Test with actual TRC20 Transfer data
	fromTopic, _ := hex.DecodeString("000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	toTopic, _ := hex.DecodeString("0000000000000000000000004e83362442b8d1bec281594cea3050c8eb01311c")
	amountData, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000003e8") // 1000

	topics := [][]byte{sigTopic, fromTopic, toTopic}
	data := amountData

	ev, err := DecodeLog(topics, data)
	if err != nil {
		t.Fatalf("DecodeLog failed: %v", err)
	}
	if ev.EventName != "Transfer" {
		t.Fatalf("unexpected event name: %s", ev.EventName)
	}
	if len(ev.Parameters) != 3 {
		t.Fatalf("unexpected param count: %d", len(ev.Parameters))
	}

	// Check each parameter
	params := make(map[string]DecodedEventParameter)
	for _, param := range ev.Parameters {
		params[param.Name] = param
	}

	// Check from parameter
	if param, ok := params["from"]; ok {
		if param.Type != "address" {
			t.Errorf("from type mismatch: %s", param.Type)
		}
		if param.Indexed != true {
			t.Errorf("from indexed mismatch: %v", param.Indexed)
		}
		// Should be decoded as TRON address
		if param.Value == "" {
			t.Errorf("from value is empty")
		}
	} else {
		t.Error("from not found")
	}

	// Check to parameter
	if param, ok := params["to"]; ok {
		if param.Type != "address" {
			t.Errorf("to type mismatch: %s", param.Type)
		}
		if param.Indexed != true {
			t.Errorf("to indexed mismatch: %v", param.Indexed)
		}
		// Should be decoded as TRON address
		if param.Value == "" {
			t.Errorf("to value is empty")
		}
	} else {
		t.Error("to not found")
	}

	// Check value parameter
	if param, ok := params["value"]; ok {
		if param.Type != "uint256" {
			t.Errorf("value type mismatch: %s", param.Type)
		}
		if param.Indexed != false {
			t.Errorf("value indexed mismatch: %v", param.Indexed)
		}
		if param.Value != "1000" {
			t.Errorf("value value mismatch: %s", param.Value)
		}
	} else {
		t.Error("value not found")
	}
}

// Test edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	// Test DecodeLog with no topics
	_, err := DecodeLog([][]byte{}, nil)
	if err == nil {
		t.Error("DecodeLog should fail with no topics")
	}

	// Test DecodeLog with short topic
	_, err = DecodeLog([][]byte{{0x01, 0x02}}, nil)
	if err == nil {
		t.Error("DecodeLog should fail with short topic")
	}

	// Test DecodeEventSignature with short signature
	_, found := DecodeEventSignature([]byte{0x01, 0x02})
	if found {
		t.Error("DecodeEventSignature should not find short signature")
	}

	// Test RegisterABIObject with nil ABI
	err = RegisterABIObject(nil)
	if err == nil {
		t.Error("RegisterABIObject should fail with nil ABI")
	}

	// Test RegisterABIEntries with empty entries
	err = RegisterABIEntries([]*core.SmartContract_ABI_Entry{})
	if err != nil {
		t.Errorf("RegisterABIEntries should not fail with empty entries: %v", err)
	}

	// Test DecodeLogs with nil log entry
	logs := []*core.TransactionInfo_Log{nil}
	_, err = DecodeLogs(logs)
	if err != nil {
		t.Errorf("DecodeLogs should handle nil log entries: %v", err)
	}
}
