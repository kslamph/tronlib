package utils

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── transaction.go: SetPermissionID ──────────────────────────────────────────

func TestSetPermissionID(t *testing.T) {
	minimalTx := func() *core.Transaction {
		return &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{{}},
			},
		}
	}
	minimalExt := func() *api.TransactionExtention {
		return &api.TransactionExtention{
			Transaction: minimalTx(),
		}
	}

	t.Run("nil", func(t *testing.T) {
		assert.Error(t, SetPermissionID(nil, 1))
	})
	t.Run("core.NilRawData", func(t *testing.T) {
		assert.Error(t, SetPermissionID(&core.Transaction{}, 1))
	})
	t.Run("core.EmptyContract", func(t *testing.T) {
		assert.Error(t, SetPermissionID(&core.Transaction{RawData: &core.TransactionRaw{}}, 1))
	})
	t.Run("core.Success", func(t *testing.T) {
		tx := minimalTx()
		require.NoError(t, SetPermissionID(tx, 3))
		assert.Equal(t, int32(3), tx.RawData.Contract[0].PermissionId)
	})
	t.Run("ext.NilTransaction", func(t *testing.T) {
		assert.Error(t, SetPermissionID(&api.TransactionExtention{}, 1))
	})
	t.Run("ext.NilRawData", func(t *testing.T) {
		assert.Error(t, SetPermissionID(&api.TransactionExtention{Transaction: &core.Transaction{}}, 1))
	})
	t.Run("ext.EmptyContract", func(t *testing.T) {
		assert.Error(t, SetPermissionID(&api.TransactionExtention{Transaction: &core.Transaction{RawData: &core.TransactionRaw{}}}, 1))
	})
	t.Run("ext.Success", func(t *testing.T) {
		ext := minimalExt()
		require.NoError(t, SetPermissionID(ext, 5))
		assert.Equal(t, int32(5), ext.Transaction.RawData.Contract[0].PermissionId)
	})
	t.Run("WrongType", func(t *testing.T) {
		assert.Error(t, SetPermissionID("bad", 1))
	})
}

// ── transaction.go: SetFeeLimit ──────────────────────────────────────────────

func TestSetFeeLimit(t *testing.T) {
	minimalTx := func() *core.Transaction {
		return &core.Transaction{RawData: &core.TransactionRaw{}}
	}
	minimalExt := func() *api.TransactionExtention {
		return &api.TransactionExtention{Transaction: minimalTx()}
	}

	t.Run("nil", func(t *testing.T) {
		assert.Error(t, SetFeeLimit(nil, 100))
	})
	t.Run("core.NilRawData", func(t *testing.T) {
		assert.Error(t, SetFeeLimit(&core.Transaction{}, 100))
	})
	t.Run("core.Success", func(t *testing.T) {
		tx := minimalTx()
		require.NoError(t, SetFeeLimit(tx, 150_000_000))
		assert.Equal(t, int64(150_000_000), tx.RawData.FeeLimit)
	})
	t.Run("ext.NilTransaction", func(t *testing.T) {
		assert.Error(t, SetFeeLimit(&api.TransactionExtention{}, 100))
	})
	t.Run("ext.NilRawData", func(t *testing.T) {
		assert.Error(t, SetFeeLimit(&api.TransactionExtention{Transaction: &core.Transaction{}}, 100))
	})
	t.Run("ext.Success", func(t *testing.T) {
		ext := minimalExt()
		require.NoError(t, SetFeeLimit(ext, 200_000_000))
		assert.Equal(t, int64(200_000_000), ext.Transaction.RawData.FeeLimit)
	})
	t.Run("WrongType", func(t *testing.T) {
		assert.Error(t, SetFeeLimit(12345, 100))
	})
}

// ── transaction.go: SetTimestamp ─────────────────────────────────────────────

