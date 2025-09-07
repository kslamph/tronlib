package signer

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	_ "github.com/ethereum/go-ethereum/common" // Used in SignMessageV2
	_ "github.com/ethereum/go-ethereum/crypto" // Used in SignMessageV2
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

// MockSigner for testing SignTx and SignMessageV2 without actual private keys
type MockSigner struct {
	addr     *types.Address
	pubKey   *ecdsa.PublicKey
	signFunc func([]byte) ([]byte, error)
}

func (m *MockSigner) Address() *types.Address {
	return m.addr
}

func (m *MockSigner) PublicKey() *ecdsa.PublicKey {
	return m.pubKey
}

func (m *MockSigner) Sign(hash []byte) ([]byte, error) {
	if m.signFunc != nil {
		return m.signFunc(hash)
	}
	// Default mock behavior: return a dummy signature
	return []byte("mock_signature"), nil
}

func TestSignTx(t *testing.T) {
	// Setup a mock signer
	mockSigner := &MockSigner{
		addr:   types.MustNewAddressFromBase58("TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu"), // Valid dummy address
		pubKey: &ecdsa.PublicKey{},                                                   // Dummy public key
		signFunc: func(hash []byte) ([]byte, error) {
			// Simulate a successful signing
			return []byte(fmt.Sprintf("signed_%x", hash)), nil
		},
	}

	t.Run("Sign core.Transaction successfully", func(t *testing.T) {
		tx := &core.Transaction{
			RawData: &core.TransactionRaw{
				Timestamp:  time.Now().UnixMilli(),
				Expiration: time.Now().Add(time.Minute).UnixMilli(),
				Contract: []*core.Transaction_Contract{
					{
						Type: core.Transaction_Contract_TransferContract,
						Parameter: &anypb.Any{
							TypeUrl: "/protocol.TransferContract",
							Value:   []byte("some transfer data"),
						},
					},
				},
			},
			Signature: [][]byte{},
		}

		err := SignTx(mockSigner, tx)
		require.NoError(t, err)
		assert.Len(t, tx.Signature, 1)
		assert.Contains(t, string(tx.Signature[0]), "signed_")

		// Verify that the signature is based on the rawData hash
		rawData, _ := proto.Marshal(tx.GetRawData())
		h256h := sha256.New()
		h256h.Write(rawData)
		expectedSignaturePart := fmt.Sprintf("signed_%x", h256h.Sum(nil))
		assert.Equal(t, expectedSignaturePart, string(tx.Signature[0]))
	})

	t.Run("Sign api.TransactionExtention successfully", func(t *testing.T) {
		txExt := &api.TransactionExtention{
			Transaction: &core.Transaction{
				RawData: &core.TransactionRaw{
					Timestamp:  time.Now().UnixMilli(),
					Expiration: time.Now().Add(time.Minute).UnixMilli(),
					Contract: []*core.Transaction_Contract{
						{
							Type: core.Transaction_Contract_TransferContract,
							Parameter: &anypb.Any{
								TypeUrl: "/protocol.TransferContract",
								Value:   []byte("some transfer data extension"),
							},
						},
					},
				},
				Signature: [][]byte{},
			},
		}

		err := SignTx(mockSigner, txExt)
		require.NoError(t, err)
		assert.Len(t, txExt.Transaction.Signature, 1)
		assert.Contains(t, string(txExt.Transaction.Signature[0]), "signed_")

		// Verify that the signature is based on the rawData hash
		rawData, _ := proto.Marshal(txExt.GetTransaction().GetRawData())
		h256h := sha256.New()
		h256h.Write(rawData)
		expectedSignaturePart := fmt.Sprintf("signed_%x", h256h.Sum(nil))
		assert.Equal(t, expectedSignaturePart, string(txExt.Transaction.Signature[0]))
	})

	t.Run("Handle nil transaction", func(t *testing.T) {
		err := SignTx(mockSigner, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "transaction cannot be nil")
	})

	t.Run("Handle signer Sign error", func(t *testing.T) {
		errorSigner := &MockSigner{
			addr: types.MustNewAddressFromBase58("TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu"), // Valid dummy address
			signFunc: func(hash []byte) ([]byte, error) {
				return nil, fmt.Errorf("mock signing error")
			},
		}
		tx := &core.Transaction{RawData: &core.TransactionRaw{}}
		err := SignTx(errorSigner, tx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to sign transaction: mock signing error")
	})

	t.Run("Handle unsupported transaction type", func(t *testing.T) {
		err := SignTx(mockSigner, "unsupported type")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported transaction type: string")
	})
}

func TestSignMessageV2(t *testing.T) {
	// Setup a mock signer
	mockSigner := &MockSigner{
		addr:   types.MustNewAddressFromBase58("TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu"), // Valid dummy address
		pubKey: &ecdsa.PublicKey{},                                                   // Dummy public key
		signFunc: func(hash []byte) ([]byte, error) {
			// Simulate a successful signing by returning a dummy 65-byte signature
			return make([]byte, 65), nil
		},
	}

	t.Run("Sign plain text message successfully", func(t *testing.T) {
		message := "hello world"
		signature, err := SignMessageV2(mockSigner, message)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(signature, "0x")) // Check for "0x" prefix
		assert.Len(t, signature, 132)                      // 0x + 65 bytes * 2 = 130 hex chars = 132 length
	})

	t.Run("Sign hex message successfully", func(t *testing.T) {
		message := "0x68656c6c6f" // "hello" in hex
		signature, err := SignMessageV2(mockSigner, message)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(signature, "0x")) // Check for "0x" prefix
		assert.Len(t, signature, 132)                      // 0x + 65 bytes * 2 = 130 hex chars = 132 length
	})

	t.Run("Handle signer Sign error in message signing", func(t *testing.T) {
		errorSigner := &MockSigner{
			addr: types.MustNewAddressFromBase58("TQJ6R9SPvD5SyqgYqTBq3yc6mFtEgatPDu"), // Valid dummy address
			signFunc: func(hash []byte) ([]byte, error) {
				return nil, fmt.Errorf("mock message signing error")
			},
		}
		message := "test"
		signature, err := SignMessageV2(errorSigner, message)
		require.Error(t, err)
		assert.Empty(t, signature)
		assert.Contains(t, err.Error(), "failed to sign message: mock message signing error")
	})
}
