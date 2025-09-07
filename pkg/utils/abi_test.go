package utils

import (
	"encoding/hex"
	"math/big"
	"testing"

	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHexToBytes(t *testing.T) {
	tests := []struct {
		name     string
		hexStr   string
		expected []byte
		hasError bool
	}{
		{
			name:     "Valid hex without prefix",
			hexStr:   "a9059cbb",
			expected: []byte{0xa9, 0x05, 0x9c, 0xbb},
			hasError: false,
		},
		{
			name:     "Valid hex with prefix",
			hexStr:   "0xa9059cbb",
			expected: []byte{0xa9, 0x05, 0x9c, 0xbb},
			hasError: false,
		},
		{
			name:     "Valid hex with odd length",
			hexStr:   "0x0a9059cbb",
			expected: []byte{0x00, 0xa9, 0x05, 0x9c, 0xbb},
			hasError: false,
		},
		{
			name:     "Empty string",
			hexStr:   "",
			expected: []byte{},
			hasError: false,
		},
		{
			name:     "Invalid hex characters",
			hexStr:   "0xzzzz",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HexToBytes(tt.hexStr)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBytesToHex(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "Valid bytes",
			data:     []byte{0xa9, 0x05, 0x9c, 0xbb},
			expected: "0xa9059cbb",
		},
		{
			name:     "Empty bytes",
			data:     []byte{},
			expected: "0x",
		},
		{
			name:     "Single byte",
			data:     []byte{0xff},
			expected: "0xff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BytesToHex(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		length   int
		expected []byte
	}{
		{
			name:     "Pad shorter data",
			data:     []byte{0x01, 0x02},
			length:   4,
			expected: []byte{0x00, 0x00, 0x01, 0x02},
		},
		{
			name:     "No padding needed",
			data:     []byte{0x01, 0x02, 0x03, 0x04},
			length:   4,
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "Pad to longer length",
			data:     []byte{0x01},
			length:   8,
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		},
		{
			name:     "Empty data",
			data:     []byte{},
			length:   4,
			expected: []byte{0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadLeft(tt.data, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		length   int
		expected []byte
	}{
		{
			name:     "Pad shorter data",
			data:     []byte{0x01, 0x02},
			length:   4,
			expected: []byte{0x01, 0x02, 0x00, 0x00},
		},
		{
			name:     "No padding needed",
			data:     []byte{0x01, 0x02, 0x03, 0x04},
			length:   4,
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "Pad to longer length",
			data:     []byte{0x01},
			length:   8,
			expected: []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "Empty data",
			data:     []byte{},
			length:   4,
			expected: []byte{0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadRight(tt.data, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEncodeMethodSignature(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected []byte
	}{
		{
			name:     "Standard method",
			method:   "transfer(address,uint256)",
			expected: []byte{'t', 'r', 'a', 'n'},
		},
		{
			name:     "Short method",
			method:   "a()",
			expected: []byte{'a', '(', ')', 0x00},
		},
		{
			name:     "Empty method",
			method:   "",
			expected: []byte{0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeMethodSignature(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEncodeUint256(t *testing.T) {
	tests := []struct {
		name     string
		value    *big.Int
		expected []byte
	}{
		{
			name:     "Small number",
			value:    big.NewInt(1),
			expected: append(make([]byte, 31), 0x01),
		},
		{
			name:     "Large number",
			value:    big.NewInt(1000000),
			expected: append(make([]byte, 24), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x42, 0x40}...),
		},
		{
			name:     "Zero",
			value:    big.NewInt(0),
			expected: make([]byte, 32),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeUint256(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecodeUint256(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected *big.Int
	}{
		{
			name:     "Standard 32-byte data",
			data:     append(make([]byte, 31), 0x01),
			expected: big.NewInt(1),
		},
		{
			name:     "Zero value",
			data:     make([]byte, 32),
			expected: big.NewInt(0),
		},
		{
			name:     "Short data",
			data:     []byte{0x01},
			expected: big.NewInt(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeUint256(tt.data)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected.Cmp(result), 0)
			}
		})
	}
}

func TestEncodeAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected []byte
		hasError bool
	}{
		{
			name:     "Valid hex address",
			address:  "41e28b3cfd4e0e909077821478e9fcb86b84be786e",
			expected: append(make([]byte, 11), []byte{0x41, 0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82, 0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78, 0x6e}...),
			hasError: false,
		},
		{
			name:     "Valid hex address with 0x prefix",
			address:  "0x41e28b3cfd4e0e909077821478e9fcb86b84be786e",
			expected: append(make([]byte, 11), []byte{0x41, 0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82, 0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78, 0x6e}...),
			hasError: false,
		},
		{
			name:     "Invalid hex characters",
			address:  "0xzzzz",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeAddress(tt.address)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecodeAddress(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "Valid address data",
			data:     append(make([]byte, 11), []byte{0x41, 0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82, 0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78, 0x6e}...),
			expected: "0xe28b3cfd4e0e909077821478e9fcb86b84be786e",
		},
		{
			name:     "Short data",
			data:     []byte{0x01, 0x02},
			expected: "",
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeAddress(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestABIEncoder_EncodeMethod(t *testing.T) {
	processor := NewABIProcessor(nil)

	tests := []struct {
		name       string
		method     string
		paramTypes []string
		params     []interface{}
		hasError   bool
	}{
		{
			name:       "Constructor with no parameters",
			method:     "",
			paramTypes: []string{},
			params:     []interface{}{},
			hasError:   false,
		},
		{
			name:       "Method with no parameters",
			method:     "getValue",
			paramTypes: []string{},
			params:     []interface{}{},
			hasError:   false,
		},
		{
			name:       "Method with parameters mismatch",
			method:     "setValue",
			paramTypes: []string{"uint256"},
			params:     []interface{}{},
			hasError:   false, // The function doesn't validate parameter count
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := processor.EncodeMethod(tt.method, tt.paramTypes, tt.params)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestABIEncoder_ConvertAddress(t *testing.T) {
	processor := NewABIProcessor(nil)

	// Test with a valid TRON address
	tronAddr := "TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh"

	t.Run("String address", func(t *testing.T) {
		result, err := processor.convertAddress(tronAddr)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	})

	t.Run("Invalid address string", func(t *testing.T) {
		_, err := processor.convertAddress("invalid_address")
		assert.Error(t, err)
	})
}

func TestABIEncoder_EncodeParameters(t *testing.T) {
	processor := NewABIProcessor(nil)

	tests := []struct {
		name       string
		paramTypes []string
		params     []interface{}
		hasError   bool
	}{
		{
			name:       "No parameters",
			paramTypes: []string{},
			params:     []interface{}{},
			hasError:   false,
		},
		{
			name:       "Invalid parameter type",
			paramTypes: []string{"invalidType"},
			params:     []interface{}{"test"},
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := processor.EncodeMethod("testMethod", tt.paramTypes, tt.params)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestABIDecoder_DecodeParameters(t *testing.T) {
	processor := NewABIProcessor(nil)

	// Test with simple parameters
	tests := []struct {
		name     string
		data     []byte
		hasError bool
	}{
		{
			name:     "Empty parameters",
			data:     []byte{},
			hasError: false,
		},
		{
			name:     "Invalid data",
			data:     []byte{0x01},
			hasError: false, // The function doesn't validate data format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't fully test this without proper ABI entries, but we can test error cases
			params := []*core.SmartContract_ABI_Entry_Param{}
			_, err := processor.decodeParameters(tt.data, params)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestABIDecoder_FormatDecodedValue(t *testing.T) {
	processor := NewABIProcessor(nil)

	t.Run("Address decoding", func(t *testing.T) {
		// Create an Ethereum address from the EVM bytes of our TRON address
		evmBytes, _ := hex.DecodeString("c24e347a3d32f348aef332e819fb87586d4fb77c")
		ethAddr := eCommon.BytesToAddress(evmBytes)

		result := processor.formatDecodedValue(ethAddr, "address")
		exp := types.MustNewAddressFromBase58("TTgbn3yTSzVDP3BrH9EonLNLvhbfyT3TXh")
		// The result should be a string (the base58 representation of the TRON address)
		assert.IsType(t, exp, result)
		assert.Equal(t, exp, result)
	})
}

func TestABIParser_GetMethodTypes(t *testing.T) {
	// Test with a simple ABI
	abiJSON := `[{"name":"getValue","type":"function","inputs":[],"outputs":[{"name":"","type":"uint256"}]}]`
	abi, err := NewABIProcessor(nil).ParseABI(abiJSON)
	require.NoError(t, err)

	processor := NewABIProcessor(abi)

	tests := []struct {
		name        string
		methodName  string
		hasError    bool
		inputTypes  []string
		outputTypes []string
	}{
		{
			name:        "Existing method",
			methodName:  "getValue",
			hasError:    false,
			inputTypes:  []string{},
			outputTypes: []string{"uint256"},
		},
		{
			name:       "Non-existing method",
			methodName: "setValue",
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputTypes, outputTypes, err := processor.GetMethodTypes(tt.methodName)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.inputTypes, inputTypes)
			assert.Equal(t, tt.outputTypes, outputTypes)
		})
	}
}
