package key

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/kslamph/tronlib/v2/tron"
)

func TestSignMessageV2RoundTrip(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	sig, err := SignMessageV2(s, "Hello Tron!")
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	if sig == "" {
		t.Fatal("empty signature")
	}
	if !strings.HasPrefix(sig, "0x") {
		t.Fatalf("signature %q not 0x-prefixed hex", sig)
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(sig, "0x")); err != nil {
		t.Fatalf("signature not hex: %v", err)
	}
	ok, err := VerifyMessageV2("Hello Tron!", sig, s.Address())
	if err != nil {
		t.Fatalf("VerifyMessageV2: %v", err)
	}
	if !ok {
		t.Fatal("VerifyMessageV2 = false, want true for own signature")
	}
}

// TestVerifyMessageV2Missing0xPrefix pins TronWeb verifyMessageV2 tolerance:
// a signature without the 0x prefix must still verify.
func TestVerifyMessageV2Missing0xPrefix(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	sig, err := SignMessageV2(s, "Hello Tron!")
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	stripped := strings.TrimPrefix(sig, "0x")
	ok, err := VerifyMessageV2("Hello Tron!", stripped, s.Address())
	if err != nil {
		t.Fatalf("VerifyMessageV2 (no 0x): %v", err)
	}
	if !ok {
		t.Fatal("signature without 0x prefix failed to verify")
	}
}

func TestSignMessageV2HexInput(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	// v1 SignMessageV2 treats a "0x" prefix as hex-encoded bytes, not text.
	sig, err := SignMessageV2(s, "0x48656c6c6f")
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	ok, err := VerifyMessageV2("0x48656c6c6f", sig, s.Address())
	if err != nil {
		t.Fatalf("VerifyMessageV2: %v", err)
	}
	if !ok {
		t.Fatal("hex-encoded message failed to verify")
	}
}

func TestVerifyMessageV2TamperedMessage(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	sig, err := SignMessageV2(s, "Hello Tron!")
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	ok, err := VerifyMessageV2("Hello Tron?", sig, s.Address())
	if err != nil {
		t.Fatalf("VerifyMessageV2: %v", err)
	}
	if ok {
		t.Fatal("tampered message verified")
	}
}

func TestVerifyMessageV2WrongAddress(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	other, err := PrivateKeyFromHex("1111111111111111111111111111111111111111111111111111111111111111")
	if err != nil {
		t.Fatalf("second key: %v", err)
	}
	sig, err := SignMessageV2(s, "Hello Tron!")
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	ok, err := VerifyMessageV2("Hello Tron!", sig, other.Address())
	if err != nil {
		t.Fatalf("VerifyMessageV2: %v", err)
	}
	if ok {
		t.Fatal("signature verified against wrong address")
	}
}

func TestVerifyMessageV2BadSignature(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	// Not hex.
	if _, err := VerifyMessageV2("Hello Tron!", "not hex!!", s.Address()); err == nil {
		t.Fatal("expected error for non-hex signature")
	}
	// Valid hex but wrong length.
	short := "0x" + hex.EncodeToString([]byte("too short"))
	if _, err := VerifyMessageV2("Hello Tron!", short, s.Address()); err == nil {
		t.Fatal("expected error for non-65-byte signature")
	}
	// Valid 65-byte hex but bad recovery id.
	badV := "0x" + hex.EncodeToString(make([]byte, 65))
	if _, err := VerifyMessageV2("Hello Tron!", badV, s.Address()); err == nil {
		t.Fatal("expected error for invalid recovery id")
	}
}

// TestSignMessageV2RejectsMalformedHexMessage: a 0x-prefixed message is a
// hex-encoded byte string; malformed input (non-hex or odd length) must be a
// classified error, NOT silently truncated the way go-ethereum's
// common.FromHex does.
func TestSignMessageV2RejectsMalformedHexMessage(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	for _, msg := range []string{"0x12zz", "0x123", "0xz", "0xzz"} {
		if _, err := SignMessageV2(s, msg); !tron.HasCode(err, tron.CodeKeyInvalid) {
			t.Errorf("SignMessageV2(%q): err = %v, want key.invalid", msg, err)
		}
	}
}

