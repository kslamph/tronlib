// Package types defines fundamental types and error values used across the
// TRON SDK, including Address, transaction wrappers, and common constants.
//
// The Address type supports multiple encodings:
//   - Base58Check string (T-prefixed, length 34) via Address.Base58() or String()
//   - TRON bytes (0x41-prefixed 21 bytes) via Address.Bytes()
//   - EVM bytes (20 bytes) via Address.BytesEVM()
//   - Hex forms via Address.Hex() (41-prefixed) and Address.HexEVM() (0x-prefixed)
//
// Constructors accept flexible inputs and validate length and prefixes. Prefer
// NewAddress[...] constructors rather than constructing Address directly.
package types

