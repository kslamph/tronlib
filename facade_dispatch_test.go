package tronlib

// The facade is a set of one-line delegations, and that is exactly what makes
// it worth testing: a delegation can point at the wrong subpackage function,
// drop an argument, or start wrapping instead of aliasing, and nothing in the
// subpackages notices. Every user enters here, so a wrong delegation breaks
// them all while the subpackage suites stay green — the v1 failure class the
// facade was built to remove.

import (
	"bytes"
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// facadeKinds builds one transaction of every kind through the facade's own
// client, so the delegation under test is the same wire path a user takes.
func facadeKinds(t *testing.T) map[string]Tx {
	t.Helper()
	c := newFacadeTestClient(t, &fakeFacadeServer{})
	acc := c.Account(facadeFrom)
	ctx := t.Context()

	native, err := acc.TransferTRX(ctx, facadeTo, TRX(1))
	if err != nil {
		t.Fatalf("TransferTRX: %v", err)
	}
	contract, err := tx.BuildTriggerSmartContract(ctx, c.Raw(), facadeFrom, facadeTo, []byte{0x01}, 0)
	if err != nil {
		t.Fatalf("BuildTriggerSmartContract: %v", err)
	}
	deploy, err := acc.Deploy(ctx, tx.DeployParams{Name: "Facade", Bytecode: []byte{0x60, 0x60}})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	asset, err := acc.TransferTRC10(ctx, facadeTo, "1000001", 1)
	if err != nil {
		t.Fatalf("TransferTRC10: %v", err)
	}
	return map[string]Tx{
		"native":   native,
		"contract": contract,
		"deploy":   deploy,
		"asset":    asset,
	}
}

// TestFacadeEncodeDecodeMatchSubpackage: the facade's Encode must produce the
// same envelope the subpackage's does, and Decode must rebuild the same kind
// with the same id.
//
// Failure mode: a delegation wired to a different function, or to a variant
// with a different envelope. Users persist these envelopes for offline
// multi-signing and hand them to another machine; an envelope that only this
// build can read is a stuck multisig.
func TestFacadeEncodeDecodeMatchSubpackage(t *testing.T) {
	for name, orig := range facadeKinds(t) {
		t.Run(name, func(t *testing.T) {
			facadeBlob, ferr := Encode(orig)
			subBlob, serr := tx.Encode(orig)
			if (ferr == nil) != (serr == nil) {
				t.Fatalf("Encode: facade err = %v, subpackage err = %v", ferr, serr)
			}
			if ferr != nil {
				if !sameCode(ferr, serr) {
					t.Errorf("Encode error codes differ: facade %v, subpackage %v", ferr, serr)
				}
				return
			}
			if !bytes.Equal(facadeBlob, subBlob) {
				t.Errorf("Encode produced different bytes:\n facade:      %x\n subpackage: %x", facadeBlob, subBlob)
			}

			facadeTx, err := Decode(facadeBlob)
			if err != nil {
				t.Fatalf("facade Decode: %v", err)
			}
			subTx, err := tx.Decode(subBlob)
			if err != nil {
				t.Fatalf("subpackage Decode: %v", err)
			}
			if facadeTx.Kind() != orig.Kind() || subTx.Kind() != orig.Kind() {
				t.Errorf("kind after Decode: facade %v, subpackage %v, want %v", facadeTx.Kind(), subTx.Kind(), orig.Kind())
			}
			if facadeTx.ID() != orig.ID() {
				t.Errorf("the decoded transaction's id %s != the original's %s", facadeTx.ID(), orig.ID())
			}
			if !proto.Equal(facadeTx.Transaction(), orig.Transaction()) {
				t.Error("the decoded transaction differs from the original")
			}
		})
	}
}

// TestFacadeSignAndSignHashMatchSubpackage: the signing entry points must
// agree with the subpackage's, byte for byte, including which error they
// refuse with.
func TestFacadeSignAndSignHashMatchSubpackage(t *testing.T) {
	s := mustFacadeSigner(t)
	for name, orig := range facadeKinds(t) {
		t.Run(name, func(t *testing.T) {
			facadeDigest, ferr := SignHash(orig)
			subDigest, serr := tx.SignHash(orig)
			if (ferr == nil) != (serr == nil) {
				t.Fatalf("SignHash: facade err = %v, subpackage err = %v", ferr, serr)
			}
			if ferr == nil && !bytes.Equal(facadeDigest, subDigest) {
				t.Errorf("SignHash differs:\n facade:      %x\n subpackage: %x", facadeDigest, subDigest)
			}

			facadeSigned, ferr := Sign(orig, s)
			subSigned, serr := tx.Sign(orig, s)
			if (ferr == nil) != (serr == nil) {
				t.Fatalf("Sign: facade err = %v, subpackage err = %v", ferr, serr)
			}
			if ferr != nil {
				if !sameCode(ferr, serr) {
					t.Errorf("Sign error codes differ: facade %v, subpackage %v", ferr, serr)
				}
				return
			}
			if !proto.Equal(facadeSigned.Transaction(), subSigned.Transaction()) {
				t.Error("Sign produced different bytes through the facade and the subpackage")
			}

			// The offline path, through the facade: sign the exported
			// digest out of process and attach the result.
			offDigest, err := SignHash(orig)
			if err != nil {
				t.Fatalf("SignHash: %v", err)
			}
			sig, err := s.Sign(offDigest)
			if err != nil {
				t.Fatalf("sign: %v", err)
			}
			attached, err := AttachSignature(orig, s.Address(), sig)
			if err != nil {
				t.Fatalf("AttachSignature: %v", err)
			}
			if !proto.Equal(attached.Transaction(), facadeSigned.Transaction()) {
				t.Error("the facade's offline path and its in-process Sign disagree")
			}
			// A signature made for a different transaction must still be
			// refused through the facade. The donor is a transaction of a
			// different kind, so its digest genuinely differs: every
			// builder here answers from the same canned extentions, so two
			// transfers would carry identical raw_data and the check would
			// be vacuous.
			kinds := facadeKinds(t)
			var other Tx
			for kind, candidate := range kinds {
				if kind != name {
					other = candidate
					break
				}
			}
			otherDigest, err := SignHash(other)
			if err != nil {
				t.Fatalf("SignHash(other): %v", err)
			}
			foreign, err := s.Sign(otherDigest)
			if err != nil {
				t.Fatalf("sign other: %v", err)
			}
			if _, err := AttachSignature(orig, s.Address(), foreign); !tron.HasCode(err, tron.CodeKeyInvalid) {
				t.Errorf("the facade accepted a signature made for another transaction: err = %v, want key.invalid", err)
			}
		})
	}
}

// TestFacadeRefusesNilLikeTheSubpackage: the nil-input behaviour must match
// exactly. A facade that panicked (or invented a code) on nil would be the
// one place a user learns the error taxonomy does not hold.
func TestFacadeRefusesNilLikeTheSubpackage(t *testing.T) {
	s := mustFacadeSigner(t)
	cases := []struct {
		name string
		f    func() error
		s    func() error
	}{
		{"Encode", func() error { _, err := Encode(nil); return err }, func() error { _, err := tx.Encode(nil); return err }},
		{"SignHash", func() error { _, err := SignHash(nil); return err }, func() error { _, err := tx.SignHash(nil); return err }},
		{"Sign", func() error { _, err := Sign(nil, s); return err }, func() error { _, err := tx.Sign(nil, s); return err }},
		{"AttachSignature", func() error { _, err := AttachSignature(nil, s.Address(), make([]byte, 65)); return err },
			func() error { _, err := tx.AttachSignature(nil, s.Address(), make([]byte, 65)); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fErr, sErr := tc.f(), tc.s()
			if fErr == nil || sErr == nil {
				t.Fatalf("nil must be refused: facade %v, subpackage %v", fErr, sErr)
			}
			if !sameCode(fErr, sErr) {
				t.Errorf("error codes differ: facade %v, subpackage %v", fErr, sErr)
			}
		})
	}
}

// TestFacadeOperationsRoundTrip: the active-permission bitmap and its inverse
// must agree with the subpackage's, and round-trip every contract type.
//
// Failure mode: a wrong bit index. A permission built from the wrong bitmap
// authorises the wrong operations — an account permission that can transfer
// but not vote, or the reverse — and the node accepts it silently.
func TestFacadeOperationsRoundTrip(t *testing.T) {
	all := []ContractType{
		TypeAccountCreate, TypeTransfer, TypeTransferAsset,
		TypeFreezeBalanceV2, TypeCreateSmartContract, TypeUpdateSetting,
	}
	bitmap, err := OperationsBitmap(all...)
	if err != nil {
		t.Fatalf("OperationsBitmap: %v", err)
	}
	subBitmap, err := tx.OperationsBitmap(all...)
	if err != nil {
		t.Fatalf("tx.OperationsBitmap: %v", err)
	}
	if !bytes.Equal(bitmap, subBitmap) {
		t.Errorf("bitmap differs from the subpackage's:\n facade:      %x\n subpackage: %x", bitmap, subBitmap)
	}
	if len(bitmap) != 32 {
		t.Errorf("bitmap is %d bytes, want the 32-byte word the permission carries", len(bitmap))
	}
	back, err := OperationsList(bitmap)
	if err != nil {
		t.Fatalf("OperationsList: %v", err)
	}
	if len(back) != len(all) {
		t.Fatalf("round-trip lost types: got %v, want %v", back, all)
	}
	seen := map[ContractType]bool{}
	for _, ct := range back {
		seen[ct] = true
	}
	for _, ct := range all {
		if !seen[ct] {
			t.Errorf("round-trip lost %v", ct)
		}
	}
	// A single type must not light up its neighbours.
	one, err := OperationsBitmap(TypeFreezeBalanceV2)
	if err != nil {
		t.Fatalf("OperationsBitmap(one type): %v", err)
	}
	list, err := OperationsList(one)
	if err != nil {
		t.Fatalf("OperationsList: %v", err)
	}
	if len(list) != 1 || list[0] != TypeFreezeBalanceV2 {
		t.Errorf("a one-type bitmap decoded to %v, want [%v]", list, TypeFreezeBalanceV2)
	}
	// A bitmap of the wrong length is refused, and refused the same way
	// through both doors: the Hint names the expected width, which is what
	// a caller needs to build a correct permission.
	_, fe := OperationsList(bitmap[:8])
	_, se := tx.OperationsList(subBitmap[:8])
	if fe == nil || se == nil {
		t.Fatalf("a short bitmap must be refused: facade %v, subpackage %v", fe, se)
	}
	if !sameCode(fe, se) {
		t.Errorf("OperationsList on a short bitmap: facade %v, subpackage %v", fe, se)
	}
	if !tron.HasCode(fe, tron.CodeTxInvalidArgument) {
		t.Errorf("short bitmap: err = %v, want tx.invalid_argument", fe)
	}
	if _, err := OperationsBitmap(ContractType(9999)); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("OperationsBitmap(9999): err = %v, want tx.invalid_argument", err)
	}
}

