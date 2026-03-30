package eventdecoder

import (
	"encoding/hex"
	"math/big"
	"testing"

	eCommon "github.com/ethereum/go-ethereum/common"
)

func TestDecodeTopicValue(t *testing.T) {
	t.Run("address", func(t *testing.T) {
		// 32-byte padded Ethereum address
		topic, _ := hex.DecodeString("000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		result := decodeTopicValue(topic, "address")
		// Should return a TRON address (starts with T)
		if result == "" {
			t.Fatal("expected non-empty result")
		}
		if result[0] != 'T' {
			t.Fatalf("expected TRON address starting with T, got: %s", result)
		}
	})

	t.Run("uint256", func(t *testing.T) {
		// 1000 in 32-byte big-endian
		topic, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000003e8")
		result := decodeTopicValue(topic, "uint256")
		if result != "1000" {
			t.Fatalf("expected 1000, got: %s", result)
		}
	})

	t.Run("uint8", func(t *testing.T) {
		topic, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000006")
		result := decodeTopicValue(topic, "uint8")
		if result != "6" {
			t.Fatalf("expected 6, got: %s", result)
		}
	})

	t.Run("int256", func(t *testing.T) {
		topic, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000003e8")
		result := decodeTopicValue(topic, "int256")
		if result != "1000" {
			t.Fatalf("expected 1000, got: %s", result)
		}
	})

	t.Run("bool true", func(t *testing.T) {
		topic, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
		result := decodeTopicValue(topic, "bool")
		if result != "true" {
			t.Fatalf("expected true, got: %s", result)
		}
	})

	t.Run("bool false", func(t *testing.T) {
		topic, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
		result := decodeTopicValue(topic, "bool")
		if result != "false" {
			t.Fatalf("expected false, got: %s", result)
		}
	})

	t.Run("bytes32", func(t *testing.T) {
		topic, _ := hex.DecodeString("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
		result := decodeTopicValue(topic, "bytes32")
		expected := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
		if result != expected {
			t.Fatalf("expected %s, got: %s", expected, result)
		}
	})

	t.Run("unknown type falls back to hex", func(t *testing.T) {
		topic := []byte{0xde, 0xad, 0xbe, 0xef}
		result := decodeTopicValue(topic, "customType")
		if result != "deadbeef" {
			t.Fatalf("expected deadbeef, got: %s", result)
		}
	})

	t.Run("large uint256", func(t *testing.T) {
		// 2^256 - 1 (max uint256)
		topic := make([]byte, 32)
		for i := range topic {
			topic[i] = 0xff
		}
		result := decodeTopicValue(topic, "uint256")
		expected := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1)).String()
		if result != expected {
			t.Fatalf("expected %s, got: %s", expected, result)
		}
	})
}

func TestFormatEventValue(t *testing.T) {
	t.Run("address", func(t *testing.T) {
		addr := eCommon.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		result := formatEventValue(addr, "address")
		if result == "" {
			t.Fatal("expected non-empty result")
		}
		if result[0] != 'T' {
			t.Fatalf("expected TRON address starting with T, got: %s", result)
		}
	})

	t.Run("uint256", func(t *testing.T) {
		val := big.NewInt(1000000)
		result := formatEventValue(val, "uint256")
		if result != "1000000" {
			t.Fatalf("expected 1000000, got: %s", result)
		}
	})

	t.Run("uint8", func(t *testing.T) {
		val := big.NewInt(42)
		result := formatEventValue(val, "uint8")
		if result != "42" {
			t.Fatalf("expected 42, got: %s", result)
		}
	})

	t.Run("int256", func(t *testing.T) {
		val := big.NewInt(-100)
		result := formatEventValue(val, "int256")
		if result != "-100" {
			t.Fatalf("expected -100, got: %s", result)
		}
	})

	t.Run("bytes", func(t *testing.T) {
		val := []byte{0xde, 0xad, 0xbe, 0xef}
		result := formatEventValue(val, "bytes")
		if result != "deadbeef" {
			t.Fatalf("expected deadbeef, got: %s", result)
		}
	})

	t.Run("bytes32", func(t *testing.T) {
		val := make([]byte, 32)
		for i := range val {
			val[i] = byte(i)
		}
		result := formatEventValue(val, "bytes32")
		expected := hex.EncodeToString(val)
		if result != expected {
			t.Fatalf("expected %s, got: %s", expected, result)
		}
	})

	t.Run("string", func(t *testing.T) {
		result := formatEventValue("hello world", "string")
		if result != "hello world" {
			t.Fatalf("expected hello world, got: %s", result)
		}
	})

	t.Run("bool true", func(t *testing.T) {
		result := formatEventValue(true, "bool")
		if result != "true" {
			t.Fatalf("expected true, got: %s", result)
		}
	})

	t.Run("bool false", func(t *testing.T) {
		result := formatEventValue(false, "bool")
		if result != "false" {
			t.Fatalf("expected false, got: %s", result)
		}
	})

	t.Run("unknown type uses fmt.Sprint", func(t *testing.T) {
		result := formatEventValue(42, "unknown")
		if result != "42" {
			t.Fatalf("expected 42, got: %s", result)
		}
	})

	t.Run("uint128", func(t *testing.T) {
		val := big.NewInt(999999)
		result := formatEventValue(val, "uint128")
		if result != "999999" {
			t.Fatalf("expected 999999, got: %s", result)
		}
	})

	t.Run("uint64", func(t *testing.T) {
		val := big.NewInt(12345)
		result := formatEventValue(val, "uint64")
		if result != "12345" {
			t.Fatalf("expected 12345, got: %s", result)
		}
	})

	t.Run("address wrong type fallback", func(t *testing.T) {
		// Passing a non-address value for address type
		result := formatEventValue("not an address", "address")
		if result != "not an address" {
			t.Fatalf("expected fallback, got: %s", result)
		}
	})

	t.Run("uint256 wrong type fallback", func(t *testing.T) {
		result := formatEventValue("not a big int", "uint256")
		if result != "not a big int" {
			t.Fatalf("expected fallback, got: %s", result)
		}
	})

	t.Run("bytes wrong type fallback", func(t *testing.T) {
		result := formatEventValue("not bytes", "bytes")
		if result != "not bytes" {
			t.Fatalf("expected fallback, got: %s", result)
		}
	})

	t.Run("bool wrong type fallback", func(t *testing.T) {
		result := formatEventValue("not bool", "bool")
		if result != "not bool" {
			t.Fatalf("expected fallback, got: %s", result)
		}
	})
}
