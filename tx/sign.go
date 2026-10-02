package tx

import (
	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// recoverSigners recovers the signer address of every attached signature over
// the digest of raw_data. Both Sign (to reject duplicates) and baseTx.Signers
// (so a decoded portable transaction reports the same signers) go through it,
// which keeps one rule in one place: a signature that does not recover is
// key.invalid, and its position is named.
func recoverSigners(raw *core.TransactionRaw, sigs [][]byte, op string) ([]tron.Address, error) {
	if len(sigs) == 0 {
		return nil, nil
	}
	digest, err := rawDigest(raw)
	if err != nil || digest == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Cause: err,
			Hint: "transaction carries signatures but no raw data to verify them against"}
	}
	out := make([]tron.Address, 0, len(sigs))
	for i, sig := range sigs {
		addr, err := key.RecoverAddress(digest, sig)
		if err != nil {
			return nil, &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Cause: err,
				Hint: "signature " + itoa(int64(i)) + " does not recover to an address; it is malformed or does not cover this raw_data"}
		}
		out = append(out, addr)
	}
	return out, nil
}

// signBase is the shared Sign mutation for all four kinds. It returns a
// cloned base carrying one appended signature per signer: sha256 of the
// proto-marshaled raw_data, Sign on that hash, append to the pb
// transaction's signature list. Signatures
// accumulate: multi-sig composes as tx = tx.Sign(a).Sign(b). The receiver is
// never modified.
//
// Duplicate signers are rejected: TRON validates multi-signature transactions
// strictly, and the same address signing twice (whether in one call or across
// a handoff after Decode) invalidates the whole transaction at broadcast. The
// check is against the recovered signatures already attached, so it also
// catches a signer that arrives twice through separate processes.
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
	digest, err := rawDigest(raw)
	if err != nil || digest == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: "tx.Sign", Cause: err}
	}
	tx := c.tx()
	existing, err := recoverSigners(raw, tx.GetSignature(), "tx.Sign")
	if err != nil {
		return nil, err
	}
	seen := make(map[tron.Address]struct{}, len(existing)+len(signers))
	for _, a := range existing {
		seen[a] = struct{}{}
	}
	for _, s := range signers {
		if s == nil {
			return nil, &tron.Error{
				Code: tron.CodeKeyInvalid,
				Op:   "tx.Sign",
				Hint: "a signer is nil",
			}
		}
		if _, dup := seen[s.Address()]; dup {
			return nil, &tron.Error{
				Code: tron.CodeTxAlreadySigned,
				Op:   "tx.Sign",
				Hint: "signer " + s.Address().String() + " already signed this transaction; a duplicate signature invalidates the whole transaction",
			}
		}
		sig, err := s.Sign(digest)
		if err != nil {
			return nil, &tron.Error{
				Code:  tron.CodeKeyInvalid,
				Op:    "tx.Sign",
				Hint:  "the signer refused to sign the transaction hash",
				Cause: err,
			}
		}
		// The recorded signer address must be the one the signature actually
		// belongs to: a custom signer (the interface is public) could return a
		// signature from key A while claiming address B. This is the load-bearing
		// check — without it Tx.Signers() would report an unverified address.
		recovered, rerr := key.RecoverAddress(digest, sig)
		if rerr != nil || recovered != s.Address() {
			return nil, &tron.Error{
				Code: tron.CodeKeyInvalid,
				Op:   "tx.Sign",
				Hint: "the signer returned a signature that does not recover to its Address(); the signer is inconsistent",
			}
		}
		tx.Signature = append(tx.Signature, sig)
		seen[s.Address()] = struct{}{}
	}
	return c, nil
}

// SignHash returns the 32-byte digest a signature must cover for t (sha256 of
// proto-marshaled raw_data). It is exported for signers that live outside this
// process — hardware wallets, remote signers, HSM bridges — which need the
// exact bytes to sign without importing the whole transaction type. The
// digest is a pure function of raw_data: sign it, then attach the raw 65-byte
// [R || S || V] result with WithSignature.
func SignHash(t Tx) ([]byte, error) {
	const op = "tx.SignHash"
	if t == nil {
		return nil, nilTx(op)
	}
	digest, err := rawDigest(t.Transaction().GetRawData())
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Cause: err}
	}
	if digest == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "transaction has no raw data; build it with the tx.Build* functions"}
	}
	return digest, nil
}

