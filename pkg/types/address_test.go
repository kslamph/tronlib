package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data migrated from pkg_old/types/address_test.go
func TestAddressConversion(t *testing.T) {
	testCases := []struct {
		name   string
		base58 string
		hex    string // Without 0x41 prefix (20 bytes)
	}{
		{
			name:   "Valid address pair 1",
			base58: "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb",
			hex:    "e28b3cfd4e0e909077821478e9fcb86b84be786e",
		},
		{
			name:   "Valid address pair 2",
			base58: "TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx",
			hex:    "eac49bc766be29be1b6d36619eff8f86ed4d04df",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test Base58 -> Address object
			addr, err := NewAddressFromBase58(tc.base58)
			require.NoError(t, err)
			assert.True(t, addr.IsValid())

			// Test Hex -> Address object
			addr2, err := NewAddressFromHex(tc.hex)
			require.NoError(t, err)
			assert.True(t, addr2.IsValid())

			// Test round-trip consistency
			assert.Equal(t, addr.Hex(), addr2.Hex())
			assert.Equal(t, addr.Base58(), addr2.Base58())
			assert.True(t, addr.Equal(addr2))
		})
	}
}

func TestInvalidBase58Addresses(t *testing.T) {
	invalidCases := []struct {
		name    string
		address string
		reason  string
	}{
		{
			name:    "Wrong prefix",
			address: "AWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb",
			reason:  "should start with T",
		},
		{
			name:    "Wrong length - too short",
			address: "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5",
			reason:  "incorrect length",
		},
		{
			name:    "Out of range characters",
			address: "T9rfYxWFRJMk9kRTAjvfYFsw2NLY92fd65",
			reason:  "invalid base58 characters",
		},
		{
			name:    "Empty address",
			address: "",
			reason:  "empty string",
		},
		{
			name:    "Invalid checksum",
			address: "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwc",
			reason:  "invalid checksum",
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAddressFromBase58(tc.address)
			assert.Error(t, err, "Expected error for %s: %s", tc.address, tc.reason)
		})
	}
}

func TestInvalidHexAddresses(t *testing.T) {
	invalidCases := []struct {
		name    string
		address string
		reason  string
	}{
		{
			name:    "Wrong prefix",
			address: "51e28b3cfd4e0e909077821478e9fcb86b84be786e",
			reason:  "should start with 41",
		},
		{
			name:    "Wrong length - too long",
			address: "41e28b3cfd4e0e909077821478e9fcb86b84be783840",
			reason:  "too many characters",
		},
		{
			name:    "Invalid hex characters",
			address: "41x28b3cfd4e0e909077821478e9fcb86b84be786e",
			reason:  "contains non-hex characters",
		},
		{
			name:    "Empty address",
			address: "",
			reason:  "empty string",
		},
		{
			name:    "Too short",
			address: "41e28b",
			reason:  "insufficient length",
		},
		{
			name:    "With 0x prefix but wrong length",
			address: "0x41e28b3cfd4e0e909077821478e9fcb86b84be786e12",
			reason:  "wrong length even with 0x prefix",
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAddressFromHex(tc.address)
			assert.Error(t, err, "Expected error for %s: %s", tc.address, tc.reason)
		})
	}
}

func TestAddressCreationFromBytes(t *testing.T) {
	t.Run("Valid bytes", func(t *testing.T) {
		validBytes := []byte{
			0x41, 0xe2, 0x8b, 0x3c, 0xfd, 0x4e, 0x0e, 0x90, 0x90, 0x77, 0x82,
			0x14, 0x78, 0xe9, 0xfc, 0xb8, 0x6b, 0x84, 0xbe, 0x78, 0x6e,
		}

		addr, err := NewAddressFromBytes(validBytes)
		require.NoError(t, err)
		assert.True(t, addr.IsValid())
		assert.Equal(t, validBytes, addr.Bytes())
	})

	t.Run("Invalid length - too short", func(t *testing.T) {
		invalidBytes := []byte{0x41, 0xe2, 0x8b}

		_, err := NewAddressFromBytes(invalidBytes)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid address length")
	})

	t.Run("Invalid length - too long", func(t *testing.T) {
		invalidBytes := make([]byte, 25)
		invalidBytes[0] = 0x41

		_, err := NewAddressFromBytes(invalidBytes)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid address length")
	})
}

