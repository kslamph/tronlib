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
