package tx

// The offline-signing path: SignHash (export the digest) → sign on a device
// that never sees the private key → WithSignature/AttachSignature (attach the
// raw 65-byte [R || S || V]). These tests pin the one property that makes the
// path safe at all — a transaction assembled offline must be indistinguishable
// from one signed in-process, and must carry exactly the signatures its signer
// claims to.

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/kslamph/tronlib/v2/tron"
)

// TestOfflineSigningMatchesInProcessSign: for every kind, the two ways of
// signing must produce the same bytes.
//
// Failure mode this catches: any divergence between the digest SignHash
// exports and the one Sign signs over (a re-marshal, a field added to
// raw_data, an encoding change in the signature). Such a transaction passes
// every local check — WithSignature accepts it, Signers() reports the right
// address, the ID looks right — and is only rejected by the node, after the
// user has already paid to assemble it.
func TestOfflineSigningMatchesInProcessSign(t *testing.T) {
	s := mustSigner(t, testKeyHex)
	for name, orig := range buildAllKinds(t) {
		t.Run(name, func(t *testing.T) {
			inProcess, err := Sign(orig, s)
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}
			digest, err := SignHash(orig)
			if err != nil {
				t.Fatalf("SignHash: %v", err)
			}
			sig, err := s.Sign(digest)
			if err != nil {
				t.Fatalf("sign the exported digest: %v", err)
			}
			offline := attachPerKind(t, orig, s.Address(), sig)

			if !proto.Equal(inProcess.Transaction(), offline.Transaction()) {
				t.Errorf("offline signing produced different bytes:\n in-process: %v\n offline:     %v",
					inProcess.Transaction(), offline.Transaction())
			}
			if inProcess.ID() != offline.ID() {
				t.Errorf("ID differs: in-process %s, offline %s", inProcess.ID(), offline.ID())
			}
			if !sameSigners(t, inProcess, s.Address()) {
				t.Error("Signers() differs between the in-process and the offline path")
			}
		})
	}
}

// TestOfflineSigningComposesAcrossHandoff: multi-sig through the offline
// path only — party A signs and hands the envelope to party B, who attaches
// without ever holding a key. The result must equal the in-process two-key
// signature, in the same order.
//
// Failure mode: AttachSignature appending in the wrong position, or dropping
// a signature already present. TRON validates the approved list by position,
// so a reordered multi-sig is rejected at broadcast even though both
// signatures are individually valid.
func TestOfflineSigningComposesAcrossHandoff(t *testing.T) {
	a := mustSigner(t, testKeyHex)
	b := mustSigner(t, testKeyHex2)
	for name, orig := range buildAllKinds(t) {
		t.Run(name, func(t *testing.T) {
			// Party A signs in-process (it owns the key), then the envelope
			// travels to party B, who signs the same digest offline.
			afterA, err := Sign(orig, a)
			if err != nil {
				t.Fatalf("Sign(a): %v", err)
			}
			blob, err := Encode(afterA)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			received, err := Decode(blob)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if !received.IsSigned() {
				t.Fatal("the decoded envelope lost party A's signature")
			}
			digest, err := SignHash(received)
			if err != nil {
				t.Fatalf("SignHash: %v", err)
			}
			sigB, err := b.Sign(digest)
			if err != nil {
				t.Fatalf("sign digest: %v", err)
			}
			final, err := AttachSignature(received, b.Address(), sigB)
			if err != nil {
				t.Fatalf("AttachSignature: %v", err)
			}
			want := []tron.Address{a.Address(), b.Address()}
			if !sameSigners(t, final, want...) {
				t.Error("signers are not [A, B] in order after the handoff")
			}
			inProcess, err := Sign(orig, a, b)
			if err != nil {
				t.Fatalf("Sign(a, b): %v", err)
			}
			if !proto.Equal(inProcess.Transaction(), final.Transaction()) {
				t.Errorf("handoff bytes differ from the in-process two-key signature:\n in-process: %v\n handoff:     %v",
					inProcess.Transaction(), final.Transaction())
			}
			// A third attach by a signer already present must be refused:
			// the same address twice invalidates the whole transaction.
			digest2, _ := SignHash(final)
			dup, _ := a.Sign(digest2)
			if _, err := AttachSignature(final, a.Address(), dup); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				t.Errorf("re-attaching party A's signature: err = %v, want tx.already_signed", err)
			}
		})
	}
}

