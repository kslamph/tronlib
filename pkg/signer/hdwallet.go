package signer

import (
	"crypto/ecdsa"
	"fmt"

	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

// HDWalletSigner implements the Signer interface using HD wallet
type HDWalletSigner struct {
	*PrivateKeySigner // Embed PrivateKeySigner for implementation
	wallet            *hdwallet.Wallet
	derivationPath    string
}

// NewHDWalletSigner creates a new HDWalletSigner from mnemonic and derivation path
func NewHDWalletSigner(mnemonic string, passphrase string, path string) (*HDWalletSigner, error) {
	if path == "" {
		path = "m/44'/195'/0'/0/0" // Default BIP44 path for TRON
	}

	wallet, err := hdwallet.NewFromMnemonic(mnemonic, passphrase)
	if err != nil {
		return nil, fmt.Errorf("invalid mnemonic: %w", err)
	}

	derivationPath, err := hdwallet.ParseDerivationPath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid derivation path: %w", err)
	}

	account, err := wallet.Derive(derivationPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account: %w", err)
	}

	privKey, err := wallet.PrivateKey(account)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	// Create the underlying PrivateKeySigner
	privateKeySigner, err := newPrivateKeySigner(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key signer: %w", err)
	}

	return &HDWalletSigner{
		PrivateKeySigner: privateKeySigner,
		wallet:           wallet,
		derivationPath:   path,
	}, nil
}

// NewHDWalletSignerFromSeed creates a new HDWalletSigner from seed bytes
func NewHDWalletSignerFromSeed(seed []byte, path string) (*HDWalletSigner, error) {
	if path == "" {
		path = "m/44'/195'/0'/0/0" // Default BIP44 path for TRON
	}

	wallet, err := hdwallet.NewFromSeed(seed)
	if err != nil {
		return nil, fmt.Errorf("invalid seed: %w", err)
	}

	derivationPath, err := hdwallet.ParseDerivationPath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid derivation path: %w", err)
	}

	account, err := wallet.Derive(derivationPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account: %w", err)
	}

	privKey, err := wallet.PrivateKey(account)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	// Create the underlying PrivateKeySigner
	privateKeySigner, err := newPrivateKeySigner(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key signer: %w", err)
	}

	return &HDWalletSigner{
		PrivateKeySigner: privateKeySigner,
		wallet:           wallet,
		derivationPath:   path,
	}, nil
}

// DerivationPath returns the derivation path used
func (s *HDWalletSigner) DerivationPath() string {
	return s.derivationPath
}

// DeriveAccount derives a new account at the given index
func (s *HDWalletSigner) DeriveAccount(index uint32) (*HDWalletSigner, error) {
	// Parse the current path and modify the last index
	path := fmt.Sprintf("m/44'/195'/0'/0/%d", index)

	derivationPath, err := hdwallet.ParseDerivationPath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid derivation path: %w", err)
	}

	account, err := s.wallet.Derive(derivationPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account: %w", err)
	}

	privKey, err := s.wallet.PrivateKey(account)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	// Create the underlying PrivateKeySigner
	privateKeySigner, err := newPrivateKeySigner(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key signer: %w", err)
	}

	return &HDWalletSigner{
		PrivateKeySigner: privateKeySigner,
		wallet:           s.wallet,
		derivationPath:   path,
	}, nil
}

// GetMasterKey returns the master public key
func (s *HDWalletSigner) GetMasterKey() (*ecdsa.PublicKey, error) {
	derivationPath, err := hdwallet.ParseDerivationPath("m/44'/195'/0'/0")
	if err != nil {
		return nil, fmt.Errorf("failed to parse master derivation path: %w", err)
	}

	account, err := s.wallet.Derive(derivationPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to derive master account: %w", err)
	}

	pubKey, err := s.wallet.PublicKey(account)
	if err != nil {
		return nil, fmt.Errorf("failed to get master public key: %w", err)
	}

	return pubKey, nil
}

// Wallet returns the underlying HD wallet (use with caution)
func (s *HDWalletSigner) Wallet() *hdwallet.Wallet {
	return s.wallet
}

// All Signer interface methods are inherited from PrivateKeySigner
// Address() *types.Address
// PublicKey() *ecdsa.PublicKey
// Sign(tx *core.Transaction) (*core.Transaction, error)
// SignWithPermissionID(tx *core.Transaction, permissionID int32) (*core.Transaction, error)
// SignMessageV2(message string) (string, error)
