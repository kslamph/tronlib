package signer

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/kslamph/tronlib/pb/core"
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

func TestPrivateKeySigner_Sign(t *testing.T) {
	privateKey := "f8c6f45b2aa8b68ab5f3910bdeb5239428b731618113e2881f46e374bf796b02"

	signer, err := NewPrivateKeySigner(privateKey)
	require.NoError(t, err)
	require.NotNil(t, signer)

	// Create a minimal *core.Transaction for testing
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
						Value:   []byte("some transfer contract data"), // Placeholder
					},
				},
			},
		},
		Signature: make([][]byte, 0),
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
	expectedSignature := []byte{0x3f, 0x39, 0xec, 0xd2, 0x72, 0xe7, 0x5a, 0xde, 0x1e, 0x05, 0x84, 0xd5, 0xb2, 0x0a, 0xb6, 0x0b, 0xa5, 0x3b, 0x00, 0x9f, 0xbe, 0x8c, 0x3c, 0x95, 0xee, 0x4b, 0x81, 0xee, 0x32, 0xea, 0xa2, 0x80, 0x13, 0x43, 0x37, 0xa4, 0xaa, 0x88, 0xc6, 0xc9, 0x59, 0x8f, 0x1a, 0xec, 0x4c, 0x1e, 0xe7, 0xcb, 0x8a, 0x3c, 0x38, 0xb4, 0xad, 0x9f, 0x73, 0xdc, 0xfa, 0xbc, 0x02, 0x9c, 0x26, 0xf6, 0xbe, 0x38, 0x00}
	require.Equal(t, expectedSignature, signature) // Compare directly with the returned signature

}

func TestPrivateKeySigner_SignMessageV2(t *testing.T) {
	// Test data migrated from pkg_old/crypto/verify_message_v2_test.go
	privateKey := "f8c6f45b2aa8b68ab5f3910bdeb5239428b731618113e2881f46e374bf796b02"
	message := "sign message testing"
	expectedSignature := "0x88bacb8549cbe7c3e26d922b05e88757197b77410fb0db1fabb9f30480202c84691b7025e928d36be962cfd7b4a8d2353b97f36d64bdc14398e9568091b701201b"

	signer, err := NewPrivateKeySigner(privateKey)
	require.NoError(t, err)

	signature, err := SignMessageV2(signer, message) // Use the package-level SignMessageV2
	require.NoError(t, err)

	assert.Equal(t, expectedSignature, signature)
	assert.True(t, len(signature) == 132) // 0x + 130 hex chars (65 bytes * 2)
}
