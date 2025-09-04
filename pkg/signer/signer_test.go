// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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