func TestAddressStringMethods(t *testing.T) {
	base58Addr := "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb"

	addr, err := NewAddressFromBase58(base58Addr)
	require.NoError(t, err)

	t.Run("Hex method", func(t *testing.T) {
		hexResult := addr.Hex()
		assert.NotEmpty(t, hexResult)
		assert.Equal(t, 42, len(hexResult)) // 21 bytes * 2 = 42 hex chars
	})

	t.Run("Base58 method", func(t *testing.T) {
		assert.Equal(t, base58Addr, addr.Base58())
	})

	t.Run("String method (should return Base58)", func(t *testing.T) {
		assert.Equal(t, base58Addr, addr.String())
	})
}

func TestAddressEquality(t *testing.T) {
	base58Addr := "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb"

	addr1, err := NewAddressFromBase58(base58Addr)
	require.NoError(t, err)

	addr2, err := NewAddressFromBase58(base58Addr)
	require.NoError(t, err)

	addr3, err := NewAddressFromBase58("TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx")
	require.NoError(t, err)

	t.Run("Equal addresses", func(t *testing.T) {
		assert.True(t, addr1.Equal(addr2))
		assert.True(t, addr2.Equal(addr1))
	})

	t.Run("Different addresses", func(t *testing.T) {
		assert.False(t, addr1.Equal(addr3))
		assert.False(t, addr3.Equal(addr1))
	})

	t.Run("Nil address comparisons", func(t *testing.T) {
		var nilAddr *Address
		assert.True(t, nilAddr.Equal(nilAddr))
		assert.False(t, addr1.Equal(nilAddr))
		assert.False(t, nilAddr.Equal(addr1))
	})
}

func TestNilAddressMethods(t *testing.T) {
	var nilAddr *Address

	t.Run("Nil address methods", func(t *testing.T) {
		assert.Nil(t, nilAddr.Bytes())
		assert.Equal(t, "", nilAddr.Hex())

		assert.Equal(t, "", nilAddr.Base58())
		assert.Equal(t, "", nilAddr.String())
		assert.False(t, nilAddr.IsValid())
	})
}

func TestGenerateContractAddress(t *testing.T) {
	t.Run("Valid contract address generation", func(t *testing.T) {
		creatorAddr, err := NewAddressFromBase58("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
		require.NoError(t, err)

		contractAddr, err := GenerateContractAddress(creatorAddr, 1)
		require.NoError(t, err)
		assert.True(t, contractAddr.IsValid())
		assert.NotEqual(t, creatorAddr.Hex(), contractAddr.Hex())
	})

	t.Run("Different nonces generate different addresses", func(t *testing.T) {
		creatorAddr, err := NewAddressFromBase58("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
		require.NoError(t, err)

		addr1, err := GenerateContractAddress(creatorAddr, 1)
		require.NoError(t, err)

		addr2, err := GenerateContractAddress(creatorAddr, 2)
		require.NoError(t, err)

		assert.False(t, addr1.Equal(addr2))
	})

	t.Run("Nil creator address", func(t *testing.T) {
		_, err := GenerateContractAddress(nil, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "creator address is nil")
	})
}

func TestHexAddressWithPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Hex with 0x prefix",
			input:    "0xe28b3cfd4e0e909077821478e9fcb86b84be786e",
			expected: "41e28b3cfd4e0e909077821478e9fcb86b84be786e",
		},
		{
			name:     "Hex without prefix",
			input:    "e28b3cfd4e0e909077821478e9fcb86b84be786e",
			expected: "41e28b3cfd4e0e909077821478e9fcb86b84be786e",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addr, err := NewAddressFromHex(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, addr.Hex())
		})
	}
}