func TestSetTimestamp(t *testing.T) {
	minimalTx := func() *core.Transaction {
		return &core.Transaction{RawData: &core.TransactionRaw{}}
	}
	minimalExt := func() *api.TransactionExtention {
		return &api.TransactionExtention{Transaction: minimalTx()}
	}

	t.Run("nil", func(t *testing.T) {
		assert.Error(t, SetTimestamp(nil, 100))
	})
	t.Run("core.NilRawData", func(t *testing.T) {
		assert.Error(t, SetTimestamp(&core.Transaction{}, 100))
	})
	t.Run("core.Success", func(t *testing.T) {
		tx := minimalTx()
		require.NoError(t, SetTimestamp(tx, 1700000000000))
		assert.Equal(t, int64(1700000000000), tx.RawData.Timestamp)
	})
	t.Run("ext.NilTransaction", func(t *testing.T) {
		assert.Error(t, SetTimestamp(&api.TransactionExtention{}, 100))
	})
	t.Run("ext.NilRawData", func(t *testing.T) {
		assert.Error(t, SetTimestamp(&api.TransactionExtention{Transaction: &core.Transaction{}}, 100))
	})
	t.Run("ext.Success", func(t *testing.T) {
		ext := minimalExt()
		require.NoError(t, SetTimestamp(ext, 1700000000001))
		assert.Equal(t, int64(1700000000001), ext.Transaction.RawData.Timestamp)
	})
	t.Run("WrongType", func(t *testing.T) {
		assert.Error(t, SetTimestamp([]byte{1, 2}, 100))
	})
}

// ── transaction.go: SetExpiration ────────────────────────────────────────────

func TestSetExpiration(t *testing.T) {
	minimalTx := func() *core.Transaction {
		return &core.Transaction{RawData: &core.TransactionRaw{}}
	}
	minimalExt := func() *api.TransactionExtention {
		return &api.TransactionExtention{Transaction: minimalTx()}
	}

	t.Run("nil", func(t *testing.T) {
		assert.Error(t, SetExpiration(nil, 100))
	})
	t.Run("core.NilRawData", func(t *testing.T) {
		assert.Error(t, SetExpiration(&core.Transaction{}, 100))
	})
	t.Run("core.Success", func(t *testing.T) {
		tx := minimalTx()
		require.NoError(t, SetExpiration(tx, 1700000060000))
		assert.Equal(t, int64(1700000060000), tx.RawData.Expiration)
	})
	t.Run("ext.NilTransaction", func(t *testing.T) {
		assert.Error(t, SetExpiration(&api.TransactionExtention{}, 100))
	})
	t.Run("ext.NilRawData", func(t *testing.T) {
		assert.Error(t, SetExpiration(&api.TransactionExtention{Transaction: &core.Transaction{}}, 100))
	})
	t.Run("ext.Success", func(t *testing.T) {
		ext := minimalExt()
		require.NoError(t, SetExpiration(ext, 1700000060001))
		assert.Equal(t, int64(1700000060001), ext.Transaction.RawData.Expiration)
	})
	t.Run("WrongType", func(t *testing.T) {
		assert.Error(t, SetExpiration(42.0, 100))
	})
}

// ── transaction.go: ExtractSigners ───────────────────────────────────────────

func TestExtractSigners(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		signers, err := ExtractSigners(nil)
		assert.NoError(t, err)
		assert.Nil(t, signers)
	})
	t.Run("NoSignatures", func(t *testing.T) {
		tx := &core.Transaction{
			RawData:   &core.TransactionRaw{},
			Signature: [][]byte{},
		}
		signers, err := ExtractSigners(tx)
		assert.Error(t, err)
		assert.Nil(t, signers)
	})
	t.Run("EmptySignature", func(t *testing.T) {
		tx := &core.Transaction{
			RawData:   &core.TransactionRaw{},
			Signature: [][]byte{[]byte("short")},
		}
		// Short sig (<64 bytes) is skipped; if all skipped, returns no signers
		signers, err := ExtractSigners(tx)
		assert.NoError(t, err)
		assert.Empty(t, signers)
	})
}

// ── encoding.go: EncodeParameters / DecodeParameters ─────────────────────────

const testTransferABI = `[
  {
    "name": "transfer",
    "type": "function",
    "inputs": [
      {"name": "to", "type": "address"},
      {"name": "amount", "type": "uint256"}
    ],
    "outputs": [{"name": "", "type": "bool"}],
    "stateMutability": "nonpayable"
  },
  {
    "name": "totalSupply",
    "type": "function",
    "inputs": [],
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view"
  },
  {
    "name": "balanceOf",
    "type": "function",
    "inputs": [{"name": "owner", "type": "address"}],
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view"
  }
]`

