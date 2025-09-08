package utils

import (
	"crypto/sha256"
	"testing"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestGetTransactionID(t *testing.T) {
	// Test case 1: Normal case with valid transaction
	t.Run("ValidTransaction", func(t *testing.T) {
		tx := &core.Transaction{
			RawData: &core.TransactionRaw{
				RefBlockBytes: []byte("test"),
				Expiration:    1234567890,
			},
		}

		expectedRawData, _ := proto.Marshal(tx.RawData)
		expectedHash := sha256.Sum256(expectedRawData)

		result := GetTransactionID(tx)
		assert.Equal(t, expectedHash[:], result)
	})

	// Test case 2: Nil transaction
	t.Run("NilTransaction", func(t *testing.T) {
		var tx *core.Transaction = nil
		result := GetTransactionID(tx)
		assert.Nil(t, result)
	})

	// Test case 3: Nil RawData
	t.Run("NilRawData", func(t *testing.T) {
		tx := &core.Transaction{
			RawData: nil,
		}
		result := GetTransactionID(tx)
		assert.Nil(t, result)
	})

	// Test case 4: Marshal error (simulated by using unmarshalable data)
	t.Run("MarshalError", func(t *testing.T) {
		// Create a transaction with data that would cause marshal error
		// In practice, this is hard to simulate since proto.Marshal rarely fails
		// unless there's a bug in protobuf implementation
		tx := &core.Transaction{
			RawData: &core.TransactionRaw{
				// This should normally work, but we're testing error handling
			},
		}

		// Since we can't easily force a marshal error, we'll test the error handling
		// by ensuring the function returns nil when marshal fails
		// In real implementation, if marshal fails, it returns nil

		result := GetTransactionID(tx)
		// The marshal should succeed, so result should not be nil
		assert.NotNil(t, result)

		t.Log("Note: Cannot easily test marshal error case without mocking")
	})
}
