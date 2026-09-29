package key

import (
	"crypto/ecdsa"

	"github.com/kslamph/tronlib/v2/tron"
)

// Signer signs hashes with a private key. Address returns the TRON address
// derived from the public key (0x41-prefixed, value type). Sign signs the
// GIVEN hash with no extra hashing or prefixing and returns the raw 65-byte
// go-ethereum [R || S || V] signature, where V is 0 or 1.
type Signer interface {
	Address() tron.Address
	PublicKey() *ecdsa.PublicKey
	Sign(hash []byte) ([]byte, error)
}
