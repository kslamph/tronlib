package tx

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// buildAllKinds returns one builder-produced transaction per kind, for the
// round-trip matrix.
func buildAllKinds(t *testing.T) map[string]Tx {
	t.Helper()
	cp := newTxTestClient(t, &fakeWalletServer{})
	ctx := t.Context()
	native, err := BuildTransfer(ctx, cp, testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	contract, err := BuildTriggerSmartContract(ctx, cp, testFrom, testTo, []byte{0x01}, 0)
	if err != nil {
		t.Fatalf("BuildTriggerSmartContract: %v", err)
	}
	deploy, err := BuildDeploy(ctx, cp, testFrom, DeployParams{Bytecode: []byte{0x60}})
	if err != nil {
		t.Fatalf("BuildDeploy: %v", err)
	}
	asset, err := BuildAssetTransfer(ctx, cp, testFrom, testTo, "1000001", 1)
	if err != nil {
		t.Fatalf("BuildAssetTransfer: %v", err)
	}
	return map[string]Tx{
		"native":   native,
		"contract": contract,
		"deploy":   deploy,
		"asset":    asset,
	}
}

// TestPortableRoundTripAllKinds: Encode/Decode preserves the concrete kind,
// the raw bytes and therefore the txid for every builder-produced kind.
func TestPortableRoundTripAllKinds(t *testing.T) {
	for name, orig := range buildAllKinds(t) {
		t.Run(name, func(t *testing.T) {
			blob, err := Encode(orig)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			got, err := Decode(blob)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if got.Kind() != orig.Kind() {
				t.Errorf("Kind = %v, want %v", got.Kind(), orig.Kind())
			}
			if got.ID() != orig.ID() {
				t.Errorf("ID = %s, want %s", got.ID(), orig.ID())
			}
			if got.FeeLimit() != orig.FeeLimit() {
				t.Errorf("FeeLimit = %d, want %d", got.FeeLimit(), orig.FeeLimit())
			}
			if got.Expiration().UnixMilli() != orig.Expiration().UnixMilli() {
				t.Errorf("Expiration = %v, want %v", got.Expiration(), orig.Expiration())
			}
			// The concrete type must match the kind, so a signing loop can
			// type-assert safely.
			switch orig.Kind() {
			case KindNative:
				if _, ok := got.(*NativeTx); !ok {
					t.Errorf("decoded type %T, want *NativeTx", got)
				}
			case KindContract:
				if _, ok := got.(*ContractTx); !ok {
					t.Errorf("decoded type %T, want *ContractTx", got)
				}
			case KindDeploy:
				if _, ok := got.(*DeployTx); !ok {
					t.Errorf("decoded type %T, want *DeployTx", got)
				}
			case KindAssetTransfer:
				if _, ok := got.(*AssetTx); !ok {
					t.Errorf("decoded type %T, want *AssetTx", got)
				}
			}
		})
	}
}

// TestPortableCarriesPartialSignatures: the offline handoff case — encode an
// unsigned transaction, sign it on the "other side", and the signatures
// survive a second round trip. Signers() must recover them from the bytes
// (the decoded value has no in-memory signer list).
func TestPortableCarriesPartialSignatures(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	orig, err := BuildTransfer(t.Context(), cp, testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	unsigned, err := Encode(orig)
	if err != nil {
		t.Fatalf("Encode(unsigned): %v", err)
	}

	// Signer 1 imports, signs, exports.
	imported, err := Decode(unsigned)
	if err != nil {
		t.Fatalf("Decode(unsigned): %v", err)
	}
	s1 := mustSigner(t, testKeyHex)
	signed1, err := Sign(imported, s1)
	if err != nil {
		t.Fatalf("Sign(imported): %v", err)
	}
	blob1, err := Encode(signed1)
	if err != nil {
		t.Fatalf("Encode(signed1): %v", err)
	}

	// Signer 2 imports the partially signed envelope and adds its signature.
	imported2, err := Decode(blob1)
	if err != nil {
		t.Fatalf("Decode(signed1): %v", err)
	}
	if signers, err := imported2.Signers(); err != nil || len(signers) != 1 || signers[0] != s1.Address() {
		t.Fatalf("Signers after decode = %v, %v; want [%v]", signers, err, s1.Address())
	}
	s2 := mustSigner(t, testKeyHex2)
	signed2, err := Sign(imported2, s2)
	if err != nil {
		t.Fatalf("Sign(imported2): %v", err)
	}
	if signers, _ := signed2.Signers(); len(signers) != 2 || signers[0] != s1.Address() || signers[1] != s2.Address() {
		t.Fatalf("Signers = %v, want [s1 s2]", signers)
	}
	// Neither side changed raw_data: the txid is stable across all handoffs.
	if signed2.ID() != orig.ID() {
		t.Errorf("ID = %s, want %s (signatures must not change raw_data)", signed2.ID(), orig.ID())
	}
}

// TestSignRejectsDuplicateSignerAcrossHandoff: the same key signing twice
// invalidates a TRON multi-sig transaction. The duplicate must be caught
// either within one Sign call or when the second signature arrives through a
// decoded envelope.
func TestSignRejectsDuplicateSignerAcrossHandoff(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	orig, err := BuildTransfer(t.Context(), cp, testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	s := mustSigner(t, testKeyHex)

	// Same call, same signer twice.
	if _, err := orig.Sign(s, s); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
		t.Errorf("Sign(s, s) err = %v, want tx.already_signed", err)
	}

	// Across a handoff.
	signed, err := orig.Sign(s)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	blob, err := Encode(signed)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	back, err := Decode(blob)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if _, err := Sign(back, s); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
		t.Errorf("Sign(decoded, same signer) err = %v, want tx.already_signed", err)
	}
}

// TestDecodeRejectsCorruptEnvelopes pins every rejection path: bad magic,
// unknown version, declared kind contradicting the wrapped contract type,
// malformed body, and a signature that does not recover.
func TestDecodeRejectsCorruptEnvelopes(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	orig, err := BuildTransfer(t.Context(), cp, testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	good, err := Encode(orig)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	t.Run("short header", func(t *testing.T) {
		if _, err := Decode([]byte{'T', 'L'}); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
			t.Errorf("err = %v, want tx.invalid_argument", err)
		}
	})
	t.Run("bad magic", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[0] = 'X'
		if _, err := Decode(bad); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
			t.Errorf("err = %v, want tx.invalid_argument", err)
		}
	})
	t.Run("unknown version", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[4] = 99
		if _, err := Decode(bad); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
			t.Errorf("err = %v, want tx.invalid_argument", err)
		}
	})
	t.Run("declared kind contradicts contract", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[5] = byte(KindContract) // body still wraps TransferContract
		if _, err := Decode(bad); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
			t.Errorf("err = %v, want tx.invalid_argument", err)
		}
	})
	t.Run("unknown kind byte", func(t *testing.T) {
		bad := append([]byte(nil), good...)
		bad[5] = 0
		if _, err := Decode(bad); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
			t.Errorf("err = %v, want tx.invalid_argument", err)
		}
	})
	t.Run("malformed body", func(t *testing.T) {
		bad := append(append([]byte(nil), good[:portableHeaderLen]...), 0xff, 0xfe, 0xfd)
		if _, err := Decode(bad); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
			t.Errorf("err = %v, want tx.invalid_argument", err)
		}
	})
	t.Run("unrecoverable signature", func(t *testing.T) {
		// A truncated signature is not a valid [R || S || V] tuple, so
		// recovery fails. (A byte-flipped signature recovers to a *different*
		// address, which Decode cannot detect: the envelope records no
		// intended signer — that is what the node's sign-weight check is for.)
		s := mustSigner(t, testKeyHex)
		signed, err := orig.Sign(s)
		if err != nil {
			t.Fatalf("Sign: %v", err)
		}
		txp := signed.Transaction()
		txp.Signature = [][]byte{{0x01, 0x02, 0x03, 0x04}}
		body, err := proto.Marshal(txp)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		blob := append([]byte{'T', 'L', 'T', 'X', PortableVersion, byte(KindNative)}, body...)
		if _, err := Decode(blob); !tron.HasCode(err, tron.CodeKeyInvalid) {
			t.Errorf("err = %v, want key.invalid", err)
		}
	})
}

