package tx

import (
	"time"

	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/tron"
)

// ContractTx is a TriggerSmartContract transaction — contract calls and all
// TRC-20 operations. It is the only kind carrying Simulate and EstimateEnergy
// (the read-only paths take a TriggerSmartContract,
// which no other kind wraps.
type ContractTx struct{ baseTx }

// txInternal seals Tx: only the kinds in this package implement it.
func (*ContractTx) txInternal() {}

// clone returns a deep copy for the copy-on-write With*/Sign methods.
func (t *ContractTx) clone() *ContractTx { return &ContractTx{baseTx: *t.cloneBase()} }

// WithFeeLimit returns a COPY of t with raw_data.fee_limit set to s, the
// maximum SUN the node may burn for it. The builder already applied the
// documented default (150_000_000); this overrides it. It must be called
// before Sign — options are part of raw_data and a post-sign mutation would
// detach the signatures from the bytes they authorize, so signing first is
// tx.already_signed here rather than a SIGERROR at broadcast.
func (t *ContractTx) WithFeeLimit(s tron.SUN) (*ContractTx, error) {
	if err := ensureUnsigned(t.IsSigned(), "tx.ContractTx.WithFeeLimit"); err != nil {
		return nil, err
	}
	c := t.clone()
	c.raw().FeeLimit = int64(s)
	return c, nil
}

// WithExpiration returns a COPY of t whose raw_data.expiration is moved to
// now+d (milliseconds). Use
// it to circulate an unsigned transaction between signers for longer than the
// node's head+60s build default. Call it before Sign (tx.already_signed
// otherwise).
func (t *ContractTx) WithExpiration(d time.Duration) (*ContractTx, error) {
	if err := ensureUnsigned(t.IsSigned(), "tx.ContractTx.WithExpiration"); err != nil {
		return nil, err
	}
	c := t.clone()
	setExpiration(c.raw(), d)
	return c, nil
}

// WithPermissionID returns a COPY of t with Permission_id set on the wrapped
// contract message (2–9 for multi-sig under active permissions). Call it
// before Sign. It returns an error when the transaction is already signed, or
// when the wrapped contract message is missing (reachable only by replacing
// the node's build response through the Extension() escape hatch).
func (t *ContractTx) WithPermissionID(id int32) (*ContractTx, error) {
	if err := ensureUnsigned(t.IsSigned(), "tx.ContractTx.WithPermissionID"); err != nil {
		return nil, err
	}
	c := t.clone()
	if err := setPermissionID(c.raw(), "tx.ContractTx.WithPermissionID", id); err != nil {
		return nil, err
	}
	return c, nil
}

// Sign returns a COPY of t with a signature from each signer appended to the
// pb transaction (sha256 of raw_data, Sign, append). Signatures
// accumulate, so multi-sig composes as
// tx = tx.Sign(a).Sign(b); the receiver is left untouched and unsigned.
func (t *ContractTx) Sign(signers ...key.Signer) (*ContractTx, error) {
	b, err := signBase(&t.baseTx, signers)
	if err != nil {
		return nil, err
	}
	return &ContractTx{baseTx: *b}, nil
}