// TestVerifyMessageV2RejectsMalformedHexMessage: the same validation applies
// on the verify path.
func TestVerifyMessageV2RejectsMalformedHexMessage(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	sig, err := SignMessageV2(s, "0x48656c6c6f")
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	if _, err := VerifyMessageV2("0x12zz", sig, s.Address()); !tron.HasCode(err, tron.CodeKeyInvalid) {
		t.Errorf("VerifyMessageV2(malformed hex): err = %v, want key.invalid", err)
	}
}

// TestMessageHexPrefixCaseInsensitive: 0X and 0x both select hex decoding and
// must sign the same bytes.
func TestMessageHexPrefixCaseInsensitive(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	lower, err := SignMessageV2(s, "0x48656c6c6f")
	if err != nil {
		t.Fatalf("SignMessageV2(0x): %v", err)
	}
	upper, err := SignMessageV2(s, "0X48656c6c6f")
	if err != nil {
		t.Fatalf("SignMessageV2(0X): %v", err)
	}
	if lower != upper {
		t.Errorf("0x and 0X messages signed different bytes: %s vs %s", lower, upper)
	}
}

// TestVerifyMessageV2RejectsHighS: an ECDSA signature is malleable — flipping
// S to N-S with the recovery id flipped is the same signature. It must be
// rejected as non-canonical (low-S is the canonical form, the one
// SignMessageV2 emits).
func TestVerifyMessageV2RejectsHighS(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	sigHex, err := SignMessageV2(s, "Hello Tron!")
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	sv := new(big.Int).SetBytes(raw[32:64])
	highS := new(big.Int).Sub(crypto.S256().Params().N, sv)
	highS.FillBytes(raw[32:64])
	switch raw[64] {
	case 27:
		raw[64] = 28
	case 28:
		raw[64] = 27
	}
	high := "0x" + hex.EncodeToString(raw)
	if _, err := VerifyMessageV2("Hello Tron!", high, s.Address()); !tron.HasCode(err, tron.CodeKeyInvalid) {
		t.Errorf("high-S signature: err = %v, want key.invalid", err)
	}
}

// TestVerifyMessageV2Uppercase0XSignature: 0X is accepted like 0x.
func TestVerifyMessageV2Uppercase0XSignature(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	sig, err := SignMessageV2(s, "Hello Tron!")
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	ok, err := VerifyMessageV2("Hello Tron!", "0X"+strings.TrimPrefix(sig, "0x"), s.Address())
	if err != nil || !ok {
		t.Errorf("VerifyMessageV2(0X sig) = (%v, %v), want (true, nil)", ok, err)
	}
}

// TestRecoverAddress: the exported recovery helper returns the signer's
// address for a valid signature, accepts the TIP-191 27/28 recovery id, and
// rejects malformed input.
func TestRecoverAddress(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	hash := keccak256([]byte("recover me"))
	sig, err := s.Sign(hash)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	got, err := RecoverAddress(hash, sig)
	if err != nil || got != s.Address() {
		t.Errorf("RecoverAddress = (%s, %v), want %s", got, err, s.Address())
	}

	// The Ethereum-display V form (27/28) also recovers.
	sig27 := append([]byte{}, sig...)
	sig27[64] += 27
	got, err = RecoverAddress(hash, sig27)
	if err != nil || got != s.Address() {
		t.Errorf("RecoverAddress(V+27) = (%s, %v), want %s", got, err, s.Address())
	}

	if _, err := RecoverAddress(hash, sig[:64]); !tron.HasCode(err, tron.CodeKeyInvalid) {
		t.Errorf("RecoverAddress(short sig): err = %v, want key.invalid", err)
	}
	if _, err := RecoverAddress(hash[:16], sig); !tron.HasCode(err, tron.CodeKeyInvalid) {
		t.Errorf("RecoverAddress(short hash): err = %v, want key.invalid", err)
	}
}

