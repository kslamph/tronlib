package utils

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeTRC20Transfer(t *testing.T) {
	tests := []struct {
		name        string
		to          string
		amount      *big.Int
		expectedLen int
		hasError    bool
	}{
		{
			name:        "Valid transfer",
			to:          "41e28b3cfd4e0e909077821478e9fcb86b84be786e",
			amount:      big.NewInt(1000000),
			expectedLen: 68, // 4 bytes method sig + 32 bytes address + 32 bytes amount
			hasError:    false,
		},
		{
			name:        "Invalid address",
			to:          "invalid_hex",
			amount:      big.NewInt(1000000),
			expectedLen: 0,
			hasError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeTRC20Transfer(tt.to, tt.amount)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedLen, len(result))
			// Check that first 4 bytes are the method signature for transfer
			expectedSig := []byte{0xa9, 0x05, 0x9c, 0xbb}
			assert.Equal(t, expectedSig, result[:4])
		})
	}
}

func TestEncodeTRC20BalanceOf(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		expectedLen int
		hasError    bool
	}{
		{
			name:        "Valid address",
			address:     "41e28b3cfd4e0e909077821478e9fcb86b84be786e",
			expectedLen: 36, // 4 bytes method sig + 32 bytes address
			hasError:    false,
		},
		{
			name:        "Invalid address",
			address:     "invalid_hex",
			expectedLen: 0,
			hasError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeTRC20BalanceOf(tt.address)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedLen, len(result))
			// Check that first 4 bytes are the method signature for balanceOf
			expectedSig := []byte{0x70, 0xa0, 0x82, 0x31}
			assert.Equal(t, expectedSig, result[:4])
		})
	}
}

func TestDecodeTRC20Balance(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected *big.Int
		hasError bool
	}{
		{
			name:     "Valid balance data",
			data:     append(make([]byte, 31), 0x01), // 31 zeros + 1
			expected: big.NewInt(1),
			hasError: false,
		},
		{
			name:     "Invalid data length",
			data:     []byte{0x01, 0x02}, // Only 2 bytes
			expected: nil,
			hasError: true,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeTRC20Balance(tt.data)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 0, tt.expected.Cmp(result))
		})
	}
}

func TestEncodeString(t *testing.T) {
	tests := []struct {
		name   string
		str    string
		minLen int
	}{
		{
			name:   "Simple string",
			str:    "hello",
			minLen: 64, // At least 64 bytes for offset + length + data
		},
		{
			name:   "Empty string",
			str:    "",
			minLen: 64,
		},
		{
			name:   "Longer string",
			str:    "This is a longer test string for encoding",
			minLen: 96, // Should be padded to 32-byte boundary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeString(tt.str)
			assert.GreaterOrEqual(t, len(result), tt.minLen)
			// First 32 bytes should be offset (typically 32)
			offset := DecodeUint256(result[:32])
			assert.Equal(t, int64(32), offset.Int64())
		})
	}
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
		hasError bool
	}{
		{
			name:     "Invalid data length",
			data:     []byte{0x01, 0x02}, // Only 2 bytes
			expected: "",
			hasError: true,
		},
		{
			name:     "Invalid offset",
			data:     append([]byte{0, 0, 0, 0, 0, 0, 0, 64, 0, 0, 0, 0, 0, 0, 0, 5, 'h', 'e', 'l', 'l', 'o'}, make([]byte, 11)...),
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeString(tt.data)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONToMap(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		hasError bool
	}{
		{
			name:     "Valid JSON",
			jsonStr:  `{"key": "value", "number": 123}`,
			hasError: false,
		},
		{
			name:     "Invalid JSON",
			jsonStr:  `{"key": "value", "number": 123`, // Missing closing brace
			hasError: true,
		},
		{
			name:     "Empty JSON",
			jsonStr:  `{}`,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JSONToMap(tt.jsonStr)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestMapToJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		hasError bool
	}{
		{
			name:     "Valid map",
			data:     map[string]interface{}{"key": "value", "number": 123},
			hasError: false,
		},
		{
			name:     "Empty map",
			data:     map[string]interface{}{},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MapToJSON(tt.data)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}
}

func TestParseBigInt(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		expected *big.Int
		hasError bool
	}{
		{
			name:     "Valid decimal",
			str:      "123456789",
			expected: big.NewInt(123456789),
			hasError: false,
		},
		{
			name:     "Valid hex with 0x prefix",
			str:      "0x1234abcd",
			expected: big.NewInt(0x1234abcd),
			hasError: false,
		},
		{
			name:     "Invalid format",
			str:      "invalid123",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseBigInt(tt.str)

			if tt.hasError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 0, tt.expected.Cmp(result))
		})
	}
}

func TestFormatBigInt(t *testing.T) {
	tests := []struct {
		name     string
		value    *big.Int
		decimals int
		expected string
	}{
		{
			name:     "No decimals",
			value:    big.NewInt(123456789),
			decimals: 0,
			expected: "123456789",
		},
		{
			name:     "With decimals",
			value:    big.NewInt(123456789),
			decimals: 6,
			expected: "123.456789",
		},
		{
			name:     "Zero value",
			value:    big.NewInt(0),
			decimals: 6,
			expected: "0",
		},
		{
			name:     "Exact division",
			value:    big.NewInt(1000000),
			decimals: 6,
			expected: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBigInt(tt.value, tt.decimals)
			assert.Equal(t, tt.expected, result)
		})
	}
}
