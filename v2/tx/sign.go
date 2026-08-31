package tx

import (
	"crypto/sha256"

	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/protobuf/proto"
)

// signBase is the shared Sign mutation for all four kinds. It returns a
// cloned base carrying one appended signature per signer — the exact port of
// v1's signer.SignTx (sha256 of proto-marshaled raw_data, signer.Sign on
// that hash, append to the pb transaction's signature list). Signatures
// accumulate: multi-sig composes as tx = tx.Sign(a).Sign(b). The receiver is
// never modified.
func signBase(b *baseTx, signers []key.Signer) (*baseTx, error) {
	if len(signers) == 0 {
		return nil, &tron.Error{
			Code: tron.CodeTxNoSigner,
			Op:   "tx.Sign",
			Hint: "Sign requires at least one signer; broadcast an unsigned transaction fails with the same code",
		}
	}
	c := b.cloneBase()
	raw := c.raw()
	if raw == nil {
		return nil, &tron.Error{
			Code: tron.CodeTxInvalidArgument,
			Op:   "tx.Sign",
			Hint: "transaction has no raw data; the builders always set it",
		}
	}
	data, err := proto.Marshal(raw)
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: "tx.Sign", Cause: err}
	}
	sum := sha256.Sum256(data)
	tx := c.tx()
	for _, s := range signers {
		if s == nil {
			return nil, &tron.Error{
				Code: tron.CodeKeyInvalid,
				Op:   "tx.Sign",
				Hint: "a signer is nil",
			}
		}
		sig, err := s.Sign(sum[:])
		if err != nil {
			return nil, &tron.Error{
				Code:  tron.CodeKeyInvalid,
				Op:    "tx.Sign",
				Hint:  "the signer refused to sign the transaction hash",
				Cause: err,
			}
		}
		tx.Signature = append(tx.Signature, sig)
		c.signers = append(c.signers, s.Address())
	}
	return c, nil
}