// TestFacadeChainParamsOfReadsTheNode: the facade must return the node's live
// governance parameters, not zeroes and not constants, and must refuse to
// price a fee the node did not report.
//
// Failure mode: a delegation that drops the client, or a wrapper that
// returns a zero struct. A zero permission-update fee understates the cost
// of every multisig the user builds.
func TestFacadeChainParamsOfReadsTheNode(t *testing.T) {
	c := newFacadeTestClient(t, &fakeFacadeServer{})
	params, err := ChainParamsOf(t.Context(), c.Raw())
	if err != nil {
		t.Fatalf("ChainParamsOf: %v", err)
	}
	if params.UpdateAccountPermissionFee != 100*TRX(1) {
		t.Errorf("UpdateAccountPermissionFee = %s, want the node's 100 TRX", params.UpdateAccountPermissionFee)
	}
	if params.MultiSignFee != TRX(1) {
		t.Errorf("MultiSignFee = %s, want the node's 1 TRX", params.MultiSignFee)
	}
	if params.UnfreezeDelayDays != 14 {
		t.Errorf("UnfreezeDelayDays = %d, want the node's 14", params.UnfreezeDelayDays)
	}
	if params.MaxDelegateLockPeriod != 107520 {
		t.Errorf("MaxDelegateLockPeriod = %d, want the node's 107520", params.MaxDelegateLockPeriod)
	}
	if len(params.Missing) != 0 {
		t.Errorf("Missing = %v, want empty: the fake reports every parameter", params.Missing)
	}
}

