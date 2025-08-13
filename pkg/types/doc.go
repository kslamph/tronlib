// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package types defines fundamental types and error values used across the
// TRON SDK, including Address, transaction wrappers, and common constants.
//
// # Address Type
//
// The Address type supports multiple encodings:
//   - Base58Check string (T-prefixed, length 34) via Address.Base58() or String()
//   - TRON bytes (0x41-prefixed 21 bytes) via Address.Bytes()
//   - EVM bytes (20 bytes) via Address.BytesEVM()
//   - Hex forms via Address.Hex() (41-prefixed) and Address.HexEVM() (0x-prefixed)
//
// Constructors accept flexible inputs and validate length and prefixes. Prefer
// NewAddress[...] constructors rather than constructing Address directly.
//
//	addr, _ := types.NewAddress("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1")
//	_ = addr.Hex()
//
// # Error Types
//
// The package defines sentinel errors used throughout the SDK:
//   - ErrInvalidAddress - Invalid TRON address format
//   - ErrInvalidTransaction - Invalid transaction structure
//   - ErrInvalidSignature - Invalid transaction signature
//
// # Constants
//
// Common constants used in TRON operations:
//   - SUN_PER_TRX - Number of SUN in 1 TRX (1,000,000)
//   - ADDRESS_SIZE - Size of TRON address in bytes (21)
//
// Always check for errors in production code.
package types
