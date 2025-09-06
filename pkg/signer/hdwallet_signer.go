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
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/bip39-hdwallet/bip39"
	"github.com/kslamph/bip39-hdwallet/hdwallet"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/protobuf/proto"
)

// HDWalletSigner implements the Signer interface using a HD wallet
type HDWalletSigner struct {
	mnemonic string
	path     string
	privKey  *ecdsa.PrivateKey
	address  *types.Address
}

// NewHDWalletSigner creates a new HDWalletSigner from a mnemonic and derivation path.
func NewHDWalletSigner(mnemonic, passphrase, path string) (*HDWalletSigner, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	seed := bip39.NewSeed(mnemonic, passphrase) // No password for now
	masterKey, err := hdwallet.NewMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	wallet, err := masterKey.DerivePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to derive path: %w", err)
	}

	privKey, err := wallet.ToECDSA() // Returns *ecdsa.PrivateKey, error
	if err != nil {
		return nil, fmt.Errorf("failed to get private key from wallet: %w", err)
	}

	address, err := types.NewAddressFromEVM(crypto.PubkeyToAddress(privKey.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create tron address from EVM address: %w", err)
	}

	return &HDWalletSigner{
		mnemonic: mnemonic,
		path:     path,
		privKey:  privKey,
		address:  address,
	}, nil
}

// Address returns the account's address.
func (s *HDWalletSigner) Address() *types.Address {
	return s.address
}

// PublicKey returns the account's public key.
func (s *HDWalletSigner) PublicKey() *ecdsa.PublicKey {
	return &s.privKey.PublicKey
}

// Sign signs a transaction.
func (s *HDWalletSigner) Sign(tx any) error {
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

// SignMessageV2 signs a message using TIP-191 format (v2).
func (s *HDWalletSigner) SignMessageV2(message string) (string, error) {
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
