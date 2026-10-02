package tx

import (
	"time"

	"github.com/kslamph/tronlib/v2/key"
)

// NativeTx is a non-contract transaction (a TRX transfer built by
// BuildTransfer; further native operations are added as new builders, not as
// new kinds). It consumes no energy, so no fee-limit option is offered —
// WithFeeLimit is contract-shaped only, because the ceiling exists to bound
// the energy purchase, which is the only cost that can approach it.
// BuildTransfer still stamps the DefaultFeeLimit default like the other three
// core builders; on a native transfer the cap is never the binding
// constraint (the largest native outlay this design documents is ~1.1 TRX of
// recipient activation, two orders of magnitude below the default), but it is
// not inert: an unbandwidth-covered transfer still burns TRX.
type NativeTx struct{ baseTx }

// txInternal seals Tx: only the kinds in this package implement it.
func (*NativeTx) txInternal() {}

// clone returns a deep copy for the copy-on-write With*/Sign methods.
func (t *NativeTx) clone() *NativeTx { return &NativeTx{baseTx: *t.cloneBase()} }

// WithExpiration returns a COPY of t whose raw_data.expiration is moved to
// now+d (milliseconds). Use
// it to circulate an unsigned transaction between signers for longer than the
// node's head+60s build default. It must be called BEFORE Sign: the signature
// covers raw_data, so signing first is tx.already_signed here rather than a
// SIGERROR at broadcast.
func (t *NativeTx) WithExpiration(d time.Duration) (*NativeTx, error) {
	if err := ensureUnsigned(t.IsSigned(), "tx.NativeTx.WithExpiration"); err != nil {
		return nil, err
	}
	c := t.clone()
	setExpiration(c.raw(), d)
	return c, nil
}

// WithPermissionID returns a COPY of t with Permission_id set on the wrapped
// contract message (2–9 for multi-sig under active permissions). Call it
// before Sign (tx.already_signed otherwise).
func (t *NativeTx) WithPermissionID(id int32) (*NativeTx, error) {
	if err := ensureUnsigned(t.IsSigned(), "tx.NativeTx.WithPermissionID"); err != nil {
		return nil, err
	}
	c := t.clone()
	if err := setPermissionID(c.raw(), "tx.NativeTx.WithPermissionID", id); err != nil {
		return nil, err
	}
	return c, nil
}

// Sign returns a COPY of t with a signature from each signer appended to the
// pb transaction (sha256 of raw_data, Sign, append). Signatures
// accumulate, so multi-sig composes as
// tx = tx.Sign(a).Sign(b); the receiver is left untouched and unsigned.
func (t *NativeTx) Sign(signers ...key.Signer) (*NativeTx, error) {
	b, err := signBase(&t.baseTx, signers)
	if err != nil {
		return nil, err
	}
	return &NativeTx{baseTx: *b}, nil
}
