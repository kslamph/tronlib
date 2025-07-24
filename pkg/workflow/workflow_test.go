package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"google.golang.org/protobuf/types/known/anypb"
)

// Mock client for testing
type mockClient struct{}

func (m *mockClient) BroadcastTransaction(ctx context.Context, tx *core.Transaction) error {
	return nil
}

func createMockClient() *client.Client {
	// Return nil for now since we're not actually using the client in these tests
	return nil
}

func createTestTransaction() *core.Transaction {
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type: core.Transaction_Contract_TransferContract,
					Parameter: &anypb.Any{
						Value: []byte("test_data"),
					},
				},
			},
			Timestamp:     time.Now().UnixMilli(),
			Expiration:    time.Now().Add(time.Hour).UnixMilli(),
			RefBlockBytes: []byte{0x12, 0x34},
			RefBlockHash:  []byte{0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34},
		},
	}
}

func createTestSigner() *signer.PrivateKeySigner {
	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	s, _ := signer.NewPrivateKeySigner(privateKey)
	return s
}

func TestNewTransactionWorkflow(t *testing.T) {
	client := createMockClient()
	tx := createTestTransaction()

	workflow := NewTransactionWorkflow(client, tx)
	if workflow == nil {
		t.Error("Expected workflow to be created, got nil")
	}

	if workflow.GetState() != StateUnsigned {
		t.Errorf("Expected initial state to be StateUnsigned, got %v", workflow.GetState())
	}

	if workflow.GetError() != nil {
		t.Errorf("Expected no error on creation, got %v", workflow.GetError())
	}
}

func TestWorkflowStateTransitions(t *testing.T) {
	client := createMockClient()
	tx := createTestTransaction()
	signer := createTestSigner()

	workflow := NewTransactionWorkflow(client, tx)

	// Test unsigned state operations
	workflow.SetTimeout(time.Now().Add(1 * time.Hour).UnixMilli())
	if workflow.GetError() != nil {
		t.Errorf("SetTimeout should work on unsigned transaction: %v", workflow.GetError())
	}

	workflow.SetFeeLimit(1000000)
	if workflow.GetError() != nil {
		t.Errorf("SetFeeLimit should work on unsigned transaction: %v", workflow.GetError())
	}

	// Test signing
	workflow.Sign(signer)
	if workflow.GetError() != nil {
		t.Errorf("Sign should work: %v", workflow.GetError())
	}

	if workflow.GetState() != StateSigned {
		t.Errorf("Expected state to be StateSigned after signing, got %v", workflow.GetState())
	}

	// Test that unsigned operations fail on signed transaction
	workflow.SetTimeout(time.Now().Add(2 * time.Hour).UnixMilli())
	if workflow.GetError() == nil {
		t.Error("SetTimeout should fail on signed transaction")
	}
}

func TestGetTxid(t *testing.T) {
	client := createMockClient()
	tx := createTestTransaction()
	signer := createTestSigner()

	workflow := NewTransactionWorkflow(client, tx)

	// Should return empty string for unsigned transaction
	txID := workflow.GetTxid()
	if txID != "" {
		t.Errorf("Expected empty txID for unsigned transaction, got %s", txID)
	}

	// Sign the transaction
	workflow.Sign(signer)
	if workflow.GetError() != nil {
		t.Fatalf("Failed to sign transaction: %v", workflow.GetError())
	}

	// Should return non-empty txID for signed transaction
	txID = workflow.GetTxid()
	if txID == "" {
		t.Error("Expected non-empty txID for signed transaction")
	}
}

func TestGetSignedTransaction(t *testing.T) {
	client := createMockClient()
	tx := createTestTransaction()
	signer := createTestSigner()

	workflow := NewTransactionWorkflow(client, tx)

	// Should fail for unsigned transaction
	_, _, err := workflow.GetSignedTransaction()
	if err == nil {
		t.Error("GetSignedTransaction should fail for unsigned transaction")
	}

	// Sign the transaction
	workflow.Sign(signer)
	if workflow.GetError() != nil {
		t.Fatalf("Failed to sign transaction: %v", workflow.GetError())
	}

	// Should succeed for signed transaction
	txID, signedTx, err := workflow.GetSignedTransaction()
	if err != nil {
		t.Errorf("GetSignedTransaction should succeed for signed transaction: %v", err)
	}

	if txID == "" {
		t.Error("Expected non-empty txID")
	}

	if signedTx == nil {
		t.Error("Expected non-nil signed transaction")
	}

	if signedTx.Transaction == nil {
		t.Error("Expected non-nil transaction in signed transaction")
	}
}

func TestErrorHandling(t *testing.T) {
	client := createMockClient()

	// Test with invalid transaction (nil raw data)
	invalidTx := &core.Transaction{
		RawData: nil,
	}

	workflow := NewTransactionWorkflow(client, invalidTx)
	if workflow.GetError() == nil {
		t.Error("Expected error for invalid transaction")
	}

	if workflow.GetState() != StateError {
		t.Errorf("Expected state to be StateError, got %v", workflow.GetState())
	}

	// Test with nil transaction
	workflow2 := NewTransactionWorkflow(client, nil)
	if workflow2.GetError() == nil {
		t.Error("Expected error for nil transaction")
	}
}

func TestMultiSignature(t *testing.T) {
	client := createMockClient()
	tx := createTestTransaction()
	signer1 := createTestSigner()

	privateKey2 := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	signer2, err := signer.NewPrivateKeySigner(privateKey2)
	if err != nil {
		t.Fatalf("Failed to create second signer: %v", err)
	}

	workflow := NewTransactionWorkflow(client, tx)

	// Sign with first signer
	workflow.Sign(signer1)
	if workflow.GetError() != nil {
		t.Fatalf("Failed to sign with first signer: %v", workflow.GetError())
	}

	// Multi-sign with second signer
	workflow.MultiSign(signer2, 1)
	if workflow.GetError() != nil {
		t.Fatalf("Failed to multi-sign with second signer: %v", workflow.GetError())
	}

	// Should still be in signed state
	if workflow.GetState() != StateSigned {
		t.Errorf("Expected state to be StateSigned after multi-sign, got %v", workflow.GetState())
	}

	// Should have a valid transaction ID
	txID := workflow.GetTxid()
	if txID == "" {
		t.Error("Expected non-empty txID after multi-sign")
	}
}

func TestChainedOperations(t *testing.T) {
	client := createMockClient()
	tx := createTestTransaction()
	signer := createTestSigner()

	// Test method chaining
	workflow := NewTransactionWorkflow(client, tx).
		SetTimeout(time.Now().Add(1*time.Hour).UnixMilli()).
		SetFeeLimit(1000000).
		Sign(signer)

	if workflow.GetError() != nil {
		t.Errorf("Chained operations should succeed: %v", workflow.GetError())
	}

	if workflow.GetState() != StateSigned {
		t.Errorf("Expected state to be StateSigned after chained operations, got %v", workflow.GetState())
	}

	// Verify the transaction has the expected properties
	txID := workflow.GetTxid()
	if txID == "" {
		t.Error("Expected non-empty txID after chained operations")
	}
}