// TestSignMessageV2TronscanInteropVector pins byte-for-byte compatibility with
// the TronScan "verify-sign" tool (TIP-191 v2 / TronWeb signMessageV2). The
// signature below is the one that tool produced on 2026-09-29 for this exact
// message and address, and it is reproducible exactly because both sides use
// RFC 6979 deterministic ECDSA over the same digest:
//
//	keccak256("\x19TRON Signed Message:\n" + len(msg) + msg)
//
// Any change to the prefix, the length encoding, the hashing, or the
// signature layout breaks this test — which is the point: it is the interop
// contract with TronWeb/TronScan.
//
// The key is the repo's public throwaway Nile test key
// (integration_test/test.env NILE_TEST_KEY1); it must never be funded with
// anything of value nor used on mainnet.
func TestSignMessageV2TronscanInteropVector(t *testing.T) {
	const (
		nileTestKey = "69004ce41c53bcddab3f74d5d358d0b5099e0d536e72c9b551b1420080296f21"
		wantAddr    = "TLibQrqpdqPyg11VBJR97Q4H2714xa9GT1"
		message     = "this is something i signed on https://tronscan.org/tools/verify-sign"
		wantSig     = "0xebdc82bdcc80a151d1d509e8e11048ce689e0096d80cad6e6126c4ac4af1e7be003d8b3157b378b4db9dd2f3bc60cb601d0c55cc1aca5679c57095d1e2a6a7781c"
	)
	s, err := PrivateKeyFromHex(nileTestKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	if got := s.Address().String(); got != wantAddr {
		t.Fatalf("address = %s, want %s", got, wantAddr)
	}

	got, err := SignMessageV2(s, message)
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	if got != wantSig {
		t.Errorf("signature drift vs TronScan:\n got  %s\n want %s", got, wantSig)
	}

	ok, err := VerifyMessageV2(message, wantSig, s.Address())
	if err != nil || !ok {
		t.Errorf("VerifyMessageV2(TronScan sig) = (%v, %v), want (true, nil)", ok, err)
	}

	// The signature must NOT verify under a perturbed message: proves the
	// digest binds the exact bytes rather than a prefix.
	if ok, _ := VerifyMessageV2(message+"\n", wantSig, s.Address()); ok {
		t.Error("TronScan signature verified against a newline-perturbed message")
	}
}

func TestSignProduces65ByteRecoverableSignature(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	hash := keccak256([]byte("arbitrary hash chosen by the caller"))
	sig, err := s.Sign(hash)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) != 65 {
		t.Fatalf("signature length = %d, want 65", len(sig))
	}
	// Sign must sign the GIVEN hash with no extra hashing or prefixing, so
	// recovering over the same raw hash yields the signer's own address.
	rec, err := recoverAddress(hash, sig)
	if err != nil {
		t.Fatalf("recoverAddress: %v", err)
	}
	if rec != s.Address() {
		t.Fatalf("recovered %s, want %s", rec, s.Address())
	}
}

func TestSignMessageV2UsesTIP191Prefix(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	msg := "manual prefix check"
	sigHex, err := SignMessageV2(s, msg)
	if err != nil {
		t.Fatalf("SignMessageV2: %v", err)
	}
	sig, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	sig[64] -= 27 // Tron v (27/28) back to go-ethereum (0/1)
	prefixed, err := tip191Prefixed(msg)
	if err != nil {
		t.Fatalf("tip191Prefixed: %v", err)
	}
	rec, err := recoverAddress(keccak256(prefixed), sig)
	if err != nil {
		t.Fatalf("recoverAddress: %v", err)
	}
	if rec != s.Address() {
		t.Fatalf("TIP-191 recovery mismatch: recovered %s, want %s", rec, s.Address())
	}
}
