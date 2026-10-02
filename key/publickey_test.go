package key

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/kslamph/tronlib/v2/tron"
)

// TestSignerPublicKeyMatchesItsAddress pins the one invariant that makes
// Signer.PublicKey() trustworthy: the key it hands back must be the key the
// address was derived from.
//
// This is not a restatement of the constructor. Address() and PublicKey() are
// two accessors over the same private key, and nothing in Go connects them —
// returning some other valid public key compiles, runs, and looks fine. The
// suite did not notice: a mutation returning a *different* key survived every
// test in the repo. That matters because PublicKey() is exported on the
// interface: a caller verifying a signature against signer.PublicKey() would
// silently reach the wrong conclusion, and the inconsistency between
// PublicKey() and Address() is invisible at the call site.
func TestSignerPublicKeyMatchesItsAddress(t *testing.T) {
	signers := map[string]func(t *testing.T) Signer{
		"private key": func(t *testing.T) Signer {
			t.Helper()
			s, err := PrivateKeyFromHex(fixtureHexKey)
			if err != nil {
				t.Fatalf("PrivateKeyFromHex: %v", err)
			}
			return s
		},
		"hd wallet": func(t *testing.T) Signer {
			t.Helper()
			s, err := PrivateKeyFromMnemonic(
				"abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
				"", "m/44'/195'/0'/0/0")
			if err != nil {
				t.Fatalf("PrivateKeyFromMnemonic: %v", err)
			}
			return s
		},
	}

	for name, mk := range signers {
		t.Run(name, func(t *testing.T) {
			s := mk(t)
			pub := s.PublicKey()
			if pub == nil {
				t.Fatal("PublicKey() = nil")
			}

			// The address is defined as 0x41 || keccak256(pubkey)[12:]. So
			// re-deriving it from the returned key must reproduce Address()
			// exactly. This is the link between the two accessors.
			want := s.Address()
			got, err := tron.AddressFromBytes(append([]byte{0x41}, crypto.PubkeyToAddress(*pub).Bytes()...))
			if err != nil {
				t.Fatalf("re-deriving the address from PublicKey(): %v", err)
			}
			if got != want {
				t.Fatalf("address from PublicKey() = %s, but Address() = %s", got, want)
			}
		})
	}
}

// TestSignerSignVerifiesAgainstItsPublicKey closes the loop through the
// signature itself, which is the only path a caller actually uses.
//
// Recovering the signer from Sign()'s output with crypto's own ecrecover must
// yield PublicKey(). A signer whose Sign and PublicKey disagree would produce
// signatures nobody can verify under the key the signer advertises.
func TestSignerSignVerifiesAgainstItsPublicKey(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}

	hash := crypto.Keccak256([]byte("the message under test"))
	sig, err := s.Sign(hash)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) != 65 {
		t.Fatalf("signature is %d bytes, want 65", len(sig))
	}

	// go-ethereum recovers [R || S || V] with V in {0,1}.
	recovered, err := crypto.SigToPub(hash, sig)
	if err != nil {
		t.Fatalf("SigToPub: %v", err)
	}
	if !recovered.Equal(s.PublicKey()) {
		t.Fatal("the key recovered from Sign() output is not the signer's advertised PublicKey()")
	}

	// And the recovered address must be the signer's own, so the whole
	// PublicKey -> Sign -> Address triangle holds.
	if got, err := RecoverAddress(hash, sig); err != nil {
		t.Fatalf("RecoverAddress: %v", err)
	} else if got != s.Address() {
		t.Fatalf("RecoverAddress = %s, want %s", got, s.Address())
	}
}

// TestSignerPublicKeyIsNotLoadBearingOnSign documents the one sharp edge in
// PublicKey(): it returns a pointer into the signer's own key struct rather
// than a copy, so callers must treat the result as read-only. The contract
// note on Signer says so out loud; this test makes the consequence checkable.
//
// The alias is observed by pointer identity rather than by writing to the key.
// ecdsa.PublicKey's raw coordinates are deprecated as of Go 1.26 precisely
// because mutating them can produce invalid keys, so a test that tampered with
// them to make the point would be doing the very thing the deprecation warns
// about.
//
// What is worth pinning is that nothing load-bearing depends on the accessor:
// Address is derived from the private key at construction, and Sign works from
// the private scalar. A caller writing through the returned pointer can only
// corrupt what the accessor hands out afterwards — never the reported address,
// and never the signatures.
func TestSignerPublicKeyIsNotLoadBearingOnSign(t *testing.T) {
	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}

	first, second := s.PublicKey(), s.PublicKey()
	if first == nil || second == nil {
		t.Fatal("PublicKey() = nil")
	}
	if first != second {
		t.Log("PublicKey() now returns a fresh copy each call; the alias is gone (behaviour changed)")
	}

	// Address is fixed at construction and must not depend on the accessor.
	want := s.Address()
	if s.Address() != want {
		t.Fatalf("Address() is not stable: %s then %s", want, s.Address())
	}

	// Signing works from the private scalar, so it stays consistent with the
	// address regardless of what the accessor has handed out.
	hash := crypto.Keccak256([]byte("the message under test"))
	sig, err := s.Sign(hash)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	recovered, err := crypto.SigToPub(hash, sig)
	if err != nil {
		t.Fatalf("SigToPub: %v", err)
	}
	if !recovered.Equal(first) {
		t.Fatal("the signature does not verify under the key PublicKey() reported")
	}
	if got, err := RecoverAddress(hash, sig); err != nil {
		t.Fatalf("RecoverAddress: %v", err)
	} else if got != want {
		t.Fatalf("RecoverAddress = %s, want the signer's Address() %s", got, want)
	}
}

// TestSignerPublicKeyIsStableAndComplete is a compile-time-ish reminder that
// every Signer implementation must carry all three accessors, and that the
// accessor answers the same key every time. A new implementation that stubs
// PublicKey, or returns a fresh key per call, fails here.
func TestSignerPublicKeyIsStableAndComplete(t *testing.T) {
	var _ Signer = (*privateKeySigner)(nil)
	var _ Signer = (*hdWalletSigner)(nil)

	s, err := PrivateKeyFromHex(fixtureHexKey)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	a, b := s.PublicKey(), s.PublicKey()
	var _ *ecdsa.PublicKey = a
	if !a.Equal(b) {
		t.Fatal("PublicKey() is not stable across calls")
	}
	if a.Curve != b.Curve {
		t.Fatal("PublicKey() returned a different curve on the second call")
	}
	// TRON is secp256k1, so the advertised key must live on a 256-bit curve.
	// (Checked via BitSize rather than Params().Name: go-ethereum's S256 is a
	// native bitCurve whose Params().Name is empty, so asserting on the name
	// would be testing go-ethereum's internals, not this package.)
	if got := a.Curve.Params().BitSize; got != 256 {
		t.Fatalf("PublicKey() is on a %d-bit curve, want secp256k1 (256)", got)
	}
}
