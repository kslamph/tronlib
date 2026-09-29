package key

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/kslamph/tronlib/v2/tron"
)

// tronMessagePrefix is the TIP-191 message prefix: length-annotated,
// EIP-191-style, with TRON in place of Ethereum.
const tronMessagePrefix = "\x19TRON Signed Message:\n"

// tip191Prefixed returns the TIP-191 v2 wire bytes for message:
// prefix || len(data) || data, where data is the raw message bytes, or the
// hex-decoded bytes when message carries a "0x"/"0X" prefix.
func tip191Prefixed(message string) ([]byte, error) {
	data, err := messagePayload(message)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("%s%d%s", tronMessagePrefix, len(data), string(data))), nil
}

// messagePayload interprets message as the TIP-191 payload bytes. A "0x"/"0X"
// prefix selects hex decoding, and that hex MUST be well-formed (even number
// of hex digits); anything else is signed as UTF-8 text.
//
// This is deliberately stricter than go-ethereum's common.FromHex, which
// discards the decode error and returns a silently truncated prefix — a
// wallet could then display one message and sign a different byte string.
func messagePayload(message string) ([]byte, error) {
	if hasHexPrefix(message) {
		body := trimHexPrefix(message)
		if len(body)%2 != 0 {
			return nil, fmt.Errorf("hex message has an odd number of digits (%d)", len(body))
		}
		data, err := hex.DecodeString(body)
		if err != nil {
			return nil, fmt.Errorf("hex message is not valid hex: %w", err)
		}
		return data, nil
	}
	return []byte(message), nil
}

// hasHexPrefix reports whether s carries a 0x or 0X prefix.
func hasHexPrefix(s string) bool {
	return len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X')
}

// trimHexPrefix strips a leading 0x/0X.
func trimHexPrefix(s string) string {
	if hasHexPrefix(s) {
		return s[2:]
	}
	return s
}

// SignMessageV2 signs message in TIP-191 v2 format and returns the 0x-prefixed
// 65-byte hex signature (TronWeb signMessageV2 compatible). A message starting
// with "0x" or "0X" is treated as hex-encoded bytes and MUST be well-formed
// hex (an odd digit count or a non-hex character is key.invalid); anything
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
// prefix or an uppercase 0X is tolerated) — recovers to addr over the
// TIP-191-prefixed message. Only CANONICAL (low-S) signatures are accepted;
// a malleated high-S variant is key.invalid. The message is interpreted
// exactly as in SignMessageV2.
func VerifyMessageV2(message, signature string, addr tron.Address) (bool, error) {
	const op = "VerifyMessageV2"

	sigBytes, err := hex.DecodeString(trimHexPrefix(signature))
	if err != nil {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "pass the 0x-hex signature returned by SignMessageV2", Cause: err}
	}
	if len(sigBytes) != 65 {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: fmt.Sprintf("signature must be 65 bytes, got %d bytes", len(sigBytes))}
	}

	// Adjust the recovery ID back to go-ethereum format, then reject
	// non-canonical signatures. ValidateSignatureValues(homestead=true)
	// requires v in {0,1}, 1 <= r,s < N, and LOW S — the canonical form
	// SignMessageV2 emits; it rejects the malleated duplicate (N-S with the
	// recovery id flipped).
	v := sigBytes[64]
	if v < 27 || v > 28 {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: fmt.Sprintf("invalid recovery ID, must be 27 or 28, got %d", v)}
	}
	v -= 27
	rVal := new(big.Int).SetBytes(sigBytes[:32])
	sVal := new(big.Int).SetBytes(sigBytes[32:64])
	if !crypto.ValidateSignatureValues(v, rVal, sVal, true) {
		return false, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "signature is non-canonical (high-S, or r/s/v out of range)"}
	}
	sigBytes[64] = v

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

// RecoverAddress recovers the signer's TRON address from a 32-byte hash and a
// raw 65-byte [R || S || V] signature. V may be go-ethereum form (0/1) or the
// TIP-191/Ethereum display form (27/28). Malformed input is a classified
// key.invalid error. Callers use it to check that a signature actually
// belongs to the signer it is attributed to.
func RecoverAddress(hash, sig []byte) (tron.Address, error) {
	const op = "key.RecoverAddress"
	if len(hash) != 32 {
		return tron.Address{}, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: fmt.Sprintf("hash must be 32 bytes, got %d", len(hash))}
	}
	if len(sig) != 65 {
		return tron.Address{}, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: fmt.Sprintf("signature must be 65 bytes, got %d", len(sig))}
	}
	s := append([]byte{}, sig...)
	if s[64] >= 27 {
		s[64] -= 27
	}
	addr, err := recoverAddress(hash, s)
	if err != nil {
		return tron.Address{}, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Hint: "the signature does not recover to a valid address", Cause: err}
	}
	return addr, nil
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
