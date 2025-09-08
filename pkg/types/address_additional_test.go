package types

import (
	"testing"

	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustNewAddressFromBase58(t *testing.T) {
	t.Run("Valid address", func(t *testing.T) {
		validAddr := "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb"
		assert.NotPanics(t, func() {
			addr := MustNewAddressFromBase58(validAddr)
			assert.NotNil(t, addr)
			assert.Equal(t, validAddr, addr.Base58())
		})
	})

	t.Run("Invalid address should panic", func(t *testing.T) {
		invalidAddr := "invalid_address"
		assert.Panics(t, func() {
			MustNewAddressFromBase58(invalidAddr)
		})
	})
}

func TestMustNewAddressFromHex(t *testing.T) {
	t.Run("Valid hex address without prefix", func(t *testing.T) {
		validHex := "e28b3cfd4e0e909077821478e9fcb86b84be786e"
		assert.NotPanics(t, func() {
			addr := MustNewAddressFromHex(validHex)
			assert.NotNil(t, addr)
			assert.Equal(t, "41"+validHex, addr.Hex())
		})
	})

	t.Run("Valid hex address with prefix", func(t *testing.T) {
		validHex := "0xe28b3cfd4e0e909077821478e9fcb86b84be786e"
		assert.NotPanics(t, func() {
			addr := MustNewAddressFromHex(validHex)
			assert.NotNil(t, addr)
			assert.Equal(t, "41e28b3cfd4e0e909077821478e9fcb86b84be786e", addr.Hex())
		})
	})

	t.Run("Invalid hex address should panic", func(t *testing.T) {
		invalidHex := "invalid_hex"
		assert.Panics(t, func() {
			MustNewAddressFromHex(invalidHex)
		})
	})
}

func TestMustNewAddressFromBytes(t *testing.T) {
	t.Run("Valid 21-byte address", func(t *testing.T) {
		validBytes := []byte{
			0x41, 0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82,
			0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78, 0x6e,
		}
		assert.NotPanics(t, func() {
			addr := MustNewAddressFromBytes(validBytes)
			assert.NotNil(t, addr)
			assert.Equal(t, validBytes, addr.Bytes())
		})
	})

	t.Run("Valid 20-byte address (will be prefixed)", func(t *testing.T) {
		validBytes := []byte{
			0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82,
			0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78, 0x6e,
		}
		assert.NotPanics(t, func() {
			addr := MustNewAddressFromBytes(validBytes)
			assert.NotNil(t, addr)
			assert.Len(t, addr.Bytes(), 21)
			assert.Equal(t, byte(0x41), addr.Bytes()[0])
		})
	})

	t.Run("Invalid byte length should panic", func(t *testing.T) {
		invalidBytes := []byte{0x01, 0x02, 0x03}
		assert.Panics(t, func() {
			MustNewAddressFromBytes(invalidBytes)
		})
	})
}

func TestAddressBytesEVM(t *testing.T) {
	base58Addr := "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb"
	addr, err := NewAddressFromBase58(base58Addr)
	require.NoError(t, err)

	t.Run("Valid address EVM bytes", func(t *testing.T) {
		evmBytes := addr.BytesEVM()
		assert.NotNil(t, evmBytes)
		assert.Len(t, evmBytes, 20) // Should be 20 bytes without prefix
		// Should match the original bytes without the 0x41 prefix
		expected := addr.Bytes()[1:] // Skip the first byte (0x41)
		assert.Equal(t, expected, evmBytes)
	})

	t.Run("Nil address EVM bytes", func(t *testing.T) {
		var nilAddr *Address
		evmBytes := nilAddr.BytesEVM()
		assert.Nil(t, evmBytes)
	})
}

func TestAddressHexEVM(t *testing.T) {
	base58Addr := "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb"
	addr, err := NewAddressFromBase58(base58Addr)
	require.NoError(t, err)

	t.Run("Valid address EVM hex", func(t *testing.T) {
		evmHex := addr.HexEVM()
		assert.NotEmpty(t, evmHex)
		assert.True(t, len(evmHex) > 2)   // Should have 0x prefix
		assert.Equal(t, "0x", evmHex[:2]) // Should start with 0x
		assert.Len(t, evmHex, 42)         // 0x + 40 hex chars = 42 chars
	})

	t.Run("Nil address EVM hex", func(t *testing.T) {
		var nilAddr *Address
		evmHex := nilAddr.HexEVM()
		assert.Empty(t, evmHex)
	})
}

func TestAddressEVMAddress(t *testing.T) {
	base58Addr := "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb"
	addr, err := NewAddressFromBase58(base58Addr)
	require.NoError(t, err)

	t.Run("Valid address to EVM address", func(t *testing.T) {
		evmAddr := addr.EVMAddress()
		assert.NotEmpty(t, evmAddr)
		// The EVM address should match the EVM bytes
		expectedBytes := addr.BytesEVM()
		assert.Equal(t, expectedBytes, evmAddr.Bytes())
	})

	t.Run("Nil address EVM conversion should panic", func(t *testing.T) {
		var nilAddr *Address
		assert.Panics(t, func() {
			nilAddr.EVMAddress()
		})
	})
}

func TestNewAddressFromEVM(t *testing.T) {
	// Create an Ethereum address
	evmBytes := []byte{
		0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82,
		0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78, 0x6e,
	}
	evmAddr := eCommon.BytesToAddress(evmBytes)

	t.Run("Valid EVM address", func(t *testing.T) {
		addr, err := NewAddressFromEVM(evmAddr)
		require.NoError(t, err)
		assert.NotNil(t, addr)
		assert.Len(t, addr.Bytes(), 21) // Should be 21 bytes with 0x41 prefix
		assert.Equal(t, byte(0x41), addr.Bytes()[0])
		// The EVM bytes should match
		assert.Equal(t, evmBytes, addr.BytesEVM())
	})

	t.Run("Round trip conversion", func(t *testing.T) {
		addr, err := NewAddressFromEVM(evmAddr)
		require.NoError(t, err)

		// Convert back to EVM address
		evmAddr2 := addr.EVMAddress()
		assert.Equal(t, evmAddr, evmAddr2)
	})
}

func TestNewAddressGenericFunction(t *testing.T) {
	validBase58 := "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb"
	validHex := "e28b3cfd4e0e909077821478e9fcb86b84be786e"
	validBytes := []byte{
		0x41, 0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82,
		0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78, 0x6e,
	}
	evmBytes := []byte{
		0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82,
		0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78, 0x6e,
	}
	evmAddr := eCommon.BytesToAddress(evmBytes)

	t.Run("String - Base58", func(t *testing.T) {
		addr, err := NewAddress(validBase58)
		require.NoError(t, err)
		assert.Equal(t, validBase58, addr.Base58())
	})

	t.Run("String - Hex without prefix", func(t *testing.T) {
		addr, err := NewAddress(validHex)
		require.NoError(t, err)
		assert.Equal(t, "41"+validHex, addr.Hex())
	})

	t.Run("Bytes slice", func(t *testing.T) {
		addr, err := NewAddress(validBytes)
		require.NoError(t, err)
		assert.Equal(t, validBytes, addr.Bytes())
	})

	t.Run("Address pointer", func(t *testing.T) {
		originalAddr, _ := NewAddressFromBase58(validBase58)
		addr, err := NewAddress(originalAddr)
		require.NoError(t, err)
		assert.Equal(t, originalAddr, addr)
	})

	t.Run("EVM Address", func(t *testing.T) {
		addr, err := NewAddress(&evmAddr)
		require.NoError(t, err)
		assert.Equal(t, evmBytes, addr.BytesEVM())
	})

	t.Run("20-byte array", func(t *testing.T) {
		var byteArray [20]byte
		copy(byteArray[:], evmBytes)
		addr, err := NewAddress(byteArray)
		require.NoError(t, err)
		assert.Equal(t, evmBytes, addr.BytesEVM())
	})

	t.Run("21-byte array", func(t *testing.T) {
		var byteArray [21]byte
		copy(byteArray[:], validBytes)
		addr, err := NewAddress(byteArray)
		require.NoError(t, err)
		assert.Equal(t, validBytes, addr.Bytes())
	})
}
