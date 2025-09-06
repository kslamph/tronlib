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

	"github.com/stretchr/testify/require"
)

func TestNewHDWalletSigner(t *testing.T) {
	testCases := []struct {
		name         string
		mnemonic     string
		path         string
		expectedErr  bool
		expectedAddr string // For valid cases only
	}{
		{
			name:         "Valid 12-word Mnemonic",
			mnemonic:     "rebel move punch grant loop beyond stadium dumb appear enough typical remind",
			path:         "m/44'/195'/0'/0/0",
			expectedErr:  false,
			expectedAddr: "TSnGCcTTDGUQ5ueKZDFTUGYGf9NuGGm5JA", // Actual derived address
		},
		{
			name:         "Valid 24-word Mnemonic",
			mnemonic:     "fruit oxygen mixed tape slight universe mule original advice sniff layer erode sport tomato found coffee cheap gap insect rubber broccoli silk sister point",
			path:         "m/44'/195'/0'/0/0",
			expectedErr:  false,
			expectedAddr: "TPDyYwpaVkzCARSbJmdD8PpZXTocGzotTM", // Actual derived address
		},
		{
			name:         "Invalid Mnemonic",
			mnemonic:     "this is an invalid mnemonic phrase that should fail",
			path:         "m/44'/195'/0'/0/0",
			expectedErr:  true,
			expectedAddr: "", // Expect an error, so no address is derived
		},
		{
			name:         "Invalid Path",
			mnemonic:     "rebel move punch grant loop beyond stadium dumb appear enough typical remind",
			path:         "invalid/path",
			expectedErr:  true,
			expectedAddr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			signer, err := NewHDWalletSigner(tc.mnemonic, tc.path)

			if tc.expectedErr {
				require.Error(t, err)
				require.Nil(t, signer)
			} else {
				require.NoError(t, err)
				require.NotNil(t, signer)
				require.NotNil(t, signer.Address())
				require.NotNil(t, signer.PublicKey())

				// Dynamically determine the expected address for debugging/verification
				// This might require running the test once and then updating the expectedAddr
				// manually in the testCases struct.
				t.Logf("Mnemonic: %s, Path: %s, Derived Address: %s", tc.mnemonic, tc.path, signer.Address().String())

				// For now, only assert if an expected address is provided
				if tc.expectedAddr != "" {
					require.Equal(t, tc.expectedAddr, signer.Address().String())
				}
			}
		})
	}
}

func TestHDWalletSigner_Sign(t *testing.T) {
	// TODO: Add comprehensive tests for the Sign method once it's implemented.
}

func TestHDWalletSigner_SignMessageV2(t *testing.T) {
	// TODO: Add comprehensive tests for the SignMessageV2 method once it's implemented.
}
