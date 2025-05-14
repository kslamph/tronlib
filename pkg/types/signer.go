package types

import (
	"crypto/ecdsa"

	"github.com/kslamph/tronlib/pb/core"
)

// Signer defines the interface for signing Tron transactions and messages
type Signer interface {
	// Address returns the account's address
	Address() *Address

	// PublicKey returns the account's public key
	PublicKey() *ecdsa.PublicKey

	// Sign signs a transaction with permissionID 2 (active permission)
	Sign(tx *core.Transaction) (*core.Transaction, error)

	// MultiSign signs a transaction with the specified permissionID
	MultiSign(tx *core.Transaction, permissionID int32) (*core.Transaction, error)

	// SignMessageV2 signs a message using TIP-191 format (v2)
	SignMessageV2(message string) (string, error)
}