// TestAttachSignatureRejectsSignatureFromAnotherTransaction: the realistic
// offline mistake — a device is handed transaction A, and its signature is
// attached to transaction B (a rebuilt draft, a different amount, the wrong
// row of a batch). The signature recovers to a perfectly valid address; it
// simply does not cover this raw_data.
//
// Failure mode: if this is accepted, the transaction carries a signature
// that authorizes nothing. Every local accessor looks correct, and the node
// rejects it as DUP_TRANSACTION_ERROR / signature error.
func TestAttachSignatureRejectsSignatureFromAnotherTransaction(t *testing.T) {
	other := mustSigner(t, testKeyHex2)
	kinds := buildAllKinds(t)

	// The donor is the native target with a different expiration: same
	// kind, same shape, a raw_data the targets' digests do not cover. The
	// fake node returns a canned transfer extention, so building a second
	// transfer would hand back the identical transaction and the test
	// would be vacuous.
	donor, err := kinds["native"].(*NativeTx).WithExpiration(2 * time.Hour)
	if err != nil {
		t.Fatalf("WithExpiration(donor): %v", err)
	}
	for name, target := range kinds {
		if target.ID() == donor.ID() {
			t.Fatalf("the donor transaction collides with the %s target; the test would be vacuous", name)
		}
	}

	digest, err := SignHash(donor)
	if err != nil {
		t.Fatalf("SignHash(donor): %v", err)
	}
	foreign, err := other.Sign(digest)
	if err != nil {
		t.Fatalf("sign donor digest: %v", err)
	}

	for name, target := range kinds {
		t.Run(name, func(t *testing.T) {
			_, err := AttachSignature(target, other.Address(), foreign)
			if !tron.HasCode(err, tron.CodeKeyInvalid) {
				t.Fatalf("attaching a signature made for a different transaction: err = %v, want key.invalid", err)
			}
			te := tron.ErrorOf(err)
			if te == nil || te.Hint == "" {
				t.Error("the refusal carries no Hint; the caller needs to know the signature does not cover this raw_data")
			}
		})
	}
}

// TestAttachSignatureRejectsNilAndTypedNil: the escape hatches accept a Tx
// interface, so a nil *NativeTx arrives as a non-nil interface holding a nil
// pointer. Both shapes must produce a typed error rather than a panic, because
// this function is reachable from the facade with whatever the caller had.
//
// Failure mode: a panic in a helper an agent calls with a possibly-nil
// transaction, instead of the tx.invalid the rest of the package returns.
func TestAttachSignatureRejectsNilAndTypedNil(t *testing.T) {
	addr := mustSigner(t, testKeyHex).Address()
	for _, tc := range []struct {
		name string
		tx   Tx
	}{
		{"nil interface", nil},
		{"typed nil *NativeTx", (*NativeTx)(nil)},
		{"typed nil *ContractTx", (*ContractTx)(nil)},
		{"typed nil *DeployTx", (*DeployTx)(nil)},
		{"typed nil *AssetTx", (*AssetTx)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := AttachSignature(tc.tx, addr, make([]byte, 65)); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
				t.Errorf("err = %v, want tx.invalid_argument (never a panic)", err)
			}
		})
	}
	if _, err := SignHash(nil); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("SignHash(nil): err = %v, want tx.invalid_argument", err)
	}
	if _, err := Sign(nil, mustSigner(t, testKeyHex)); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("Sign(nil): err = %v, want tx.invalid_argument", err)
	}
	if _, err := Encode(nil); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("Encode(nil): err = %v, want tx.invalid_argument", err)
	}
}

// TestWithSignatureKeepsReceiverUnsigned: attaching offline leaves the
// receiver untouched, exactly as Sign does. A caller that keeps a draft around
// (to hand to a second signer, or to re-read the ID) must not find it signed.
func TestWithSignatureKeepsReceiverUnsigned(t *testing.T) {
	s := mustSigner(t, testKeyHex)
	for name, orig := range buildAllKinds(t) {
		t.Run(name, func(t *testing.T) {
			digest, err := SignHash(orig)
			if err != nil {
				t.Fatalf("SignHash: %v", err)
			}
			sig, err := s.Sign(digest)
			if err != nil {
				t.Fatalf("sign: %v", err)
			}
			attached := attachPerKind(t, orig, s.Address(), sig)
			if orig.IsSigned() {
				t.Error("the receiver was signed in place; attaching must return a copy")
			}
			if !attached.IsSigned() {
				t.Error("the returned copy is not signed")
			}
		})
	}
}

// attachPerKind calls the concrete per-kind WithSignature, asserting the
// result's dynamic type matches the receiver's. A kind that lost its
// WithSignature would have to be reached through the interface helper, where
// a wrong-type return is invisible.
func attachPerKind(t *testing.T, orig Tx, addr tron.Address, sig []byte) Tx {
	t.Helper()
	switch v := orig.(type) {
	case *NativeTx:
		got, err := v.WithSignature(addr, sig)
		if err != nil {
			t.Fatalf("NativeTx.WithSignature: %v", err)
		}
		return got
	case *ContractTx:
		got, err := v.WithSignature(addr, sig)
		if err != nil {
			t.Fatalf("ContractTx.WithSignature: %v", err)
		}
		return got
	case *DeployTx:
		got, err := v.WithSignature(addr, sig)
		if err != nil {
			t.Fatalf("DeployTx.WithSignature: %v", err)
		}
		return got
	case *AssetTx:
		got, err := v.WithSignature(addr, sig)
		if err != nil {
			t.Fatalf("AssetTx.WithSignature: %v", err)
		}
		return got
	default:
		t.Fatalf("unexpected kind %T", orig)
		return nil
	}
}

func sameSigners(t *testing.T, got Tx, want ...tron.Address) bool {
	t.Helper()
	signers, err := got.Signers()
	if err != nil {
		t.Fatalf("Signers: %v", err)
	}
	if len(signers) != len(want) {
		t.Errorf("signers = %v, want %v", signers, want)
		return false
	}
	for i := range want {
		if signers[i] != want[i] {
			t.Errorf("signers = %v, want %v (order matters to the node)", signers, want)
			return false
		}
	}
	return true
}
