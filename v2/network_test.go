package tronlib

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/v2/tron"
)

// mustHex decodes a hex string or fails the test.
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("mustHex(%q): %v", s, err)
	}
	return b
}

// TestVerifyNetworkMatchesConfigured proves each known public network's own
// genesis block id verifies against its configured declaration.
func TestVerifyNetworkMatchesConfigured(t *testing.T) {
	for _, n := range []Network{Mainnet, Nile, Shasta} {
		t.Run(string(n), func(t *testing.T) {
			f := &fakeFacadeServer{}
			f.BlockByNum2 = func(ctx context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
				if in.GetNum() != 0 {
					t.Errorf("VerifyNetwork read block %d, want 0", in.GetNum())
				}
				return &api.BlockExtention{Blockid: mustHex(t, genesisID[n])}, nil
			}
			c := newFacadeTestClient(t, f)
			c.network = n
			if err := c.VerifyNetwork(context.Background()); err != nil {
				t.Fatalf("VerifyNetwork(%s): %v", n, err)
			}
		})
	}
}

// TestVerifyNetworkMismatch proves a declared network whose observed genesis
// disagrees yields chain.network_mismatch, with both hashes in the hint so the
// operator can paste the new genesis.
func TestVerifyNetworkMismatch(t *testing.T) {
	f := &fakeFacadeServer{}
	f.BlockByNum2 = func(ctx context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
		return &api.BlockExtention{Blockid: mustHex(t, genesisID[Mainnet])}, nil
	}
	c := newFacadeTestClient(t, f)
	c.network = Nile

	err := c.VerifyNetwork(context.Background())
	if err == nil {
		t.Fatal("VerifyNetwork(Nile with mainnet genesis) = nil, want chain.network_mismatch")
	}
	if !tron.HasCode(err, tron.CodeChainNetworkMismatch) {
		t.Fatalf("VerifyNetwork error = %v, want chain.network_mismatch", err)
	}
	hint := err.(*tron.Error).Hint
	if !strings.Contains(hint, "nile") {
		t.Errorf("hint %q does not name the configured network", hint)
	}
	if !strings.Contains(hint, genesisID[Mainnet]) || !strings.Contains(hint, genesisID[Nile]) {
		t.Errorf("hint %q does not carry both the observed and expected genesis ids", hint)
	}
}

// TestVerifyNetworkUnknownDeclarationFailsClosed proves a declared network the
// table does not know is a mismatch, not a pass: no entry matching must never
// read as "verified".
func TestVerifyNetworkUnknownDeclarationFailsClosed(t *testing.T) {
	f := &fakeFacadeServer{}
	f.BlockByNum2 = func(ctx context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
		return &api.BlockExtention{Blockid: mustHex(t, genesisID[Mainnet])}, nil
	}
	c := newFacadeTestClient(t, f)
	c.network = Network("bogus")

	err := c.VerifyNetwork(context.Background())
	if err == nil || !tron.HasCode(err, tron.CodeChainNetworkMismatch) {
		t.Fatalf("VerifyNetwork(bogus) = %v, want chain.network_mismatch (fail closed)", err)
	}
}

// TestVerifyNetworkPrivateAndUndeclaredSkipRead proves Private and the
// undeclared zero value make no node read at all — there is no expectation to
// contradict.
func TestVerifyNetworkPrivateAndUndeclaredSkipRead(t *testing.T) {
	for _, n := range []Network{Private, ""} {
		f := &fakeFacadeServer{}
		c := newFacadeTestClient(t, f)
		c.network = n
		if err := c.VerifyNetwork(context.Background()); err != nil {
			t.Fatalf("VerifyNetwork(%q) = %v, want nil", n, err)
		}
		if f.blockByNumCalls != 0 {
			t.Errorf("VerifyNetwork(%q) read block 0 %d times, want 0", n, f.blockByNumCalls)
		}
	}
}

// TestVerifyNetworkReadErrorIsNotMismatch proves a failure to read block 0
// surfaces the read error, never a fabricated mismatch — a mismatch claim the
// client cannot substantiate is worse than an honest failure. A node-returned
// error takes the rpc.method_failed path (the documented rpc boundary in
// PHASE2.md); a pool/dial failure would be chain.connection. Either way the
// result must not be chain.network_mismatch.
func TestVerifyNetworkReadErrorIsNotMismatch(t *testing.T) {
	f := &fakeFacadeServer{}
	f.BlockByNum2 = func(ctx context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
		return nil, &tron.Error{Code: tron.CodeChainConnection, Op: "fake.GetBlockByNum2"}
	}
	c := newFacadeTestClient(t, f)
	c.network = Mainnet

	err := c.VerifyNetwork(context.Background())
	if err == nil {
		t.Fatal("VerifyNetwork with a failing read = nil, want the read error")
	}
	if tron.HasCode(err, tron.CodeChainNetworkMismatch) {
		t.Fatalf("read failure reported as chain.network_mismatch: %v", err)
	}
	if !tron.HasCode(err, tron.CodeRPCMethodFailed) {
		t.Fatalf("VerifyNetwork read error = %v, want it propagated (rpc.method_failed for a node-returned error)", err)
	}
}

// TestNetworkRoundTripsWithNetworkOption proves Dial records the declared
// network without any I/O (Dial is lazy).
func TestNetworkRoundTripsWithNetworkOption(t *testing.T) {
	c, err := Dial(context.Background(), "grpc://127.0.0.1:1", WithNetwork(Nile))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer func() { _ = c.Close() }()
	if got := c.Network(); got != Nile {
		t.Fatalf("Network() = %q, want %q", got, Nile)
	}
}