// TestEncodeRejectsUnknownContractType: a native transaction whose wrapped
// contract type the SDK does not model cannot be given a portable kind.
func TestEncodeRejectsUnknownContractType(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	native, err := BuildTransfer(t.Context(), cp, testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	native.Transaction().RawData.Contract[0].Type = core.Transaction_Contract_ShieldedTransferContract
	if _, err := Encode(native); !tron.HasCode(err, tron.CodeTxUnknownContract) {
		t.Errorf("Encode(shielded) err = %v, want tx.unknown_contract", err)
	}
}

// TestRemoteSigningFlow: SignHash + AttachSignature let a signer that only
// has bytes produce a signature the transaction accepts, and reject one that
// belongs to a different address.
func TestRemoteSigningFlow(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	native, err := BuildTransfer(t.Context(), cp, testFrom, testTo, 1)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	digest, err := SignHash(native)
	if err != nil {
		t.Fatalf("SignHash: %v", err)
	}
	if len(digest) != 32 {
		t.Fatalf("digest length = %d, want 32", len(digest))
	}
	s := mustSigner(t, testKeyHex)
	sig, err := s.Sign(digest)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	attached, err := native.WithSignature(s.Address(), sig)
	if err != nil {
		t.Fatalf("WithSignature: %v", err)
	}
	if signers, _ := attached.Signers(); len(signers) != 1 || signers[0] != s.Address() {
		t.Fatalf("Signers = %v, want [%v]", signers, s.Address())
	}
	// Attaching the same signature again is a duplicate.
	if _, err := attached.WithSignature(s.Address(), sig); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
		t.Errorf("duplicate attach err = %v, want tx.already_signed", err)
	}
	// A signature attached for the wrong address is refused.
	other := mustSigner(t, testKeyHex2)
	if _, err := native.WithSignature(other.Address(), sig); !tron.HasCode(err, tron.CodeKeyInvalid) {
		t.Errorf("wrong-address attach err = %v, want key.invalid", err)
	}
	// The interface-level helper preserves the concrete kind.
	iface, err := AttachSignature(native, s.Address(), sig)
	if err != nil {
		t.Fatalf("AttachSignature: %v", err)
	}
	if _, ok := iface.(*NativeTx); !ok {
		t.Errorf("AttachSignature returned %T, want *NativeTx", iface)
	}
}