func TestEncodeParameters(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		addr := common.HexToAddress("0x412d53ffb890dd6c173c9d495cb8a4806535c1f2be")
		data, err := EncodeParameters(testTransferABI, "transfer", addr, big.NewInt(100))
		assert.NoError(t, err)
		assert.Len(t, data, 4+64) // 4-byte sig + 2x32-byte params
	})
	t.Run("InvalidABI", func(t *testing.T) {
		_, err := EncodeParameters("{bad json}", "transfer", "addr")
		assert.Error(t, err)
	})
	t.Run("MethodNotFound", func(t *testing.T) {
		_, err := EncodeParameters(testTransferABI, "nonexistent", "addr")
		assert.Error(t, err)
	})
	t.Run("WrongParamCount", func(t *testing.T) {
		_, err := EncodeParameters(testTransferABI, "transfer")
		assert.Error(t, err)
	})
}

func TestDecodeParameters(t *testing.T) {
	t.Run("InvalidABI", func(t *testing.T) {
		_, err := DecodeParameters("{bad}", "totalSupply", []byte{})
		assert.Error(t, err)
	})
	t.Run("MethodNotFound", func(t *testing.T) {
		_, err := DecodeParameters(testTransferABI, "nonexistent", []byte{})
		assert.Error(t, err)
	})
	t.Run("InvalidData", func(t *testing.T) {
		_, err := DecodeParameters(testTransferABI, "totalSupply", []byte{0x01})
		assert.Error(t, err)
	})
}

// ── encoding.go: DecodeString ────────────────────────────────────────────────

// padUint256 returns a 32-byte big-endian representation of v.
func padUint256(v int64) []byte {
	b := make([]byte, 32)
	vb := big.NewInt(v).Bytes()
	copy(b[32-len(vb):], vb)
	return b
}

func TestDecodeString_EdgeCases(t *testing.T) {
	t.Run("TooShort", func(t *testing.T) {
		_, err := DecodeString([]byte{0, 0, 0})
		assert.Error(t, err)
	})
	t.Run("BadOffset", func(t *testing.T) {
		data := make([]byte, 96)
		_, err := DecodeString(data)
		assert.Error(t, err)
	})
	t.Run("NegativeLength", func(t *testing.T) {
		data := make([]byte, 96)
		copy(data[:32], padUint256(32))
		s, err := DecodeString(data)
		assert.NoError(t, err)
		assert.Equal(t, "", s)
	})
	t.Run("LengthExceedsData", func(t *testing.T) {
		data := make([]byte, 96)
		copy(data[:32], padUint256(32))
		copy(data[32:64], padUint256(100))
		_, err := DecodeString(data)
		assert.Error(t, err)
	})
	t.Run("Valid", func(t *testing.T) {
		str := "hello"
		data := make([]byte, 64+len(str))
		copy(data[:32], padUint256(32))
		copy(data[32:64], padUint256(int64(len(str))))
		copy(data[64:], str)
		s, err := DecodeString(data)
		assert.NoError(t, err)
		assert.Equal(t, "hello", s)
	})
}

// ── validation.go: IsValidContractName / ValidateContractName ────────────────

func TestContractNameValidation(t *testing.T) {
	t.Run("EmptyIsValid", func(t *testing.T) {
		assert.True(t, IsValidContractName(""))
	})
	t.Run("NormalIsValid", func(t *testing.T) {
		assert.True(t, IsValidContractName("MyToken"))
	})
	t.Run("ControlCharInvalid", func(t *testing.T) {
		assert.False(t, IsValidContractName("bad\x00name"))
	})
	t.Run("DELInvalid", func(t *testing.T) {
		assert.False(t, IsValidContractName("bad\x7fname"))
	})
	t.Run("ValidateValid", func(t *testing.T) {
		assert.NoError(t, ValidateContractName("Token"))
	})
	t.Run("ValidateInvalid", func(t *testing.T) {
		assert.Error(t, ValidateContractName("bad\x01"))
	})
}

// ── validation.go: ValidateConsumeUserResourcePercent ────────────────────────

