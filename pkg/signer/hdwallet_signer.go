package signer

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/bip39-hdwallet/bip39"
	"github.com/kslamph/bip39-hdwallet/hdwallet"
	"github.com/kslamph/tronlib/pkg/types"
)

// HDWalletSigner implements the Signer interface using an HD wallet.
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

// Sign signs a given hash using the HD wallet's private key and returns the raw signature bytes.
// This method implements the Signer interface.
func (s *HDWalletSigner) Sign(hash []byte) ([]byte, error) {
	// Sign the hash
	signature, err := crypto.Sign(hash, s.privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %w", err)
	}
	return signature, nil
}
