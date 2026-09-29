package key

import (
	"crypto/ecdsa"
	"encoding/hex"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/kslamph/tronlib/v2/tron"
)

// zero overwrites b with zeros. Go cannot guarantee a secret is unreachable
// (the GC may have copied it, and big.Int values cannot be scrubbed), but
// this removes the explicit secret buffer this package holds;
// runtime.KeepAlive keeps the write from being elided as dead.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}

// privateKeySigner implements Signer with a fixed private key.
type privateKeySigner struct {
	address tron.Address
	privKey *ecdsa.PrivateKey
}

// PrivateKeyFromHex builds a Signer from a hex private key, with or without
// a "0x" prefix. Bad hex, a key that is not exactly 32 bytes, or a key out of
// the secp256k1 range returns an error with code key.invalid.
func PrivateKeyFromHex(hexKey string) (Signer, error) {
	const op = "PrivateKeyFromHex"
	hexKey = strings.TrimPrefix(hexKey, "0x")

	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "pass 64 hex characters (a 32-byte private key), optionally prefixed with 0x", Cause: err}
	}
	defer zero(key) // the parsed key is copied into an ecdsa.PrivateKey; drop the raw bytes

	privKey, err := crypto.ToECDSA(key)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "pass 64 hex characters (a 32-byte private key), optionally prefixed with 0x", Cause: err}
	}

	return newPrivateKeySigner(privKey)
}

// newPrivateKeySigner derives the TRON address from the key: the
// keccak-256-based EVM address, prefixed with 0x41.
func newPrivateKeySigner(privKey *ecdsa.PrivateKey) (Signer, error) {
	ethAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	// Add TRON prefix (0x41).
	tronBytes := append([]byte{0x41}, ethAddr.Bytes()...)

	tronAddr, err := tron.AddressFromBytes(tronBytes)
	if err != nil {
		return nil, err
	}

	return &privateKeySigner{
		address: tronAddr,
		privKey: privKey,
	}, nil
}

// Address returns the account's address.
func (s *privateKeySigner) Address() tron.Address {
	return s.address
}

// PublicKey returns the account's public key.
func (s *privateKeySigner) PublicKey() *ecdsa.PublicKey {
	return &s.privKey.PublicKey
}

// Sign signs a given hash with the private key and returns the raw signature
// bytes: 65-byte [R || S || V] with V in {0, 1}, per the go-ethereum convention.
func (s *privateKeySigner) Sign(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, s.privKey)
}
