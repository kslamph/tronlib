package key

import (
	"encoding/hex"
	"strings"
	"testing"
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
