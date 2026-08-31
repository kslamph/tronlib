package tx

import (
	"time"

	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/tron"
)

// ContractTx is a TriggerSmartContract transaction — contract calls and all
// TRC-20 operations. It is the only kind carrying Simulate and EstimateEnergy
// (the F1 fix, spec §6.2): the read-only paths take a TriggerSmartContract,
// which no other kind wraps.
type ContractTx struct{ baseTx }

// txInternal seals Tx: only the kinds in this package implement it.
func (*ContractTx) txInternal() {}

// clone returns a deep copy for the copy-on-write With*/Sign methods.
func (t *ContractTx) clone() *ContractTx { return &ContractTx{baseTx: *t.cloneBase()} }

// WithFeeLimit returns a COPY of t with raw_data.fee_limit set to s, the
// maximum SUN the node may burn for it. The builder already applied the
// documented default (150_000_000); this overrides it. It must be called
// before Sign (the signature covers raw_data).
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
func (t *ContractTx) WithFeeLimit(s tron.SUN) *ContractTx {
	c := t.clone()
	c.raw().FeeLimit = int64(s)
	return c
}

// WithExpiration returns a COPY of t whose raw_data.expiration is moved to
// now+d (milliseconds), the exact mutation of v1's utils.SetExpiration. Use
// it to circulate an unsigned transaction between signers for longer than the
// node's head+60s build default. Call it before Sign.
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
func (t *ContractTx) WithExpiration(d time.Duration) *ContractTx {
	c := t.clone()
	setExpiration(c.raw(), d)
	return c
}

// WithPermissionID returns a COPY of t with Permission_id set on the wrapped
// contract message (2–9 for multi-sig under active permissions). Call it
// before Sign.
// Note: setting options after signing invalidates any signature (raw_data changes; the node rejects with SIGERROR).
func (t *ContractTx) WithPermissionID(id int32) *ContractTx {
	c := t.clone()
	setPermissionID(c.raw(), id)
	return c
}

// Sign returns a COPY of t with a signature from each signer appended to the
// pb transaction (v1's signer.SignTx mutation logic: sha256 of raw_data,
// signer.Sign, append). Signatures accumulate, so multi-sig composes as
// tx = tx.Sign(a).Sign(b); the receiver is left untouched and unsigned.
func (t *ContractTx) Sign(signers ...key.Signer) (*ContractTx, error) {
	b, err := signBase(&t.baseTx, signers)
	if err != nil {
		return nil, err
	}
	return &ContractTx{baseTx: *b}, nil
}