// TestDecodedContractTxCannotSimulate: a decoded *ContractTx has no
// connection, so simulation must fail with a typed error instead of
// dereferencing a nil provider.
func TestDecodedContractTxCannotSimulate(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	contract, err := BuildTriggerSmartContract(t.Context(), cp, testFrom, testTo, []byte{0x01}, 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	blob, err := Encode(contract)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := Decode(blob)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	ct, ok := decoded.(*ContractTx)
	if !ok {
		t.Fatalf("decoded type %T, want *ContractTx", decoded)
	}
	if _, err := ct.Simulate(t.Context()); !tron.HasCode(err, tron.CodeChainConnection) {
		t.Errorf("Simulate on a decoded tx err = %v, want chain.connection", err)
	}
	if _, err := ct.EstimateEnergy(t.Context()); !tron.HasCode(err, tron.CodeChainConnection) {
		t.Errorf("EstimateEnergy on a decoded tx err = %v, want chain.connection", err)
	}
}

// TestWithOptionsRejectSignedTransactions: every option mutator must refuse a
// signed transaction, because the option lives in raw_data and would detach
// the signatures from the bytes they authorize.
func TestWithOptionsRejectSignedTransactions(t *testing.T) {
	kinds := buildAllKinds(t)
	s := mustSigner(t, testKeyHex)

	// Each closure signs its kind and asserts the applicable option mutators
	// reject the signed copy with tx.already_signed.
	cases := map[string]func(t *testing.T) error{
		"native": func(t *testing.T) error {
			signed, err := kinds["native"].(*NativeTx).Sign(s)
			if err != nil {
				return err
			}
			if _, err := signed.WithExpiration(5 * time.Minute); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				return err
			}
			_, err = signed.WithPermissionID(2)
			return err
		},
		"asset": func(t *testing.T) error {
			signed, err := kinds["asset"].(*AssetTx).Sign(s)
			if err != nil {
				return err
			}
			if _, err := signed.WithExpiration(5 * time.Minute); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				return err
			}
			_, err = signed.WithPermissionID(2)
			return err
		},
		"contract": func(t *testing.T) error {
			signed, err := kinds["contract"].(*ContractTx).Sign(s)
			if err != nil {
				return err
			}
			if _, err := signed.WithExpiration(5 * time.Minute); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				return err
			}
			if _, err := signed.WithFeeLimit(tron.TRX(5)); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				return err
			}
			_, err = signed.WithPermissionID(2)
			return err
		},
		"deploy": func(t *testing.T) error {
			signed, err := kinds["deploy"].(*DeployTx).Sign(s)
			if err != nil {
				return err
			}
			if _, err := signed.WithExpiration(5 * time.Minute); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				return err
			}
			if _, err := signed.WithFeeLimit(tron.TRX(5)); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				return err
			}
			if _, err := signed.WithPermissionID(2); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				return err
			}
			if _, err := signed.WithOriginEnergyLimit(1); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				return err
			}
			_, err = signed.WithResourcePercent(1)
			return err
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			if err := run(t); !tron.HasCode(err, tron.CodeTxAlreadySigned) {
				t.Errorf("%s: option mutation on a signed transaction err = %v, want tx.already_signed", name, err)
			}
		})
	}
}
