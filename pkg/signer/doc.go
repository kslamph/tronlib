// Package signer contains key management and transaction signing utilities,
// including HD wallet derivation and raw private key signing.
//
// # Private Key Signing
//
// Sign transactions with a private key:
//
//	pk, _ := signer.NewPrivateKeySigner("0x<hex-privkey>")
//	signature, _ := pk.Sign(transaction)
//
// # HD Wallet Support
//
// Derive keys from an HD wallet:
//
//	mnemonic := "your twelve word mnemonic phrase"
//	wallet, _ := signer.NewHDWallet(mnemonic)
//	account, _ := wallet.DerivePath("m/44'/195'/0'/0/0")
//	pk, _ := account.PrivateKey()
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