// withSignature returns a COPY of t carrying sig appended to its signature
// list, after verifying that sig recovers to addr. It is the counterpart of
// SignHash: a remote signer produces the 65-byte [R || S || V] signature over
// SignHash's digest, and the caller attaches it here without ever
// materializing a private key in this process. Attaching a signature that
// does not recover to the stated address is key.invalid, and a duplicate
// signer is tx.already_signed — the same rules the in-process Sign enforces.
func (b *baseTx) withSignature(op string, addr tron.Address, sig []byte) error {
	raw := b.raw()
	digest, err := rawDigest(raw)
	if err != nil || digest == nil {
		return &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Cause: err,
			Hint: "transaction has no raw data; build it with the tx.Build* functions"}
	}
	recovered, err := key.RecoverAddress(digest, sig)
	if err != nil {
		return &tron.Error{Code: tron.CodeKeyInvalid, Op: op, Cause: err,
			Hint: "the signature does not recover to an address; it is malformed or does not cover this raw_data"}
	}
	if recovered != addr {
		return &tron.Error{Code: tron.CodeKeyInvalid, Op: op,
			Hint: "the signature recovers to " + recovered.String() + ", not the address it was attached for (" + addr.String() + ")"}
	}
	existing, err := recoverSigners(raw, b.tx().GetSignature(), op)
	if err != nil {
		return err
	}
	for _, a := range existing {
		if a == addr {
			return &tron.Error{Code: tron.CodeTxAlreadySigned, Op: op,
				Hint: "signer " + addr.String() + " already signed this transaction; a duplicate signature invalidates the whole transaction"}
		}
	}
	b.tx().Signature = append(b.tx().Signature, sig)
	return nil
}

// WithSignature returns a COPY of t with the raw 65-byte [R || S || V]
// signature attached and attributed to addr. The signature must recover to
// addr over SignHash(t), which makes this the safe attachment point for a
// hardware wallet or remote signing service: the private key never enters
// this process, and a mismatched address fails here instead of at broadcast.
// Like Sign, the receiver is left untouched and signatures accumulate.
func (t *NativeTx) WithSignature(addr tron.Address, sig []byte) (*NativeTx, error) {
	c := t.clone()
	if err := c.withSignature("tx.NativeTx.WithSignature", addr, sig); err != nil {
		return nil, err
	}
	return c, nil
}

// WithSignature returns a COPY of t with the raw 65-byte [R || S || V]
// signature attached and attributed to addr (see NativeTx.WithSignature).
func (t *ContractTx) WithSignature(addr tron.Address, sig []byte) (*ContractTx, error) {
	c := t.clone()
	if err := c.withSignature("tx.ContractTx.WithSignature", addr, sig); err != nil {
		return nil, err
	}
	return c, nil
}

// WithSignature returns a COPY of t with the raw 65-byte [R || S || V]
// signature attached and attributed to addr (see NativeTx.WithSignature).
func (t *DeployTx) WithSignature(addr tron.Address, sig []byte) (*DeployTx, error) {
	c := t.clone()
	if err := c.withSignature("tx.DeployTx.WithSignature", addr, sig); err != nil {
		return nil, err
	}
	return c, nil
}

// WithSignature returns a COPY of t with the raw 65-byte [R || S || V]
// signature attached and attributed to addr (see NativeTx.WithSignature).
func (t *AssetTx) WithSignature(addr tron.Address, sig []byte) (*AssetTx, error) {
	c := t.clone()
	if err := c.withSignature("tx.AssetTx.WithSignature", addr, sig); err != nil {
		return nil, err
	}
	return c, nil
}

// WithSignature returns a COPY of t with the raw 65-byte [R || S || V]
// signature attached and attributed to addr, preserving the concrete kind.
// It is the interface-level counterpart of the per-kind WithSignature, for a
// transaction recovered by Decode whose static type is Tx.
func AttachSignature(t Tx, addr tron.Address, sig []byte) (Tx, error) {
	switch v := t.(type) {
	case *NativeTx:
		if v == nil {
			return nil, nilTx("tx.AttachSignature")
		}
		return v.WithSignature(addr, sig)
	case *ContractTx:
		if v == nil {
			return nil, nilTx("tx.AttachSignature")
		}
		return v.WithSignature(addr, sig)
	case *DeployTx:
		if v == nil {
			return nil, nilTx("tx.AttachSignature")
		}
		return v.WithSignature(addr, sig)
	case *AssetTx:
		if v == nil {
			return nil, nilTx("tx.AttachSignature")
		}
		return v.WithSignature(addr, sig)
	default:
		return nil, nilTx("tx.AttachSignature")
	}
}
