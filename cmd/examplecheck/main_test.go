package main

// Hermetic tests for examplecheck's pure helpers. The harness itself needs a
// live node (that is its purpose), but its classification and key handling are
// deterministic and are pinned here so a wrong "this is fine" call cannot make
// a live run silently pass.

import (
	"testing"

	tronlib "github.com/kslamph/tronlib/v2"
	"github.com/kslamph/tronlib/v2/tron"
)

func TestNoteWorthyClassifiesNodeRejections(t *testing.T) {
	// State-dependent node rejections are the documented outcome of a
	// spend-free run: they must be notes, not failures.
	for _, code := range []tron.Code{
		tron.CodeAccountInsufficientBalance,
		tron.CodeAccountInsufficientEnergy,
		tron.CodeAccountInsufficientBandwidth,
		tron.CodeAccountPermissionDenied,
		tron.CodeTxInvalidArgument,
		tron.CodeContractNotFound,
		tron.CodeReceiptReverted,
		tron.CodeReceiptOutOfEnergy,
	} {
		err := &tron.Error{Code: code, Op: "test"}
		if !noteWorthy(err) {
			t.Errorf("noteWorthy(%s) = false, want true", code)
		}
	}
	// SDK and transport failures are failures.
	for _, code := range []tron.Code{
		tron.CodeRPCMethodFailed,
		tron.CodeChainConnection,
		tron.CodeChainTimeout,
		tron.CodeTxAlreadySigned,
		tron.CodeContractArgMismatch,
	} {
		err := &tron.Error{Code: code, Op: "test"}
		if noteWorthy(err) {
			t.Errorf("noteWorthy(%s) = true, want false", code)
		}
	}
	if noteWorthy(nil) {
		t.Error("noteWorthy(nil) = true, want false")
	}
	if noteWorthy(&tron.Error{Code: tron.CodeContractBadMetadata, Op: "test"}) {
		t.Error("bad metadata must be a failure: the node's answer was not in the documented shape")
	}
}

func TestSignerFromCertifiesTheKeySource(t *testing.T) {
	// An explicit hex key is used as-is and reported as not generated.
	const hexKey = "0101010101010101010101010101010101010101010101010101010101010101"
	want, err := tronlib.KeyFromHex(hexKey)
	if err != nil {
		t.Fatalf("KeyFromHex: %v", err)
	}
	got, generated, err := signerFrom(hexKey)
	if err != nil {
		t.Fatalf("signerFrom(hex): %v", err)
	}
	if generated != "" {
		t.Errorf("generated marker = %q, want empty for an explicit key", generated)
	}
	if got.Address() != want.Address() {
		t.Errorf("address = %s, want %s", got.Address(), want.Address())
	}

	// No key means a fresh signer, and the run says so: an unfunded account is
	// why some steps come back as notes.
	fresh, generated, err := signerFrom("")
	if err != nil {
		t.Fatalf("signerFrom(fresh): %v", err)
	}
	if generated == "" {
		t.Error("generated marker is empty for a fresh key; the transcript must show the account is unfunded")
	}
	if fresh.Address().IsZero() {
		t.Error("fresh signer has the zero address")
	}
}

func TestRebalanceRoundingKeepsTransfersTidy(t *testing.T) {
	// Top-ups round up so the payee always reaches its float; returns round
	// down so the payee always keeps the reserve that pays for the return tx
	// itself. Both round to 0.1 TRX.
	cases := []struct {
		name string
		in   tron.SUN
		want tron.SUN
		up   bool
	}{
		{"exact stays", 2_000_000, 2_000_000, true},
		{"top-up rounds up", 1_950_000, 2_000_000, true},
		{"top-up to the step", 100_001, 200_000, true},
		{"return rounds down", 1_999_999, 1_900_000, false},
		{"return below a step is nothing", 50_000, 0, false},
		{"zero is zero", 0, 0, true},
		{"negative is nothing", -5, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got tron.SUN
			if tc.up {
				got = ceilTo(tc.in, rebalanceStep)
			} else {
				got = floorTo(tc.in, rebalanceStep)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestStepBookkeepingSeparatesNotesFromFailures(t *testing.T) {
	c := &checker{}
	c.step("ok", func() error { return nil })
	c.step("note", func() error { return &tron.Error{Code: tron.CodeAccountInsufficientEnergy, Op: "test"} })
	c.step("fail", func() error { return &tron.Error{Code: tron.CodeRPCMethodFailed, Op: "test"} })

	if c.ok != 1 || c.notes != 1 || c.failed != 1 {
		t.Errorf("counts = ok %d, notes %d, failed %d; want 1/1/1", c.ok, c.notes, c.failed)
	}
}
