package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
)

// MockKMSClient implements KMSClientInterface for demonstration
type MockKMSClient struct {
	keys sync.Map // thread-safe map[string]*ecdsa.PrivateKey
}

// NewMockKMSClient creates a new mock KMS client
func NewMockKMSClient() *MockKMSClient {
	return &MockKMSClient{}
}

// CreateKey generates a new key pair and stores it in the mock KMS
func (m *MockKMSClient) CreateKey() (string, error) {
	// Generate a new private key
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Generate a key ID (in real KMS this would be provided by the service)
	keyID := fmt.Sprintf("key-%x", privKey.Public())
	m.keys.Store(keyID, privKey)

	return keyID, nil
}

// GetPublicKey implements KMSClientInterface
func (m *MockKMSClient) GetPublicKey(keyID string) (*ecdsa.PublicKey, error) {
	key, ok := m.keys.Load(keyID)
	if !ok {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	privKey := key.(*ecdsa.PrivateKey)
	return &privKey.PublicKey, nil
}

// SignDigest implements KMSClientInterface
func (m *MockKMSClient) SignDigest(keyID string, digest []byte) ([]byte, error) {
	key, ok := m.keys.Load(keyID)
	if !ok {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	privKey := key.(*ecdsa.PrivateKey)

	// Sign the digest
	signature, err := crypto.Sign(digest, privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign digest: %w", err)
	}

	return signature, nil
}
