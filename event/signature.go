package event

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"
)

// CanonicalSignature builds the canonical event signature whose keccak256 is
// the registry key: the event name, "(", the comma-joined declared types, ")".
// Indexed flags and parameter names are deliberately absent — they are not
// part of the hashed signature, which is why two contracts can share a key
// while disagreeing about the layout (see scope.register).
//
// This is the exported single source of the rule. Definition.signature
// delegates to it, and tooling that derives hashes outside the registry
// (cmd/eventtool) calls it directly, so a derived hash and a registered hash
// cannot drift apart.
func CanonicalSignature(name string, inputTypes []string) string {
	return fmt.Sprintf("%s(%s)", name, strings.Join(inputTypes, ","))
}

// SignatureKey is the registry key for an event: the full 32 bytes of
// keccak256(CanonicalSignature(name, inputTypes)) — a log's entire first
// topic, not a prefix of it. Keying on a 4-byte prefix let two unrelated
// signatures share one slot, and one of them silently decoded against the
// other's definition.
//
// The hash is legacy keccak-256 (golang.org/x/crypto/sha3), not SHA3-256.
func SignatureKey(name string, inputTypes []string) [32]byte {
	return hashSignature(CanonicalSignature(name, inputTypes))
}

// hashSignature is the one keccak call in the package: every registry key and
// every exported SignatureKey flows through it.
func hashSignature(signature string) [32]byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(signature))
	var key [32]byte
	copy(key[:], hasher.Sum(nil))
	return key
}
