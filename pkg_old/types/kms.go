package types

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/protobuf/proto"
)

// KMSClientInterface defines the interface for KMS operations
type KMSClientInterface interface {
	// SignDigest signs a digest using the key identified by keyID
	SignDigest(keyID string, digest []byte) ([]byte, error)

	// GetPublicKey retrieves the public key for the given keyID
	GetPublicKey(keyID string) (*ecdsa.PublicKey, error)
}

// KMSAccount represents a Tron account where signing operations are performed by a KMS
type KMSAccount struct {
	address   *Address
	pubKey    *ecdsa.PublicKey
	kmsKeyID  string
	kmsClient KMSClientInterface
}

// NewKMSAccount creates a new KMSAccount instance
func NewKMSAccount(keyID string, client KMSClientInterface) (*KMSAccount, error) {
	// Get the public key from KMS
	pubKey, err := client.GetPublicKey(keyID)
	if err != nil {
		return nil, err
	}

	// Convert public key to Tron address (similar to newAccount)
	ethAddr := crypto.PubkeyToAddress(*pubKey)
	tronBytes := append([]byte{0x41}, ethAddr.Bytes()...)

	// Create Tron address
	tronAddr, err := NewAddressFromBytes(tronBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create tron address: %w", err)
	}

	return &KMSAccount{
		address:   tronAddr,
		pubKey:    pubKey,
		kmsKeyID:  keyID,
		kmsClient: client,
	}, nil
}

// Address implements Signer interface
func (ka *KMSAccount) Address() *Address {
	return ka.address
}

// PublicKey implements Signer interface
func (ka *KMSAccount) PublicKey() *ecdsa.PublicKey {
	return ka.pubKey
}

// Sign implements Signer interface
func (ka *KMSAccount) Sign(tx *core.Transaction) (*core.Transaction, error) {
	return ka.MultiSign(tx, 2)
}

// MultiSign implements Signer interface
func (ka *KMSAccount) MultiSign(tx *core.Transaction, permissionID int32) (*core.Transaction, error) {
	// Set permission ID
	tx.GetRawData().GetContract()[0].PermissionId = permissionID

	// Marshal raw data for signing
	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// Calculate hash
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)

	// Sign the hash using KMS
	signature, err := ka.kmsClient.SignDigest(ka.kmsKeyID, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction with KMS: %w", err)
	}

	// Clear any existing signatures and add the new one
	tx.Signature = nil
	tx.Signature = append(tx.Signature, signature)

	return tx, nil
}

// SignMessageV2 implements Signer interface
func (ka *KMSAccount) SignMessageV2(message string) (string, error) {
	var data []byte
	if strings.HasPrefix(message, "0x") {
		data = common.FromHex(message)
	} else {
		data = []byte(message)
	}

	// Prefix the message
	messageLen := len(data)
	prefixedMessage := []byte(fmt.Sprintf("%s%d%s", TronMessagePrefix, messageLen, string(data)))

	// Hash the prefixed message
	hash := crypto.Keccak256Hash(prefixedMessage)

	// Sign the hash using KMS
	signature, err := ka.kmsClient.SignDigest(ka.kmsKeyID, hash.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to sign message with KMS: %w", err)
	}

	// Adjust the recovery ID (v) as needed for Tron compatibility
	signature[64] += 27

	// Return hex-encoded signature
	return "0x" + common.Bytes2Hex(signature), nil
}
