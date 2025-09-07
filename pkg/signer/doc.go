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

// Package signer contains key management and transaction signing utilities,
// including HD wallet derivation and raw private key signing.
//
// # Transaction Signing
//
// To sign transactions, use the package-level `SignTx` function along with a `Signer` implementation:
//
// Private Key Signer Example:
//
//	pk, _ := signer.NewPrivateKeySigner("0x<hex-privkey>")
//	err := signer.SignTx(pk, transaction)
//
// HDWalletSigner Example:
//
//	mnemonic := "tag voyage vapor fence fossil mimic pelican gorilla grocery solar talent"
//	path := "m/44'/195'/0'/0/0"
//	hdSigner, _ := signer.NewHDWalletSigner(mnemonic, "", path) // Passphrase is optional
//	err := signer.SignTx(hdSigner, transaction)
//
// # Message Signing
//
// To sign arbitrary messages using TIP-191 format (v2), use the package-level `SignMessageV2` function:
//
//	privateKey := "0x..."
//	message := "Hello Tron!"
//	signer, _ := signer.NewPrivateKeySigner(privateKey)
//	signature, err := SignMessageV2(signer, message)
//
// # Error Handling
//
// Common error types:
//   - ErrInvalidPrivateKey - Invalid private key format
//   - ErrInvalidMnemonic - Invalid mnemonic phrase
//   - ErrDeriveFailed - Key derivation failed
//
// Always check for errors in production code.
package signer
