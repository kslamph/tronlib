package types

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"google.golang.org/protobuf/proto"

	"github.com/kslamph/tronlib/pb/core"
)

// Add these constants at the top of the file
const (
	SignedDataPrefix  = byte(0x19) // Prefix to prevent collision with Transaction/Block
	SignedDataVersion = byte(0x00) // Version 0 for data with intended validator

	// TronWeb message prefix (v1)
	TronMessagePrefix = "\x19TRON Signed Message:\n"
)

// Account represents a Tron account with its private key and implements the Signer interface
type Account struct {
	address *Address
	privKey *ecdsa.PrivateKey
	pubKey  *ecdsa.PublicKey
	// hexAddress string
}

// NewAccountFromPrivateKey creates an Account from a hex private key
func NewAccountFromPrivateKey(hexPrivKey string) (*Account, error) {
	// Decode and validate private key
	key, err := hex.DecodeString(hexPrivKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex private key: %w", err)
	}

	privKey, err := crypto.ToECDSA(key)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return newAccount(privKey)
}

// NewAccountFromHDWallet creates an Account from HD wallet path
func NewAccountFromHDWallet(mnemonic string, passphrase string, path string) (*Account, error) {
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

	return newAccount(privKey)
}

// newAccount creates a new Account from a private key
func newAccount(privKey *ecdsa.PrivateKey) (*Account, error) {
	pubKey := privKey.PublicKey
	ethAddr := crypto.PubkeyToAddress(pubKey)

	// Add TRON prefix (0x41)
	tronBytes := append([]byte{0x41}, ethAddr.Bytes()...)

	// Convert to hex address
	// hexAddr := hex.EncodeToString(tronBytes)

	// Convert to base58 address
	tronAddr, err := NewAddressFromBytes(tronBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create tron address: %w", err)
	}

	return &Account{
		address: tronAddr,
		privKey: privKey,
		pubKey:  &pubKey,
		// hexAddress: hexAddr,
	}, nil
}

// Address returns the account's address
func (a *Account) Address() *Address {
	return a.address
}

// PrivateKeyHex returns the account's private key in hex format
func (a *Account) PrivateKeyHex() string {
	privateKeyBytes := crypto.FromECDSA(a.privKey)
	return hex.EncodeToString(privateKeyBytes)
}

// PublicKey returns the account's public key
func (a *Account) PublicKey() *ecdsa.PublicKey {
	return a.pubKey
}

// Sign signs the given transaction with the account's private key
// This is a wrapper for the MultiSign function with permission ID 2 (active permission)
func (a *Account) Sign(tx *core.Transaction) (*core.Transaction, error) {
	return a.MultiSign(tx, 2)
}

// Sign signs the given transaction with the account's private key
func (a *Account) MultiSign(tx *core.Transaction, permissionID int32) (*core.Transaction, error) {
	// Set permission ID for active permission
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

	// Sign the hash
	signature, err := crypto.Sign(hash, a.privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Clear any existing signatures and add the new one
	tx.Signature = nil
	tx.Signature = append(tx.Signature, signature)

	return tx, nil
}

// SignMessageV2 signs a message using TIP-191 format (v2)
func (a *Account) SignMessageV2(message string) (string, error) {
	var data []byte
	if strings.HasPrefix(message, "0x") {
		// Assume hex-encoded string.
		data = common.FromHex(message)
	} else {
		data = []byte(message)
	}

	//Prefix the message.
	messageLen := len(data)
	prefixedMessage := []byte(fmt.Sprintf("%s%d%s", TronMessagePrefix, messageLen, string(data)))

	//Hash the prefixed message (Keccak256).
	hash := crypto.Keccak256Hash(prefixedMessage)

	//Sign the hash.
	signature, err := crypto.Sign(hash.Bytes(), a.privKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	// Adjust the recovery ID (v).  go-ethereum's Sign function returns
	// the signature in [R || S || V] format, where V is 0 or 1.  Tron
	// expects V to be 27 or 28, so we add 27.
	signature[64] += 27

	// 7. Return the hex-encoded signature.
	return "0x" + common.Bytes2Hex(signature), nil

}