// TestFacadeChainParamsOfRefusesAnUnpricedFee: a node that omits a fee
// parameter must fail the read, not price the fee as zero.
func TestFacadeChainParamsOfRefusesAnUnpricedFee(t *testing.T) {
	c := newFacadeTestClient(t, &fakeFacadeServer{NoFeeChainParams: true})
	if _, err := ChainParamsOf(t.Context(), c.Raw()); !tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("ChainParamsOf without a fee parameter: err = %v, want contract.bad_metadata", err)
	}
}

// TestFacadeDialOptions: the Dial options must reach the client, the network
// declaration must be recorded (last one wins), and a malformed endpoint must
// still be refused — the option loop must not swallow the endpoint error.
func TestFacadeDialOptions(t *testing.T) {
	t.Run("network is recorded, last option wins", func(t *testing.T) {
		c, err := Dial(t.Context(), "grpc://127.0.0.1:1",
			WithNetwork(Nile), WithNetwork(Shasta), WithTimeout(time.Second), WithPool(1, 2))
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		defer func() { _ = c.Close() }()
		if c.network != Shasta {
			t.Errorf("recorded network = %q, want the last option's %q", c.network, Shasta)
		}
		if c.Endpoint() != "grpc://127.0.0.1:1" {
			t.Errorf("Endpoint = %q, want the endpoint Dial was given", c.Endpoint())
		}
	})
	// Sizes <= 0 are documented to fall back to the defaults rather than
	// fail, and they must not panic on the way to the pool.
	//
	// NOT asserted here: init > max (e.g. WithPool(3, 1)). That combination
	// is refused by rpc.Dial, but reported as chain.connection — the code
	// whose remedy is "retry or switch endpoint", which can never help a
	// bad argument. Left as-is deliberately: changing an rpc error code is
	// a compatibility decision for the maintainer, not a test's to make.
	t.Run("degenerate pool sizes fall back instead of failing", func(t *testing.T) {
		for _, sizes := range [][2]int{{0, 0}, {-1, -1}, {2, 2}} {
			c, err := Dial(t.Context(), "grpc://127.0.0.1:1", WithPool(sizes[0], sizes[1]))
			if err != nil {
				t.Errorf("Dial WithPool%v: %v", sizes, err)
				continue
			}
			_ = c.Close()
		}
	})
	t.Run("a malformed endpoint is refused", func(t *testing.T) {
		for _, endpoint := range []string{"http://node:50051", "grpc://", "", "not a url"} {
			if _, err := Dial(t.Context(), endpoint); !tron.HasCode(err, tron.CodeChainConnection) {
				t.Errorf("Dial(%q): err = %v, want chain.connection", endpoint, err)
			}
		}
	})
	t.Run("Close is idempotent", func(t *testing.T) {
		c, err := Dial(t.Context(), "grpc://127.0.0.1:1")
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		if err := c.Close(); err != nil {
			t.Errorf("first Close: %v", err)
		}
		if err := c.Close(); err != nil {
			t.Errorf("second Close: %v, want nil (Close is documented as idempotent)", err)
		}
	})
}

