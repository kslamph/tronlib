package signer

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrivateKeySignerFromECDSA(t *testing.T) {
	t.Run("valid ECDSA key", func(t *testing.T) {
		privKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		s, err := NewPrivateKeySignerFromECDSA(privKey)
		require.NoError(t, err)
		require.NotNil(t, s)

		assert.NotNil(t, s.Address())
		assert.True(t, s.Address().IsValid())
		assert.NotNil(t, s.PublicKey())
		assert.NotEmpty(t, s.PrivateKeyHex())

		// Verify the derived address matches what we'd get via hex roundtrip
		hexKey := s.PrivateKeyHex()
		s2, err := NewPrivateKeySigner(hexKey)
		require.NoError(t, err)
		assert.Equal(t, s.Address().Base58(), s2.Address().Base58())
	})

	t.Run("sign with ECDSA-derived signer", func(t *testing.T) {
		privKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		s, err := NewPrivateKeySignerFromECDSA(privKey)
		require.NoError(t, err)

		hash := []byte("test hash for signing 32 bytes!!")
		sig, err := s.Sign(hash)
		require.NoError(t, err)
		assert.NotEmpty(t, sig)
		assert.Len(t, sig, 65) // r(32) + s(32) + recovery(1)
	})

	t.Run("known key matches NewPrivateKeySigner", func(t *testing.T) {
		hexKey := "cfae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e49c"
		expectedAddr := "TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu"

		keyBytes, err := hexToBytes(hexKey)
		require.NoError(t, err)
		privKey, err := crypto.ToECDSA(keyBytes)
		require.NoError(t, err)

		s, err := NewPrivateKeySignerFromECDSA(privKey)
		require.NoError(t, err)
		assert.Equal(t, expectedAddr, s.Address().Base58())
	})
}

func TestSignerInterface(t *testing.T) {
	// Verify PrivateKeySigner implements the Signer interface
	var _ Signer = (*PrivateKeySigner)(nil)
}

// hexToBytes decodes a hex string to bytes.
func hexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}