func TestValidateConsumeUserResourcePercent(t *testing.T) {
	tests := []struct {
		name    string
		pct     int64
		wantErr bool
	}{
		{"zero", 0, false},
		{"fifty", 50, false},
		{"hundred", 100, false},
		{"negative", -1, true},
		{"over100", 101, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConsumeUserResourcePercent(tt.pct)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ── abi_parse.go: parseABIEntry more types ───────────────────────────────────

func TestParseABIEntryMoreTypes(t *testing.T) {
	proc := NewABIProcessor(nil)

	t.Run("fallback", func(t *testing.T) {
		entry, err := proc.parseABIEntry(map[string]interface{}{
			"type":    "fallback",
			"payable": true,
		})
		require.NoError(t, err)
		assert.Equal(t, core.SmartContract_ABI_Entry_Fallback, entry.Type)
		assert.True(t, entry.Payable)
	})
	t.Run("unknownType", func(t *testing.T) {
		entry, err := proc.parseABIEntry(map[string]interface{}{
			"type": "somethingweird",
		})
		require.NoError(t, err)
		assert.Equal(t, core.SmartContract_ABI_Entry_UnknownEntryType, entry.Type)
	})
	t.Run("stateMutabilityPure", func(t *testing.T) {
		entry, err := proc.parseABIEntry(map[string]interface{}{
			"stateMutability": "pure",
		})
		require.NoError(t, err)
		assert.Equal(t, core.SmartContract_ABI_Entry_Pure, entry.StateMutability)
	})
	t.Run("stateMutabilityView", func(t *testing.T) {
		entry, err := proc.parseABIEntry(map[string]interface{}{
			"stateMutability": "view",
		})
		require.NoError(t, err)
		assert.Equal(t, core.SmartContract_ABI_Entry_View, entry.StateMutability)
	})
	t.Run("stateMutabilityNonpayable", func(t *testing.T) {
		entry, err := proc.parseABIEntry(map[string]interface{}{
			"stateMutability": "nonpayable",
		})
		require.NoError(t, err)
		assert.Equal(t, core.SmartContract_ABI_Entry_Nonpayable, entry.StateMutability)
	})
	t.Run("stateMutabilityPayable", func(t *testing.T) {
		entry, err := proc.parseABIEntry(map[string]interface{}{
			"stateMutability": "payable",
		})
		require.NoError(t, err)
		assert.Equal(t, core.SmartContract_ABI_Entry_Payable, entry.StateMutability)
	})
	t.Run("unknownMutability", func(t *testing.T) {
		entry, err := proc.parseABIEntry(map[string]interface{}{
			"stateMutability": "something",
		})
		require.NoError(t, err)
		assert.Equal(t, core.SmartContract_ABI_Entry_UnknownMutabilityType, entry.StateMutability)
	})
	t.Run("constantTrue", func(t *testing.T) {
		entry, err := proc.parseABIEntry(map[string]interface{}{
			"constant": true,
		})
		require.NoError(t, err)
		assert.True(t, entry.Constant)
	})
	t.Run("withOutputs", func(t *testing.T) {
		entry, err := proc.parseABIEntry(map[string]interface{}{
			"type": "event",
			"outputs": []interface{}{
				map[string]interface{}{"name": "val", "type": "uint256"},
			},
		})
		require.NoError(t, err)
		assert.Len(t, entry.Outputs, 1)
	})
}

// ── abi_decode.go: formatDecodedValue ────────────────────────────────────────

func TestFormatDecodedValue(t *testing.T) {
	proc := NewABIProcessor(nil)

	t.Run("address", func(t *testing.T) {
		addr := common.HexToAddress("0x412d53ffb890dd6c173c9d495cb8a4806535c1f2be")
		result := proc.formatDecodedValue(addr, "address")
		// Should be a *types.Address or the original value if conversion fails
		assert.NotNil(t, result)
	})
	t.Run("bytes32", func(t *testing.T) {
		result := proc.formatDecodedValue([]byte{1, 2, 3}, "bytes32")
		assert.Equal(t, []byte{1, 2, 3}, result)
	})
	t.Run("bytesType", func(t *testing.T) {
		result := proc.formatDecodedValue([]byte{0xaa}, "bytes")
		assert.Equal(t, []byte{0xaa}, result)
	})
	t.Run("bytes16", func(t *testing.T) {
		result := proc.formatDecodedValue([]byte{0xbb}, "bytes16")
		assert.Equal(t, []byte{0xbb}, result)
	})
	t.Run("string", func(t *testing.T) {
		result := proc.formatDecodedValue("hello", "string")
		assert.Equal(t, "hello", result)
	})
	t.Run("bool", func(t *testing.T) {
		result := proc.formatDecodedValue(true, "bool")
		assert.Equal(t, true, result)
	})
	t.Run("addressNonAddressType", func(t *testing.T) {
		result := proc.formatDecodedValue("not-an-addr", "address")
		assert.Equal(t, "not-an-addr", result)
	})
	t.Run("array", func(t *testing.T) {
		result := proc.formatDecodedValue([]interface{}{true, false}, "bool[]")
		assert.Equal(t, []interface{}{true, false}, result)
	})
	t.Run("nonSliceArray", func(t *testing.T) {
		// For array suffix type but non-slice value, returns value as-is
		result := proc.formatDecodedValue(42, "uint256[]")
		assert.Equal(t, 42, result)
	})
	t.Run("fallback", func(t *testing.T) {
		result := proc.formatDecodedValue(999, "uint256")
		assert.Equal(t, 999, result)
	})
}

// ── abi_encode.go: convertBytes ──────────────────────────────────────────────

func TestConvertBytesCases(t *testing.T) {
	proc := NewABIProcessor(nil)

	t.Run("stringHex0x", func(t *testing.T) {
		result, err := proc.convertBytes("0xdeadbeef", 0)
		assert.NoError(t, err)
		assert.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, result)
	})
	t.Run("stringHexNoPrefix", func(t *testing.T) {
		result, err := proc.convertBytes("deadbeef", 0)
		assert.NoError(t, err)
		assert.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, result)
	})
	t.Run("stringInvalidHex", func(t *testing.T) {
		_, err := proc.convertBytes("zzzz", 0)
		assert.Error(t, err)
	})
	t.Run("sliceDirect", func(t *testing.T) {
		result, err := proc.convertBytes([]byte{0xaa, 0xbb}, 0)
		assert.NoError(t, err)
		assert.Equal(t, []byte{0xaa, 0xbb}, result)
	})
	t.Run("fixedSize32OK", func(t *testing.T) {
		data := make([]byte, 32)
		data[0] = 0xff
		result, err := proc.convertBytes(data, 32)
		assert.NoError(t, err)
		arr, ok := result.([32]byte)
		assert.True(t, ok)
		assert.Equal(t, byte(0xff), arr[0])
	})
	t.Run("fixedSize16OK", func(t *testing.T) {
		data := make([]byte, 16)
		result, err := proc.convertBytes(data, 16)
		assert.NoError(t, err)
		_, ok := result.([16]byte)
		assert.True(t, ok)
	})
	t.Run("fixedSize8OK", func(t *testing.T) {
		data := make([]byte, 8)
		result, err := proc.convertBytes(data, 8)
		assert.NoError(t, err)
		_, ok := result.([8]byte)
		assert.True(t, ok)
	})
	t.Run("fixedSize24Custom", func(t *testing.T) {
		data := make([]byte, 24)
		result, err := proc.convertBytes(data, 24)
		assert.NoError(t, err)
		// Custom size falls through to reflect-based array
		assert.NotNil(t, result)
	})
	t.Run("fixedSizeMismatch", func(t *testing.T) {
		_, err := proc.convertBytes([]byte{0x01, 0x02}, 32)
		assert.Error(t, err)
	})
	t.Run("fixedSizeStringMismatch", func(t *testing.T) {
		_, err := proc.convertBytes("0x0102", 32)
		assert.Error(t, err)
	})
	t.Run("fixedSizeCustomStringOK", func(t *testing.T) {
		data := make([]byte, 24)
		result, err := proc.convertBytes("0x"+encodeHex(data), 24)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run("fixedSizeCustomStringMismatch", func(t *testing.T) {
		_, err := proc.convertBytes("0x010203", 24)
		assert.Error(t, err)
	})
	t.Run("fixedSizeSlice32", func(t *testing.T) {
		data := make([]byte, 32)
		result, err := proc.convertBytes(data, 32)
		assert.NoError(t, err)
		_, ok := result.([32]byte)
		assert.True(t, ok)
	})
	t.Run("fixedSizeSlice16", func(t *testing.T) {
		data := make([]byte, 16)
		result, err := proc.convertBytes(data, 16)
		assert.NoError(t, err)
		_, ok := result.([16]byte)
		assert.True(t, ok)
	})
	t.Run("fixedSizeSlice8", func(t *testing.T) {
		data := make([]byte, 8)
		result, err := proc.convertBytes(data, 8)
		assert.NoError(t, err)
		_, ok := result.([8]byte)
		assert.True(t, ok)
	})
	t.Run("fixedSizeSliceMismatch", func(t *testing.T) {
		_, err := proc.convertBytes([]byte{0x01, 0x02}, 32)
		assert.Error(t, err)
	})
	t.Run("fixedSizeSliceCustom", func(t *testing.T) {
		data := make([]byte, 24)
		result, err := proc.convertBytes(data, 24)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run("fixedSizeSliceCustomMismatch", func(t *testing.T) {
		_, err := proc.convertBytes([]byte{0x01, 0x02}, 24)
		assert.Error(t, err)
	})
	t.Run("arrayType32", func(t *testing.T) {
		var arr [32]byte
		arr[0] = 0xcc
		result, err := proc.convertBytes(arr, 0)
		assert.NoError(t, err)
		assert.Equal(t, arr, result)
	})
	t.Run("unsupportedType", func(t *testing.T) {
		_, err := proc.convertBytes(12345, 0)
		assert.Error(t, err)
	})
}

// ── abi_encode.go: convertArrayParameter ─────────────────────────────────────

func TestConvertArrayParameter(t *testing.T) {
	proc := NewABIProcessor(nil)

	t.Run("jsonString", func(t *testing.T) {
		result, err := proc.convertArrayParameter(`[1, 2, 3]`, "uint256")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run("invalidJSON", func(t *testing.T) {
		_, err := proc.convertArrayParameter(`not json`, "uint256")
		assert.Error(t, err)
	})
	t.Run("sliceDirectUint", func(t *testing.T) {
		result, err := proc.convertArrayParameter([]uint64{100, 200}, "uint256")
		assert.NoError(t, err)
		assert.Equal(t, []uint64{100, 200}, result)
	})
	t.Run("sliceDirectAddress", func(t *testing.T) {
		addrs := []string{
			"412d53ffb890dd6c173c9d495cb8a4806535c1f2be",
		}
		result, err := proc.convertArrayParameter(addrs, "address")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run("unsupportedType", func(t *testing.T) {
		_, err := proc.convertArrayParameter(42, "uint256")
		assert.Error(t, err)
	})
}

// ── abi_encode.go: convertAddress edge cases ─────────────────────────────────

func TestConvertAddressEdgeCases(t *testing.T) {
	proc := NewABIProcessor(nil)

	t.Run("validHex", func(t *testing.T) {
		result, err := proc.convertAddress("0x412d53ffb890dd6c173c9d495cb8a4806535c1f2be")
		assert.NoError(t, err)
		assert.Equal(t, common.HexToAddress("0x412d53ffb890dd6c173c9d495cb8a4806535c1f2be"), result)
	})
	t.Run("invalidString", func(t *testing.T) {
		_, err := proc.convertAddress("not-an-address")
		assert.Error(t, err)
	})
	t.Run("unsupportedType", func(t *testing.T) {
		_, err := proc.convertAddress(12345)
		assert.Error(t, err)
	})
}

// ── abi_parse.go: parseParameters edge case ──────────────────────────────────

func TestParseParametersInvalidInputType(t *testing.T) {
	proc := NewABIProcessor(nil)
	t.Run("skipsNonMapItems", func(t *testing.T) {
		result, err := proc.parseParameters([]interface{}{12345})
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

// helper to encode bytes to hex string
func encodeHex(b []byte) string {
	const hexDigits = "0123456789abcdef"
	s := make([]byte, len(b)*2)
	for i, v := range b {
		s[i*2] = hexDigits[v>>4]
		s[i*2+1] = hexDigits[v&0x0f]
	}
	return string(s)
}
