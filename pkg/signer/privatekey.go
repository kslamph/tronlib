package signer

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/kslamph/tronlib/pkg/types"
)

// PrivateKeySigner implements the Signer interface using a private key.
// It automatically derives the corresponding public key and address.
type PrivateKeySigner struct {
	address *types.Address
	privKey *ecdsa.PrivateKey
	pubKey  *ecdsa.PublicKey
}

// NewPrivateKeySigner creates a new PrivateKeySigner from a hex private key.
//
// This function creates a signer from a hexadecimal private key string. The
// private key can be provided with or without the "0x" prefix.
//
// Example:
//
//	signer, err := signer.NewPrivateKeySigner("0xYourPrivateKeyHere")
//	if err != nil {
//	    // handle error
//	}
//
//	// Get the address associated with this private key
//	address := signer.Address()
//	fmt.Printf("Address: %s\n", address.String())
func NewPrivateKeySigner(hexPrivKey string) (*PrivateKeySigner, error) {
	// Remove 0x prefix if present
	// if strings.HasPrefix(hexPrivKey, "0x") {
	// 	hexPrivKey = hexPrivKey[2:]
	// }
	hexPrivKey = strings.TrimPrefix(hexPrivKey, "0x")

	// Decode and validate private key
	key, err := hex.DecodeString(hexPrivKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex private key: %w", err)
	}

	privKey, err := crypto.ToECDSA(key)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return newPrivateKeySigner(privKey)
}

// NewPrivateKeySignerFromECDSA creates a new PrivateKeySigner from an ECDSA private key
func NewPrivateKeySignerFromECDSA(privKey *ecdsa.PrivateKey) (*PrivateKeySigner, error) {
	return newPrivateKeySigner(privKey)
}

// newPrivateKeySigner creates a new PrivateKeySigner from a private key
func newPrivateKeySigner(privKey *ecdsa.PrivateKey) (*PrivateKeySigner, error) {
	pubKey := privKey.PublicKey
	ethAddr := crypto.PubkeyToAddress(pubKey)

	// Add TRON prefix (0x41)
	tronBytes := append([]byte{0x41}, ethAddr.Bytes()...)

	// Convert to base58 address
	tronAddr, err := types.NewAddressFromBytes(tronBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create tron address: %w", err)
	}

	return &PrivateKeySigner{
		address: tronAddr,
		privKey: privKey,
		pubKey:  &pubKey,
	}, nil
}

// Address returns the account's address.
//
// This method returns the TRON address associated with the private key.
//
// Example:
//
//	signer, _ := signer.NewPrivateKeySigner("0xYourPrivateKeyHere")
//	address := signer.Address()
//	fmt.Printf("Address: %s\n", address.String())
func (s *PrivateKeySigner) Address() *types.Address {
	return s.address
}

// PublicKey returns the account's public key
func (s *PrivateKeySigner) PublicKey() *ecdsa.PublicKey {
	return s.pubKey
}

// PrivateKeyHex returns the account's private key in hex format
func (s *PrivateKeySigner) PrivateKeyHex() string {
	privateKeyBytes := crypto.FromECDSA(s.privKey)
	return hex.EncodeToString(privateKeyBytes)
}

// Sign signs a given hash using the private key and returns the raw signature bytes.
// This method implements the Signer interface.
func (s *PrivateKeySigner) Sign(hash []byte) ([]byte, error) {
	// Sign the hash
	signature, err := crypto.Sign(hash, s.privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %w", err)
	}
	return signature, nil
}
