package rpc

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// notAnAddress is 2 bytes where Witnesses' decoder requires a 0x41-prefixed
// 21-byte address.
var notAnAddress = []byte{0x01, 0x02}

// mustTestAddr builds a valid 0x41-prefixed address with every payload byte
// set to fill. Bytes() returns a copy, so an address cannot be assembled by
// copying into its own Bytes(); it has to go through the constructor.
func mustTestAddr(t *testing.T, fill byte) tron.Address {
	t.Helper()
	b := append([]byte{0x41}, bytes.Repeat([]byte{fill}, 20)...)
	a, err := tron.AddressFromBytes(b)
	if err != nil {
		t.Fatalf("building the fixture address: %v", err)
	}
	return a
}

// TestWitnessesDecodesTheWholeList pins the happy decode: the wrapper exists to
// turn raw bytes into addresses, so the test asserts the decode, not just that
// the call returned.
func TestWitnessesDecodesTheWholeList(t *testing.T) {
	addr := mustTestAddr(t, 0x11)

	srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
		"GetPaginatedNowWitnessList": func(_ context.Context, _ any) (any, error) {
			return &api.WitnessList{Witnesses: []*core.Witness{
				{Address: addr.Bytes(), VoteCount: 123, IsJobs: true},
				{Address: addr.Bytes(), VoteCount: 456},
			}}, nil
		},
	}}
	c := newBufconnClient(t, srv, time.Second)

	got, err := Witnesses(c, context.Background(), 10, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d witnesses, want 2", len(got))
	}
	if got[0].Address != addr || got[0].VoteCount != 123 || !got[0].IsJobs {
		t.Errorf("witness 0 = %+v, want %s/123/jobs", got[0], addr)
	}
	if got[1].VoteCount != 456 || got[1].IsJobs {
		t.Errorf("witness 1 = %+v, want 456/not-jobs", got[1])
	}
}

// TestWitnessesRefusesCorruptAddress is the mutation that motivated it.
//
// Witnesses decodes a node-supplied address. Skipping a witness it cannot
// decode and returning the rest looked harmless and passed the entire suite —
// but it turns a corrupt node answer into a *shorter* witness list, and the
// caller cannot tell "the node listed 27 candidates" from "the node listed 27
// and one of them was unreadable". Pagination then shifts and the caller votes
// against a list it never actually received. So a corrupt address must fail the
// whole read, naming the code, rather than degrade quietly.
func TestWitnessesRefusesCorruptAddress(t *testing.T) {
	good := mustTestAddr(t, 0x11)

	srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
		"GetPaginatedNowWitnessList": func(_ context.Context, _ any) (any, error) {
			return &api.WitnessList{Witnesses: []*core.Witness{
				{Address: good.Bytes(), VoteCount: 1},
				{Address: notAnAddress, VoteCount: 2}, // corrupt: 2 bytes
				{Address: good.Bytes(), VoteCount: 3},
			}}, nil
		},
	}}
	c := newBufconnClient(t, srv, time.Second)

	got, err := Witnesses(c, context.Background(), 0, 0)
	if err == nil {
		t.Fatalf("a corrupt witness address must fail the read, got %d witnesses", len(got))
	}
	if !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("err = %v, want address.invalid", err)
	}
	// No partial list: a caller must not be handed the witnesses that did
	// decode, because it cannot tell the list is incomplete.
	if got != nil {
		t.Errorf("got %d witnesses alongside the error, want none", len(got))
	}
	// The decoder's own error must survive as the Cause rather than being
	// flattened, so the caller can see how far the value was off.
	var te *tron.Error
	if !errors.As(err, &te) || te.Cause == nil {
		t.Fatalf("err = %v, want a *tron.Error carrying its Cause", err)
	}
}

// TestWitnessesEmptyListIsNotAnError pins that the guard above is not simply a
// refusal to decode: a node answering with no witnesses is a valid answer and
// must come back as an empty list with no error, not nil-vs-empty confusion.
func TestWitnessesEmptyListIsNotAnError(t *testing.T) {
	srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
		"GetPaginatedNowWitnessList": func(_ context.Context, _ any) (any, error) {
			return &api.WitnessList{}, nil
		},
	}}
	c := newBufconnClient(t, srv, time.Second)

	got, err := Witnesses(c, context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("an empty witness list is valid, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d witnesses, want 0", len(got))
	}
}

// TestWitnessesPropagatesRPCFailure keeps the corrupt-decode test honest: the
// guard above is about malformed values, and a transport failure must still
// surface as its own code rather than being reported as a decode problem.
func TestWitnessesPropagatesRPCFailure(t *testing.T) {
	srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
		"GetPaginatedNowWitnessList": func(_ context.Context, _ any) (any, error) {
			return nil, errors.New("node down")
		},
	}}
	c := newBufconnClient(t, srv, time.Second)

	got, err := Witnesses(c, context.Background(), 0, 0)
	if err == nil {
		t.Fatal("a failing RPC must surface")
	}
	if !tron.HasCode(err, tron.CodeRPCMethodFailed) {
		t.Errorf("err = %v, want rpc.method_failed", err)
	}
	if got != nil {
		t.Errorf("got %d witnesses alongside the error, want none", len(got))
	}
}
