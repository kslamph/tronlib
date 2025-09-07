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
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/kslamph/tronlib/pb/core"
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
			name:         "Valid Mnemonic with Passphrase - Path 0",
			mnemonic:     "rebel move punch grant loop beyond stadium dumb appear enough typical remind",
			path:         "m/44'/195'/0'/0/0",
			expectedErr:  false,
			expectedAddr: "TGhQ5hFFS5aWkdr9q2RhpeHr7BgpNFRwr8",
		},
		{
			name:         "Valid Mnemonic with Passphrase - Path 17",
			mnemonic:     "rebel move punch grant loop beyond stadium dumb appear enough typical remind",
			path:         "m/44'/195'/20'/0/17",
			expectedErr:  false,
			expectedAddr: "TCTFaDJSfi3ou4tJp3zYTfEVhiNcMaL91v",
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
			passphrase := ""
			// Apply passphrase only for relevant test cases
			if tc.name == "Valid Mnemonic with Passphrase - Path 0" || tc.name == "Valid Mnemonic with Passphrase - Path 17" {
				passphrase = "TronLib"
			}

			signer, err := NewHDWalletSigner(tc.mnemonic, passphrase, tc.path)

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
	mnemonic := "rebel move punch grant loop beyond stadium dumb appear enough typical remind"
	passphrase := "TronLib"
	path := "m/44'/195'/0'/0/0"

	signer, err := NewHDWalletSigner(mnemonic, passphrase, path)
	require.NoError(t, err)
	require.NotNil(t, signer)

	// Create a valid *core.Transaction for testing
	// This is a minimal example of a TransferContract
	// Use a fixed timestamp for deterministic testing
	fixedTimestamp := int64(1678886400000) // March 15, 2023 12:00:00 AM GMT in milliseconds

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{ // Corrected: Use TransactionRaw
			Timestamp:  fixedTimestamp,
			Expiration: fixedTimestamp + (60 * 1000), // Expiration 60 seconds after timestamp
			FeeLimit:   1_000_000,                    // Example fee limit
			Contract: []*core.Transaction_Contract{
				{
					Type: core.Transaction_Contract_TransferContract,
					Parameter: &anypb.Any{
						TypeUrl: "/protocol.TransferContract",
						Value:   []byte("some transfer contract data"), // Placeholder, ideally a marshaled TransferContract
					},
				},
			},
		},
		Signature: make([][]byte, 0), // Initialize empty signature slice
	}

	// Calculate hash of the raw transaction data
	rawData, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err)
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)

	// Sign the hash
	signature, err := signer.Sign(hash)
	require.NoError(t, err)
	require.NotEmpty(t, signature)

	// Attach the signature for further verification if needed, though the unit test focuses on signer.Sign
	tx.Signature = append(tx.Signature, signature)

	// The original expected signature logic (if correct for the given hash) should still pass.
	// Note: You might need to re-generate this expected signature based on the new Sign implementation logic.
	expectedSignature := []byte{0x20, 0x8a, 0x02, 0x7f, 0xc1, 0xf3, 0x65, 0x60, 0x0a, 0xe3, 0xc0, 0xb4, 0x7e, 0x39, 0xa7, 0x76, 0x8f, 0x19, 0x0f, 0xd8, 0x2e, 0x3b, 0x3d, 0x0a, 0xd1, 0x09, 0xe6, 0x65, 0x43, 0x24, 0x6e, 0xc3, 0x55, 0x04, 0xa4, 0x4c, 0x19, 0x19, 0xeb, 0xfb, 0x3a, 0xa4, 0x5f, 0x77, 0x9e, 0xda, 0x2a, 0xc4, 0x0e, 0xc3, 0x91, 0x10, 0xc3, 0x22, 0x5e, 0xc1, 0x03, 0x3e, 0xc0, 0x99, 0xea, 0x06, 0x61, 0xd1, 0x00}
	require.Equal(t, expectedSignature, signature) // Compare directly with the returned signature

}

func TestHDWalletSigner_SignMessageV2(t *testing.T) {
	mnemonic := "rebel move punch grant loop beyond stadium dumb appear enough typical remind"
	passphrase := "TronLib"
	path := "m/44'/195'/0'/0/0"
	message := "test message for signing"

	signer, err := NewHDWalletSigner(mnemonic, passphrase, path)
	require.NoError(t, err)
	require.NotNil(t, signer)

	signature, err := SignMessageV2(signer, message) // Use the package-level SignMessageV2
	require.NoError(t, err)
	require.NotEmpty(t, signature)

	expectedSignature := "0xaab0b6b3a50617691db46fac2bf9918176c5a11c69f4feb3a721a9ebd0b28640465a8188bcf8d37d981a9ef11a918237dd33fdc9e70b072003558c3869a9c92e1c"
	require.Equal(t, expectedSignature, signature)
	require.True(t, len(signature) == 132) // 0x + 130 hex chars (65 bytes * 2)
}
