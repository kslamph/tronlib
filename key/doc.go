// Package key provides message signing for TRON: Signer implementations
// backed by a hex private key or an HD wallet (BIP-39 mnemonic + derivation
// path), plus TIP-191 v2 message sign/verify. A Signer signs the exact hash
// it is given — hashing inputs into 32-byte digests is the caller's choice.
// The package does no I/O of any kind.
package key
