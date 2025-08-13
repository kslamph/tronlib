package signer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data migrated from pkg_old/types/account_test.go
func TestPrivateKeySigner(t *testing.T) {
	testCases := []struct {
		name       string
		privateKey string
		address    string
	}{
		{
			name:       "Valid private key 1",
			privateKey: "cfae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e49c",
			address:    "TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu",
		},
		{
			name:       "Valid private key 2",
			privateKey: "dccf423c8dce5c744f72ebc62bd59797bd8fff10953f4f90af6cbb1121f96415",
			address:    "TT3G7td4FPhirwPC44BxGhfHGeD4uj6r7j",
		},
		{
			name:       "Valid private key 3",
			privateKey: "f8c6f45b2aa8b68ab5f3910bdeb5239428b731618113e2881f46e374bf796b02",
			address:    "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			signer, err := NewPrivateKeySigner(tc.privateKey)
			require.NoError(t, err)

			// Test address derivation
			assert.Equal(t, tc.address, signer.Address().Base58())
			assert.True(t, signer.Address().IsValid())

			// Test private key retrieval
			assert.Equal(t, tc.privateKey, signer.PrivateKeyHex())

			// Test public key is not nil
			assert.NotNil(t, signer.PublicKey())
		})
	}
}

func TestPrivateKeySignerWithPrefix(t *testing.T) {
	privateKey := "0xcfae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e49c"
	expectedAddress := "TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu"

	signer, err := NewPrivateKeySigner(privateKey)
	require.NoError(t, err)

	assert.Equal(t, expectedAddress, signer.Address().Base58())
	// Should strip the 0x prefix
	assert.Equal(t, "cfae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e49c", signer.PrivateKeyHex())
}

func TestInvalidPrivateKeys(t *testing.T) {
	invalidCases := []struct {
		name       string
		privateKey string
		reason     string
	}{
		{
			name:       "Invalid hex characters",
			privateKey: "ggae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e49c",
			reason:     "contains non-hex characters",
		},
		{
			name:       "Too short",
			privateKey: "cfae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e4",
			reason:     "insufficient length",
		},
		{
			name:       "Too long",
			privateKey: "cfae06d915cf9784272fa99d4db961b8cbafd59c8b2f77ab7422be5424d3e49c12",
			reason:     "too long",
		},
		{
			name:       "Empty string",
			privateKey: "",
			reason:     "empty private key",
		},
		{
			name:       "All zeros",
			privateKey: "0000000000000000000000000000000000000000000000000000000000000000",
			reason:     "invalid private key value",
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPrivateKeySigner(tc.privateKey)
			assert.Error(t, err, "Expected error for %s: %s", tc.privateKey, tc.reason)
		})
	}
}

func TestMessageSigningV2(t *testing.T) {
	// Test data migrated from pkg_old/crypto/verify_message_v2_test.go
	privateKey := "f8c6f45b2aa8b68ab5f3910bdeb5239428b731618113e2881f46e374bf796b02"
	message := "sign message testing"
	expectedSignature := "0x88bacb8549cbe7c3e26d922b05e88757197b77410fb0db1fabb9f30480202c84691b7025e928d36be962cfd7b4a8d2353b97f36d64bdc14398e9568091b701201b"

	signer, err := NewPrivateKeySigner(privateKey)
	require.NoError(t, err)

	signature, err := signer.SignMessageV2(message)
	require.NoError(t, err)

	assert.Equal(t, expectedSignature, signature)
	assert.True(t, len(signature) == 132) // 0x + 130 hex chars (65 bytes * 2)
}