// GetChainParameters is the governance-parameter read the facade's
// ChainParamsOf performs. Canned but deliberately non-zero: a delegation that
// stopped reading the node would show up as zeroes here.
func (f *fakeFacadeServer) GetChainParameters(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
	params := []*core.ChainParameters_ChainParameter{
		{Key: "getUnfreezeDelayDays", Value: 14},
		{Key: "getMaxDelegateLockPeriod", Value: 107520},
	}
	if !f.NoFeeChainParams {
		params = append(params,
			&core.ChainParameters_ChainParameter{Key: "getUpdateAccountPermissionFee", Value: 100_000_000}, // 100 TRX in SUN
			&core.ChainParameters_ChainParameter{Key: "getMultiSignFee", Value: 1_000_000},                 // 1 TRX in SUN
		)
	}
	return &core.ChainParameters{ChainParameter: params}, nil
}

func mustFacadeSigner(t *testing.T) Signer {
	t.Helper()
	s, err := KeyFromHex(facadeKeyHex)
	if err != nil {
		t.Fatalf("KeyFromHex: %v", err)
	}
	return s
}

// sameCode reports whether two errors carry the same tron code, treating a
// nil on either side as a mismatch. It is the comparison every parity test
// above makes: the facade must refuse exactly what the subpackage refuses.
func sameCode(a, b error) bool {
	ta, tb := tron.ErrorOf(a), tron.ErrorOf(b)
	if ta == nil || tb == nil {
		return false
	}
	return ta.Code == tb.Code
}
