package tx

import (
	"time"

	"github.com/kslamph/tronlib/v2/key"
)

// AssetTx is a TransferAssetContract transaction (a TRC-10 transfer).
// Like NativeTx it consumes no energy, so no fee-limit option is
// offered.
type AssetTx struct{ baseTx }

// txInternal seals Tx: only the kinds in this package implement it.
func (*AssetTx) txInternal() {}

// clone returns a deep copy for the copy-on-write With*/Sign methods.
func (t *AssetTx) clone() *AssetTx { return &AssetTx{baseTx: *t.cloneBase()} }

// WithExpiration returns a COPY of t whose raw_data.expiration is moved to
// now+d (milliseconds). Call it before Sign (tx.already_signed otherwise).
func (t *AssetTx) WithExpiration(d time.Duration) (*AssetTx, error) {
	if err := ensureUnsigned(t.IsSigned(), "tx.AssetTx.WithExpiration"); err != nil {
		return nil, err
	}
	c := t.clone()
	setExpiration(c.raw(), d)
	return c, nil
}

// WithPermissionID returns a COPY of t with Permission_id set on the wrapped
// contract message. Call it before Sign (tx.already_signed otherwise).
func (t *AssetTx) WithPermissionID(id int32) (*AssetTx, error) {
	if err := ensureUnsigned(t.IsSigned(), "tx.AssetTx.WithPermissionID"); err != nil {
		return nil, err
	}
	c := t.clone()
	if err := setPermissionID(c.raw(), "tx.AssetTx.WithPermissionID", id); err != nil {
		return nil, err
	}
	return c, nil
}

// Sign returns a COPY of t with a signature from each signer appended to the
// pb transaction; signatures accumulate. The receiver is left untouched.
func (t *AssetTx) Sign(signers ...key.Signer) (*AssetTx, error) {
	b, err := signBase(&t.baseTx, signers)
	if err != nil {
		return nil, err
	}
	return &AssetTx{baseTx: *b}, nil
}