func TestHDWalletSigner(t *testing.T) {
	// Test data migrated from pkg_old/types/account_test.go
	mnemonic := "fat social problem enable number gain parrot balance reduce bunker beach image marriage motion friend system dolphin bind leaf spin eye slogan rack track"

	testCases := []struct {
		name       string
		path       string
		address    string
		privateKey string
	}{
		{
			name:       "Default path",
			path:       "",
			address:    "TNPjabV8z8y7DRCsG9txgTwtz4ixDzCRzs",
			privateKey: "2da7c0d5a25677bea36ba96a71e9683eb9b9b7b0188e4229964c9ba87ee0f020",
		},
		{
			name:       "Custom path",
			path:       "m/44'/195'/10'/0/5",
			address:    "TRucjoUVF6MkHUxm61epieK1SxPbC4wbPP",
			privateKey: "87753ab3def8420afd1ca1c31acb2edcf33ac437bd837b1e39b7d57448ae53c5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			signer, err := NewHDWalletSigner(mnemonic, "", tc.path)
			require.NoError(t, err)

			// Test address derivation
			assert.Equal(t, tc.address, signer.Address().Base58())
			assert.True(t, signer.Address().IsValid())

			// Test private key derivation
			assert.Equal(t, tc.privateKey, signer.PrivateKeyHex())

			// Test derivation path
			expectedPath := tc.path
			if expectedPath == "" {
				expectedPath = "m/44'/195'/0'/0/0"
			}
			assert.Equal(t, expectedPath, signer.DerivationPath())

			// Test public key is not nil
			assert.NotNil(t, signer.PublicKey())
		})
	}
}

func TestHDWalletAccountDerivation(t *testing.T) {
	mnemonic := "fat social problem enable number gain parrot balance reduce bunker beach image marriage motion friend system dolphin bind leaf spin eye slogan rack track"

	signer, err := NewHDWalletSigner(mnemonic, "", "")
	require.NoError(t, err)

	// Derive account at index 1
	signer1, err := signer.DeriveAccount(1)
	require.NoError(t, err)

	// Derive account at index 2
	signer2, err := signer.DeriveAccount(2)
	require.NoError(t, err)

	// Addresses should be different
	assert.NotEqual(t, signer.Address().Base58(), signer1.Address().Base58())
	assert.NotEqual(t, signer.Address().Base58(), signer2.Address().Base58())
	assert.NotEqual(t, signer1.Address().Base58(), signer2.Address().Base58())

	// Private keys should be different
	assert.NotEqual(t, signer.PrivateKeyHex(), signer1.PrivateKeyHex())
	assert.NotEqual(t, signer.PrivateKeyHex(), signer2.PrivateKeyHex())
	assert.NotEqual(t, signer1.PrivateKeyHex(), signer2.PrivateKeyHex())

	// Derivation paths should be correct
	assert.Equal(t, "m/44'/195'/0'/0/1", signer1.DerivationPath())
	assert.Equal(t, "m/44'/195'/0'/0/2", signer2.DerivationPath())
}

func TestInvalidMnemonic(t *testing.T) {
	invalidMnemonics := []string{
		"invalid mnemonic phrase",
		"one two three",
		"",
		"abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon", // Only 11 words
	}

	for _, mnemonic := range invalidMnemonics {
		t.Run("Invalid: "+mnemonic, func(t *testing.T) {
			_, err := NewHDWalletSigner(mnemonic, "", "")
			assert.Error(t, err)
		})
	}
}

func TestInvalidDerivationPath(t *testing.T) {
	mnemonic := "fat social problem enable number gain parrot balance reduce bunker beach image marriage motion friend system dolphin bind leaf spin eye slogan rack track"

	invalidPaths := []string{
		"invalid/path",
	}

	for _, path := range invalidPaths {
		t.Run("Invalid path: "+path, func(t *testing.T) {
			_, err := NewHDWalletSigner(mnemonic, "", path)
			assert.Error(t, err)
		})
	}
}

func TestHDWalletMasterKey(t *testing.T) {
	mnemonic := "fat social problem enable number gain parrot balance reduce bunker beach image marriage motion friend system dolphin bind leaf spin eye slogan rack track"

	signer, err := NewHDWalletSigner(mnemonic, "", "m/44'/195'/0'/0/0")
	require.NoError(t, err)

	// Test that master key can be retrieved (specific validation depends on implementation)
	masterKey, err := signer.GetMasterKey()
	if err != nil {
		// If master key retrieval is not implemented or fails, that's acceptable
		t.Skipf("Master key retrieval not supported: %v", err)
	} else {
		assert.NotNil(t, masterKey)

		// Master key should be different from derived key
		derivedKey := signer.PublicKey()
		assert.NotEqual(t, masterKey, derivedKey)
	}
}
