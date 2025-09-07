package utils

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeAddress(t *testing.T) {
	address := "0x41a614f803b6fd780986a42c78ec9c7f77e6ded13c"

	t.Run("EncodeAddress", func(t *testing.T) {
		result, err := EncodeAddress(address)
		require.NoError(t, err)
		assert.Len(t, result, 32)
		// Check that the last 21 bytes match the address (after removing 0x prefix)
		expectedBytes, _ := HexToBytes(address)
		assert.Equal(t, expectedBytes, result[11:32])
	})

	t.Run("DecodeAddress", func(t *testing.T) {
		// First encode an address
		encoded, err := EncodeAddress(address)
		require.NoError(t, err)

		// Then decode it
		decoded := DecodeAddress(encoded)
		// The decoded address should match the original (without 0x41 prefix)
		expectedWithoutPrefix := "0xa614f803b6fd780986a42c78ec9c7f77e6ded13c"
		assert.Equal(t, expectedWithoutPrefix, decoded)
	})
}

func TestTRC20Encoding(t *testing.T) {
	t.Run("EncodeTRC20Transfer", func(t *testing.T) {
		toAddress := "0xa614f803b6fd780986a42c78ec9c7f77e6ded13c"
		amount := big.NewInt(1000000) // 1 token with 6 decimals

		result, err := EncodeTRC20Transfer(toAddress, amount)
		require.NoError(t, err)
		assert.Len(t, result, 68) // 4 bytes method sig + 32 bytes address + 32 bytes amount
		// Check that first 4 bytes are the method signature for transfer
		expectedSig := []byte{0xa9, 0x05, 0x9c, 0xbb}
		assert.Equal(t, expectedSig, result[:4])
	})

	t.Run("EncodeTRC20BalanceOf", func(t *testing.T) {
		address := "0xa614f803b6fd780986a42c78ec9c7f77e6ded13c"

		result, err := EncodeTRC20BalanceOf(address)
		require.NoError(t, err)
		assert.Len(t, result, 36) // 4 bytes method sig + 32 bytes address
		// Check that first 4 bytes are the method signature for balanceOf
		expectedSig := []byte{0x70, 0xa0, 0x82, 0x31}
		assert.Equal(t, expectedSig, result[:4])
	})

	t.Run("DecodeTRC20Balance", func(t *testing.T) {
		// Create a 32-byte balance value
		balanceBytes := make([]byte, 32)
		balanceBytes[31] = 0x64 // 100 in decimal
		balanceBytes[30] = 0x01 // Additional byte for larger number

		result, err := DecodeTRC20Balance(balanceBytes)
		require.NoError(t, err)
		expected := new(big.Int).SetBytes([]byte{0x01, 0x64}) // 0x0164 = 356
		assert.Equal(t, 0, expected.Cmp(result))
	})

	t.Run("DecodeTRC20BalanceInvalidData", func(t *testing.T) {
		// Test with invalid (short) data
		_, err := DecodeTRC20Balance([]byte{0x01, 0x02})
		assert.Error(t, err)
	})
}
