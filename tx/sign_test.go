package tx

import (
	"crypto/ecdsa"
	"testing"

	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/tron"
)

func mustSigner(t *testing.T, hex string) key.Signer {
	t.Helper()
	s, err := key.PrivateKeyFromHex(hex)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	return s
}

// inconsistentSigner signs with its inner key but reports a different
// Address() — the failure mode Option A must catch (a custom Signer is
// allowed by the public interface, so a mismatch must not be silently
// recorded as the signer).
type inconsistentSigner struct {
	inner   key.Signer
	claimed tron.Address
}

func (s inconsistentSigner) Address() tron.Address         { return s.claimed }
func (s inconsistentSigner) PublicKey() *ecdsa.PublicKey   { return s.inner.PublicKey() }
func (s inconsistentSigner) Sign(h []byte) ([]byte, error) { return s.inner.Sign(h) }

func TestSignRejectsInconsistentSigner(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	native, err := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	lying := inconsistentSigner{
		inner:   mustSigner(t, testKeyHex),
		claimed: mustSigner(t, testKeyHex2).Address(), // a different, real address
	}
	if _, err := native.Sign(lying); !tron.HasCode(err, tron.CodeKeyInvalid) {
		t.Errorf("Sign(inconsistent signer): err = %v, want key.invalid", err)
	}
}

func TestSignReturnsCopyAndAccumulates(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	native, err := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	s1 := mustSigner(t, testKeyHex)
	s2 := mustSigner(t, testKeyHex2)

	signed1, err := native.Sign(s1)
	if err != nil {
		t.Fatalf("Sign(s1): %v", err)
	}
	// receiver untouched
	if native.IsSigned() {
		t.Error("receiver IsSigned after Sign")
	}
	if n := len(native.Transaction().GetSignature()); n != 0 {
		t.Errorf("receiver carries %d signatures, want 0", n)
	}
	if !signed1.IsSigned() || len(signed1.Transaction().GetSignature()) != 1 {
		t.Errorf("signed copy carries %d signatures", len(signed1.Transaction().GetSignature()))
	}
	// signature is the real secp256k1 R||S||V encoding
	if sig := signed1.Transaction().GetSignature()[0]; len(sig) != 65 {
		t.Errorf("signature length = %d, want 65", len(sig))
	}
	// the same copy of raw_data, only the signature list extended
	if signed1.ID() != native.ID() {
		t.Error("Sign changed raw_data; signature must cover the same raw bytes")
	}
	// signers recorded in order
	signers, err := signed1.Signers()
	if err != nil || len(signers) != 1 || signers[0] != s1.Address() {
		t.Errorf("Signers = %v, %v; want [%v]", signers, err, s1.Address())
	}

	// accumulation: Sign(a).Sign(b) carries both signatures, signers in order
	signed2, err := signed1.Sign(s2)
	if err != nil {
		t.Fatalf("Sign(s2): %v", err)
	}
	if n := len(signed2.Transaction().GetSignature()); n != 2 {
		t.Fatalf("accumulated signatures = %d, want 2", n)
	}
	if signed1.IsSigned() && len(signed1.Transaction().GetSignature()) != 1 {
		t.Error("intermediate copy mutated by second Sign")
	}
	signers, _ = signed2.Signers()
	if len(signers) != 2 || signers[0] != s1.Address() || signers[1] != s2.Address() {
		t.Errorf("accumulated Signers = %v, want [s1,s2]", signers)
	}
}

func TestSignAllKinds(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	s := mustSigner(t, testKeyHex)
	ctx := t.Context()
	contract, _ := BuildTriggerSmartContract(cp, ctx, testFrom, testTo, []byte{0x01}, 0)
	deploy, _ := BuildDeploy(cp, ctx, testFrom, DeployParams{Bytecode: []byte{0x60}})
	asset, _ := BuildAssetTransfer(cp, ctx, testFrom, testTo, "1000001", 1)
	for _, st := range []struct {
		name string
		sign func() (Tx, error)
	}{
		{"contract", func() (Tx, error) { return contract.Sign(s) }},
		{"deploy", func() (Tx, error) { return deploy.Sign(s) }},
		{"asset", func() (Tx, error) { return asset.Sign(s) }},
	} {
		tx, err := st.sign()
		if err != nil {
			t.Fatalf("%s.Sign: %v", st.name, err)
		}
		if !tx.IsSigned() {
			t.Errorf("%s: signed copy IsSigned = false", st.name)
		}
	}
}

func TestSignErrors(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	native, _ := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	if _, err := native.Sign(); !tron.HasCode(err, tron.CodeTxNoSigner) {
		t.Errorf("Sign() with no signers: err = %v, want tx.no_signer", err)
	}
	if _, err := native.Sign(nil); !tron.HasCode(err, tron.CodeKeyInvalid) {
		t.Errorf("Sign(nil): err = %v, want key.invalid", err)
	}
}

func TestWithPermissionIDBeforeSignIsHonored(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	native, _ := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	s := mustSigner(t, testKeyHex)
	signed, err := native.WithPermissionID(5).Sign(s)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if got := signed.PermissionID(); got != 5 {
		t.Errorf("signed tx PermissionID = %d, want 5", got)
	}
}
