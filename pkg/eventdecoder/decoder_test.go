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
