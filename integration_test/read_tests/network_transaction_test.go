package read_tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainnetGetTransactionById tests the GetTransactionById API with real transaction data
func TestMainnetGetTransactionById(t *testing.T) {
	manager := setupNetworkTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a known successful transaction ID from mainnet
	// This is a transfer transaction from the integration tests
	succTx := "f079d362f06b496cd22ccb9ec54d8c5bf0ef734e47613dc4caa76e4eb118f5a9"

	t.Run("GetTransactionById_SuccessfulTransaction", func(t *testing.T) {
		txId := succTx
		tx, err := manager.GetTransactionById(ctx, txId)
		require.NoError(t, err, "GetTransactionById should succeed for successful transaction")
		require.NotNil(t, tx, "Transaction should not be nil")

		// Validate basic transaction structure
		require.NotNil(t, tx, "Transaction should not be nil")

		// Validate raw data exists
		rawData := tx.GetRawData()
		require.NotNil(t, rawData, "Raw data should not be nil")

		// Validate contracts
		contracts := rawData.GetContract()
		assert.Greater(t, len(contracts), 0, "Transaction should have at least one contract")

		for i, contract := range contracts {
			contractType := contract.GetType()
			assert.NotEqual(t, core.Transaction_Contract_ContractType(0), contractType, "Contract type should be valid")
			t.Logf("Contract %d type: %v", i, contractType)

			parameter := contract.GetParameter()
			if parameter != nil {
				typeUrl := parameter.GetTypeUrl()
				assert.NotEmpty(t, typeUrl, "Parameter type URL should not be empty")
				t.Logf("Contract %d parameter type: %s", i, typeUrl)
			}
		}

		// Validate timestamps
		timestamp := rawData.GetTimestamp()
		assert.Greater(t, timestamp, int64(0), "Transaction timestamp should be positive")

		expiration := rawData.GetExpiration()
		assert.Greater(t, expiration, timestamp, "Expiration should be after timestamp")

		// Validate ref block data
		refBlockBytes := rawData.GetRefBlockBytes()
		assert.NotEmpty(t, refBlockBytes, "Ref block bytes should not be empty")

		refBlockHash := rawData.GetRefBlockHash()
		assert.NotEmpty(t, refBlockHash, "Ref block hash should not be empty")

		// Validate signatures
		signatures := tx.GetSignature()
		assert.Greater(t, len(signatures), 0, "Transaction should have at least one signature")
		for i, sig := range signatures {
			assert.Len(t, sig, 65, "Signature should be 65 bytes")
			t.Logf("Signature %d length: %d bytes", i, len(sig))
		}

		// Validate transaction results
		ret := tx.GetRet()
		if len(ret) > 0 {
			for i, result := range ret {
				contractRet := result.GetContractRet()
				t.Logf("Result %d: %v", i, contractRet)
			}
		}

		t.Logf("✅ Transaction validation passed")
	})

	t.Run("GetTransactionById_InputValidation", func(t *testing.T) {
		// Test empty transaction ID
		_, err := manager.GetTransactionById(ctx, "")
		assert.Error(t, err, "Should reject empty transaction ID")
		assert.Contains(t, err.Error(), "cannot be empty", "Error should mention empty ID")

		// Test invalid hex characters
		_, err = manager.GetTransactionById(ctx, "invalid_hex_characters_here_not_valid_transaction_id_format")
		assert.Error(t, err, "Should reject invalid hex characters")
		// This could be either "invalid hex" or length error - both are acceptable
		assert.True(t,
			strings.Contains(err.Error(), "invalid hex") || strings.Contains(err.Error(), "64 hex characters"),
			"Error should mention invalid hex or length requirement, got: %s", err.Error())

		// Test wrong length
		_, err = manager.GetTransactionById(ctx, "1234567890abcdef") // Too short
		assert.Error(t, err, "Should reject wrong length transaction ID")
		assert.Contains(t, err.Error(), "64 hex characters", "Error should mention correct length requirement")

		// Test with 0x prefix (should be accepted and stripped)
		txId := "0x" + succTx
		tx, err := manager.GetTransactionById(ctx, txId)
		require.NoError(t, err, "Should accept transaction ID with 0x prefix")
		require.NotNil(t, tx, "Should return valid transaction")

		// Test with 0X prefix (should be accepted and stripped)
		txId = "0X" + succTx
		tx, err = manager.GetTransactionById(ctx, txId)
		require.NoError(t, err, "Should accept transaction ID with 0X prefix")
		require.NotNil(t, tx, "Should return valid transaction")

		t.Logf("✅ Input validation working correctly")
	})

	t.Run("GetTransactionById_NonexistentTransaction", func(t *testing.T) {
		// Test with a valid format but nonexistent transaction ID
		nonexistentTxId := "11000000000000000000000000000000000000000000000000000000000000ff"
		tx, err := manager.GetTransactionById(ctx, nonexistentTxId)
		// This should either return an error or return nil - both are acceptable
		// The important thing is that it doesn't crash
		require.NoError(t, err, "Should not return error for nonexistent transaction")
		// For nonexistent transaction, we expect either nil or empty transaction
		if tx != nil {
			rawData := tx.GetRawData()
			if rawData != nil {
				// Check if contract is empty (which would indicate no valid transaction)
				contracts := rawData.GetContract()
				assert.Empty(t, contracts, "Should have no contracts for nonexistent transaction")
			}
		}
		t.Logf("Nonexistent transaction handling passed for ID: %s", nonexistentTxId)

		t.Logf("✅ Nonexistent transaction handling passed")
	})
}
