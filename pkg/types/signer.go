package types

import (
	"crypto/ecdsa"
)

// Signer defines the interface for signing Tron transactions and messages
type Signer interface {
	// Address returns the account's address
	Address() *Address

	// PublicKey returns the account's public key
	PublicKey() *ecdsa.PublicKey

	// Sign signs a transaction, supporting both core.Transaction and api.TransactionExtention types
	// It modifies the transaction in place by appending the signature
	Sign(tx any) error

	// SignMessageV2 signs a message using TIP-191 format (v2)
	SignMessageV2(message string) (string, error)
}
