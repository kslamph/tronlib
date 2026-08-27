package eventdecoder

import (
	"encoding/hex"
	"testing"

	"github.com/kslamph/tronlib/pkg/trc20"
)

// TestBug2_ParseABINoLongerStub verifies that ParseABI now correctly parses
// JSON ABI strings and returns the expected entries (was previously a no-op stub).
func TestBug2_ParseABINoLongerStub(t *testing.T) {
	parser := NewSimpleABIParser()

	result, err := parser.ParseABI(trc20.ERC20ABI)
	if err != nil {
		t.Fatalf("ParseABI returned unexpected error: %v", err)
	}

	if len(result.Entrys) == 0 {
		t.Fatal("ParseABI should return entries for valid ABI JSON, got 0")
	}

	t.Logf("FIX VERIFIED: ParseABI returned %d entries for ERC20 ABI JSON", len(result.Entrys))

	// Verify specific entries exist
	foundTransfer := false
	foundDecimals := false
	for _, entry := range result.Entrys {
		if entry.Name == "Transfer" && entry.Type == 3 { // Event type
			foundTransfer = true
		}
		if entry.Name == "decimals" && entry.Type == 2 { // Function type
			foundDecimals = true
		}
	}
	if !foundTransfer {
		t.Error("Expected to find Transfer event entry")
	}
	if !foundDecimals {
		t.Error("Expected to find decimals function entry")
	}
}

// TestBug2_RegisterABIJSONNowWorksForCustomEvents verifies that RegisterABIJSON
// now correctly registers custom (non-builtin) events.
func TestBug2_RegisterABIJSONNowWorksForCustomEvents(t *testing.T) {
	customABI := `[
	  {
	    "anonymous": false,
	    "inputs": [
	      {"indexed": true, "name": "user", "type": "address"},
	      {"indexed": false, "name": "amount", "type": "uint256"},
	      {"indexed": false, "name": "reason", "type": "string"}
	    ],
	    "name": "CustomSlashed",
	    "type": "event"
	  }
	]`

	err := RegisterABIJSON(customABI)
	if err != nil {
		t.Fatalf("RegisterABIJSON returned error: %v", err)
	}

	// keccak256("CustomSlashed(address,uint256,string)") = 0x1e13a438...
	sigBytes, _ := hex.DecodeString("1e13a438ab0f8297e5303d8f40d2cd5da80e9e015173189d4ef72fe0f660e788")

	sig, found := DecodeEventSignature(sigBytes)
	if !found {
		t.Fatal("FIX VERIFIED: CustomSlashed event should now be registered via RegisterABIJSON")
	}
	if sig != "CustomSlashed(address,uint256,string)" {
		t.Fatalf("unexpected signature: %s", sig)
	}
	t.Logf("FIX VERIFIED: Custom event registered: %s", sig)
}

// TestBug2_BuiltinEventsStillWork shows that builtin events remain functional.
func TestBug2_BuiltinEventsStillWork(t *testing.T) {
	transferSig, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	sig, found := DecodeEventSignature(transferSig)
	if !found {
		t.Fatal("Transfer should be found in builtin registry")
	}
	t.Logf("Builtin events work fine: %s", sig)
}
