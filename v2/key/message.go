package key

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/kslamph/tronlib/v2/tron"
)

// tronMessagePrefix is the TIP-191 message prefix: length-annotated,
// EIP-191-style, with TRON in place of Ethereum.
const tronMessagePrefix = "\x19TRON Signed Message:\n"

// tip191Prefixed returns the TIP-191 v2 wire bytes for message:
// prefix || len(data) || data, where data is the raw message bytes, or the
// hex-decoded bytes when message carries a "0x" prefix.
func tip191Prefixed(message string) ([]byte, error) {
	var data []byte
	if strings.HasPrefix(message, "0x") {
		// Assume hex-encoded string.
		data = common.FromHex(message)
	} else {
		data = []byte(message)
	}

	messageLen := len(data)
	return []byte(fmt.Sprintf("%s%d%s", tronMessagePrefix, messageLen, string(data))), nil
}

// SignMessageV2 signs message in TIP-191 v2 format and returns the 0x-prefixed
// 65-byte hex signature (TronWeb signMessageV2 compatible). A message starting
// with "0x" is treated as hex-encoded bytes, matching v1 behavior; anything
// else is signed as UTF-8 text.
func SignMessageV2(s Signer, message string) (string, error) {
	const op = "SignMessageV2"
	prefixedMessage, err := tip191Prefixed(message)
	if err != nil {
		return "", &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "a 0x-prefixed message must be valid hex", Cause: err}
	}

	// Hash the prefixed message (Keccak256).
	hash := crypto.Keccak256Hash(prefixedMessage)

	// Sign the hash.
	signature, err := s.Sign(hash.Bytes())
	if err != nil {
		return "", &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "check the signer's private key", Cause: err}
	}

	// Adjust the recovery ID (v). go-ethereum's Sign function returns
	// the signature in [R || S || V] format, where V is 0 or 1. Tron
	// expects V to be 27 or 28, so we add 27.
	signature[64] += 27

	return "0x" + common.Bytes2Hex(signature), nil
}

// VerifyMessageV2 reports whether signature — the 0x-prefixed 65-byte hex
// produced by SignMessageV2 (TronWeb verifyMessageV2 compatible; a missing 0x
// prefix is tolerated) — recovers to addr over the TIP-191-prefixed message.
// The message is interpreted exactly as in SignMessageV2.
func VerifyMessageV2(message, signature string, addr tron.Address) (bool, error) {
	const op = "VerifyMessageV2"

	sigBytes, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "pass the 0x-hex signature returned by SignMessageV2", Cause: err}
	}
	if len(sigBytes) != 65 {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: fmt.Sprintf("signature must be 65 bytes, got %d bytes", len(sigBytes))}
	}

	// Adjust recovery ID (v) back to go-ethereum format.
	// Tron uses 27/28, go-ethereum uses 0/1.
	if sigBytes[64] < 27 {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: fmt.Sprintf("invalid recovery ID, must be 27 or 28, got %d", sigBytes[64])}
	}
	sigBytes[64] -= 27

	prefixedMessage, err := tip191Prefixed(message)
	if err != nil {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "a 0x-prefixed message must be valid hex", Cause: err}
	}

	// Hash the prefixed message (same as signing).
	hash := crypto.Keccak256Hash(prefixedMessage)

	// Recover the public key.
	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "the signature bytes are corrupt", Cause: err}
	}

	// Convert public key to TRON address.
	ethAddr := crypto.PubkeyToAddress(*pubKey)
	tronBytes := append([]byte{0x41}, ethAddr.Bytes()...)

	recoveredAddr, err := tron.AddressFromBytes(tronBytes)
	if err != nil {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "the signature recovered an invalid address", Cause: err}
	}

	return recoveredAddr == addr, nil
}

// recoverAddress recovers the signer's address from a hash and raw
// 65-byte [R || S || V] signature (V in {0, 1}). Test helper for proving
// Sign output is a standard recoverable secp256k1 signature.
func recoverAddress(hash, sig []byte) (tron.Address, error) {
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return tron.Address{}, err
	}
	ethAddr := crypto.PubkeyToAddress(*pubKey)
	return tron.AddressFromBytes(append([]byte{0x41}, ethAddr.Bytes()...))
}

// keccak256 hashes b with Keccak-256. Test helper.
func keccak256(b []byte) []byte {
	h := crypto.Keccak256Hash(b)
	return h.Bytes()
}
