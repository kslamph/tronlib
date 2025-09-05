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
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/protobuf/proto"

	"github.com/kslamph/tronlib/pkg/types"
)

// Constants for message signing
const (
	// TronWeb message prefix (v1)
	TronMessagePrefix = "\x19TRON Signed Message:\n"
)

// PrivateKeySigner implements the Signer interface using a private key.
//
// The PrivateKeySigner allows you to sign transactions and messages using a
// private key. It automatically derives the corresponding public key and address.
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
//   signer, err := signer.NewPrivateKeySigner("0xYourPrivateKeyHere")
//   if err != nil {
//       // handle error
//   }
//   
//   // Get the address associated with this private key
//   address := signer.Address()
//   fmt.Printf("Address: %s\n", address.String())
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
//   signer, _ := signer.NewPrivateKeySigner("0xYourPrivateKeyHere")
//   address := signer.Address()
//   fmt.Printf("Address: %s\n", address.String())
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

// Sign signs a transaction using the private key.
//
// This method signs either a *core.Transaction or *api.TransactionExtention
// using the private key. The signature is appended to the transaction's
// Signature field.
//
// Example:
//   signer, _ := signer.NewPrivateKeySigner("0xYourPrivateKeyHere")
//   err := signer.Sign(transaction)
//   if err != nil {
//       // handle error
//   }
func (s *PrivateKeySigner) Sign(tx any) error {

	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}
	switch t := tx.(type) {

	case *core.Transaction:

		rawData, err := proto.Marshal(t.GetRawData())
		if err != nil {
			return fmt.Errorf("failed to marshal transaction: %w", err)
		}
		// Calculate hash
		h256h := sha256.New()
		h256h.Write(rawData)
		hash := h256h.Sum(nil)

		// Sign the hash
		signature, err := crypto.Sign(hash, s.privKey)
		if err != nil {
			return fmt.Errorf("failed to sign transaction: %w", err)
		}

		t.Signature = append(t.Signature, signature)

	case *api.TransactionExtention:
		rawData, err := proto.Marshal(t.GetTransaction().GetRawData())
		if err != nil {
			return fmt.Errorf("failed to marshal transaction: %w", err)
		}
		// Calculate hash
		h256h := sha256.New()
		h256h.Write(rawData)
		hash := h256h.Sum(nil)

		// Sign the hash
		signature, err := crypto.Sign(hash, s.privKey)
		if err != nil {
			return fmt.Errorf("failed to sign transaction: %w", err)
		}

		t.Transaction.Signature = append(t.Transaction.Signature, signature)
	default:
		return fmt.Errorf("unsupported transaction type: %T", tx)
	}

	return nil
}

// SignMessageV2 signs a message using TIP-191 format (v2)
func (s *PrivateKeySigner) SignMessageV2(message string) (string, error) {
	var data []byte
	if strings.HasPrefix(message, "0x") {
		// Assume hex-encoded string
		data = common.FromHex(message)
	} else {
		data = []byte(message)
	}

	// Prefix the message
	messageLen := len(data)
	prefixedMessage := []byte(fmt.Sprintf("%s%d%s", TronMessagePrefix, messageLen, string(data)))

	// Hash the prefixed message (Keccak256)
	hash := crypto.Keccak256Hash(prefixedMessage)

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), s.privKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	// Adjust the recovery ID (v). go-ethereum's Sign function returns
	// the signature in [R || S || V] format, where V is 0 or 1. Tron
	// expects V to be 27 or 28, so we add 27.
	signature[64] += 27

	// Return the hex-encoded signature
	return "0x" + common.Bytes2Hex(signature), nil
}
