package tx

// Every With* option is copy-on-write: it returns a new transaction and
// leaves the receiver exactly as it was. That contract is what lets a caller
// hold a draft — build options into it, hand it to a second signer, keep
// reading its ID — without the draft shifting underneath them.

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// optionCase is one mutator applied to a built transaction: a name, the
// function that produces the copy, and how to read the option's value back
// out of an arbitrary transaction.
//
// The enumeration below is deliberately exhaustive rather than
// representative: a With* option that is not listed here is silently
// untested, and "the clone deep-copies" is a property of the whole family.
type optionCase struct {
	name  string
	kind  string
	apply func(Tx) (Tx, error)
	// read returns the value the option was supposed to write, so the test
	// can tell "a copy came back" from "the copy is correct".
	read func(Tx) any
}

// deployParamOf pulls the CreateSmartContract parameter out of a DeployTx so
// the deploy-only options have something to assert on. The values those
// options write live on the nested NewContract message, inside the packed
// parameter — one level deeper than the copy has to reach.
func deployParamOf(t *testing.T, tx Tx) *core.CreateSmartContract {
	t.Helper()
	contracts := tx.Transaction().GetRawData().GetContract()
	if len(contracts) != 1 {
		t.Fatalf("DeployTx carries %d contracts, want 1", len(contracts))
	}
	param := new(core.CreateSmartContract)
	if err := proto.Unmarshal(contracts[0].GetParameter().GetValue(), param); err != nil {
		t.Fatalf("deploy contract parameter does not decode: %v", err)
	}
	return param
}

func allOptionCases(t *testing.T) []optionCase {
	t.Helper()
	return []optionCase{
		{"native.WithExpiration", "native", func(tx Tx) (Tx, error) { return tx.(*NativeTx).WithExpiration(90 * time.Minute) },
			func(tx Tx) any { return tx.Expiration() }},
		{"native.WithPermissionID", "native", func(tx Tx) (Tx, error) { return tx.(*NativeTx).WithPermissionID(7) },
			func(tx Tx) any { return tx.PermissionID() }},
		{"contract.WithFeeLimit", "contract", func(tx Tx) (Tx, error) { return tx.(*ContractTx).WithFeeLimit(tron.TRX(42)) },
			func(tx Tx) any { return tx.FeeLimit() }},
		{"contract.WithExpiration", "contract", func(tx Tx) (Tx, error) { return tx.(*ContractTx).WithExpiration(90 * time.Minute) },
			func(tx Tx) any { return tx.Expiration() }},
		{"contract.WithPermissionID", "contract", func(tx Tx) (Tx, error) { return tx.(*ContractTx).WithPermissionID(7) },
			func(tx Tx) any { return tx.PermissionID() }},
		{"deploy.WithFeeLimit", "deploy", func(tx Tx) (Tx, error) { return tx.(*DeployTx).WithFeeLimit(tron.TRX(42)) },
			func(tx Tx) any { return tx.FeeLimit() }},
		{"deploy.WithExpiration", "deploy", func(tx Tx) (Tx, error) { return tx.(*DeployTx).WithExpiration(90 * time.Minute) },
			func(tx Tx) any { return tx.Expiration() }},
		{"deploy.WithPermissionID", "deploy", func(tx Tx) (Tx, error) { return tx.(*DeployTx).WithPermissionID(7) },
			func(tx Tx) any { return tx.PermissionID() }},
		{"deploy.WithOriginEnergyLimit", "deploy", func(tx Tx) (Tx, error) { return tx.(*DeployTx).WithOriginEnergyLimit(1_000_000) },
			func(tx Tx) any { return deployParamOf(t, tx).GetNewContract().GetOriginEnergyLimit() }},
		{"deploy.WithResourcePercent", "deploy", func(tx Tx) (Tx, error) { return tx.(*DeployTx).WithResourcePercent(100) },
			func(tx Tx) any { return deployParamOf(t, tx).GetNewContract().GetConsumeUserResourcePercent() }},
		{"asset.WithExpiration", "asset", func(tx Tx) (Tx, error) { return tx.(*AssetTx).WithExpiration(90 * time.Minute) },
			func(tx Tx) any { return tx.Expiration() }},
		{"asset.WithPermissionID", "asset", func(tx Tx) (Tx, error) { return tx.(*AssetTx).WithPermissionID(7) },
			func(tx Tx) any { return tx.PermissionID() }},
	}
}

