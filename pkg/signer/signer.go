package signer

import (
	"crypto/ecdsa"

	"github.com/kslamph/tronlib/pkg/types"
)

// Signer defines the interface for signing data (e.g., transaction hashes, message hashes).
type Signer interface {
	// Address returns the account's address
	Address() *types.Address

	// PublicKey returns the account's public key
	PublicKey() *ecdsa.PublicKey

	// Sign signs a given hash and returns the raw signature bytes.
	// Implementations should ensure this function only signs the provided hash,
	// without any additional hashing or prefixing.
	Sign(hash []byte) ([]byte, error)
}