// TestOptionsDoNotMutateTheReceiver: applying an option must leave the
// receiver's raw_data byte-identical and its ID unchanged.
//
// Failure mode: a clone that shares the *core.Transaction (or its raw_data)
// instead of deep-copying it. The option then rewrites the draft the caller
// still holds — changing the expiration of a transaction a counterparty has
// already signed, or the permission id that decides whose key must sign it.
// The two copies then disagree about the ID, and a multisig handoff signs
// bytes the other party never saw.
func TestOptionsDoNotMutateTheReceiver(t *testing.T) {
	kinds := buildAllKinds(t)
	for _, opt := range allOptionCases(t) {
		receiver := kinds[opt.kind]
		t.Run(opt.name, func(t *testing.T) {
			before, err := proto.Marshal(receiver.Transaction())
			if err != nil {
				t.Fatalf("marshal receiver: %v", err)
			}
			beforeID := receiver.ID()

			cp, err := opt.apply(receiver)
			if err != nil {
				t.Fatalf("apply: %v", err)
			}

			after, err := proto.Marshal(receiver.Transaction())
			if err != nil {
				t.Fatalf("re-marshal receiver: %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("the receiver's transaction changed:\n before: %x\n after:  %x", before, after)
			}
			if receiver.ID() != beforeID {
				t.Errorf("the receiver's ID changed: %s -> %s", beforeID, receiver.ID())
			}
			if cp.ID() == receiver.ID() {
				t.Errorf("the copy carries the receiver's ID %s; the option wrote nothing into raw_data", cp.ID())
			}
			if cp.Kind() != receiver.Kind() {
				t.Errorf("the copy's kind = %v, want %v (an option must preserve the concrete kind)", cp.Kind(), receiver.Kind())
			}
			// Sanity: the option really did write its value, so this is a
			// real mutation and not a no-op copy.
			if got, before := opt.read(cp), opt.read(receiver); got == before {
				t.Errorf("the copy reads %v, the same as the receiver; the option did not apply", got)
			}
		})
	}
}

// TestOptionsReturnIndependentCopies: after an option, the two transactions
// must not share state — signing the copy must not sign the receiver.
//
// Failure mode: the shallow-clone case again, observed through signing. A
// shared *core.Transaction puts the receiver's and the copy's signature
// lists on one object, so a later Sign() retroactively signs a draft the
// caller still believes is unsigned.
func TestOptionsReturnIndependentCopies(t *testing.T) {
	kinds := buildAllKinds(t)
	s := mustSigner(t, testKeyHex)
	for _, opt := range allOptionCases(t) {
		receiver := kinds[opt.kind]
		t.Run(opt.name, func(t *testing.T) {
			cp, err := opt.apply(receiver)
			if err != nil {
				t.Fatalf("apply: %v", err)
			}
			if _, err := Sign(cp, s); err != nil {
				t.Fatalf("Sign(copy): %v", err)
			}
			if receiver.IsSigned() {
				t.Error("signing the copy signed the receiver too; the two share state")
			}
		})
	}
}

// TestOptionsChangeTheSignedBytes: the copy's digest must differ from the
// receiver's, and a signature made over the receiver's digest must not be
// accepted by the copy. This is the wire-level statement of "options come
// before signing" — a signature authorizes one exact raw_data and nothing else.
func TestOptionsChangeTheSignedBytes(t *testing.T) {
	kinds := buildAllKinds(t)
	s := mustSigner(t, testKeyHex)
	for _, opt := range allOptionCases(t) {
		receiver := kinds[opt.kind]
		t.Run(opt.name, func(t *testing.T) {
			cp, err := opt.apply(receiver)
			if err != nil {
				t.Fatalf("apply: %v", err)
			}
			before, err := SignHash(receiver)
			if err != nil {
				t.Fatalf("SignHash(receiver): %v", err)
			}
			after, err := SignHash(cp)
			if err != nil {
				t.Fatalf("SignHash(copy): %v", err)
			}
			if bytes.Equal(before, after) {
				t.Fatal("the option did not change the digest a signature covers")
			}
			sig, err := s.Sign(before)
			if err != nil {
				t.Fatalf("sign: %v", err)
			}
			if _, err := AttachSignature(cp, s.Address(), sig); !tron.HasCode(err, tron.CodeKeyInvalid) {
				t.Errorf("a signature over the pre-option digest was accepted by the copy: err = %v, want key.invalid", err)
			}
		})
	}
}

// TestOptionTableCoversEveryWithOption: the tables above are only as good as
// their enumeration. This walks the method set of the four concrete kinds by
// reflection and fails if any exported With* method has no case in
// allOptionCases, so a newly added option cannot ship untested.
//
// The inventory is derived from the code rather than hand-listed: a second
// hand-kept list would only test that someone remembered to update it.
func TestOptionTableCoversEveryWithOption(t *testing.T) {
	covered := map[string]bool{}
	for _, opt := range allOptionCases(t) {
		covered[opt.name] = true
	}
	for kind, sample := range map[string]Tx{
		"native":   &NativeTx{},
		"contract": &ContractTx{},
		"deploy":   &DeployTx{},
		"asset":    &AssetTx{},
	} {
		typ := reflect.TypeOf(sample)
		for i := range typ.NumMethod() {
			method := typ.Method(i)
			if !strings.HasPrefix(method.Name, "With") {
				continue
			}
			if method.Name == "WithSignature" {
				// The signature family takes two arguments and is pinned by
				// offline_signing_test.go, not by the option tables here.
				continue
			}
			if method.Type.NumIn() != 2 { // receiver + one argument
				t.Errorf("%s.%s takes %d arguments; the option tables assume a single-argument With* mutator",
					kind, method.Name, method.Type.NumIn()-1)
				continue
			}
			if !covered[kind+"."+method.Name] {
				t.Errorf("%s.%s has no case in allOptionCases; add one", kind, method.Name)
			}
		}
	}
}

// TestOptionNamesAreWellFormed: each case is "<kind>.<WithMethod>" and names a
// kind the reflection walk above knows about — a typo in a case name would
// otherwise make a test silently cover a different method than it claims.
func TestOptionNamesAreWellFormed(t *testing.T) {
	known := map[string]bool{"native": true, "contract": true, "deploy": true, "asset": true}
	for _, opt := range allOptionCases(t) {
		kind, method, ok := strings.Cut(opt.name, ".")
		if !ok || !known[kind] || !strings.HasPrefix(method, "With") {
			t.Errorf("option case %q is not <kind>.With<Method> for a known kind", opt.name)
		}
	}
}
