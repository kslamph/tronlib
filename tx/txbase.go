package tx

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// baseTx is the shared state of the four transaction kinds. Each exported
// kind embeds it; the builders are the only constructors, so every method
// below can assume ext carries a transaction with raw data (the builders
// enforce that with requireRaw).
type baseTx struct {
	ext  *api.TransactionExtention
	kind Kind
	cp   rpc.ConnProvider // needed by the contract-only Simulate/EstimateEnergy
}

// tx returns the wrapped pb transaction.
func (b *baseTx) tx() *core.Transaction { return b.ext.GetTransaction() }

// raw returns the wrapped raw_data.
func (b *baseTx) raw() *core.TransactionRaw { return b.tx().GetRawData() }

// rawDigest returns sha256(proto.Marshal(raw)) — the bytes a TRON signature
// covers and the preimage of the transaction id. nil raw data yields (nil,
// nil): callers report the missing-raw-data error themselves.
func rawDigest(raw *core.TransactionRaw) ([]byte, error) {
	if raw == nil {
		return nil, nil
	}
	data, err := proto.Marshal(raw)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	return sum[:], nil
}

// ID returns the hex transaction id. It is computed live from raw_data (a
// TRON txid is a pure function of raw_data) so it stays correct after a
// copy-on-write option mutated the raw data post-build — unlike ext.Txid,
// which is a stale build-time snapshot in that case.
func (b *baseTx) ID() string {
	digest, err := rawDigest(b.raw())
	if err != nil || digest == nil {
		return ""
	}
	return hex.EncodeToString(digest)
}

// Kind reports the statically-decided kind.
func (b *baseTx) Kind() Kind { return b.kind }

// Extension returns the raw node build response (escape hatch).
func (b *baseTx) Extension() *api.TransactionExtention { return b.ext }

// Transaction returns the wrapped pb transaction (escape hatch).
func (b *baseTx) Transaction() *core.Transaction { return b.tx() }

// IsSigned reports whether at least one signature is attached.
func (b *baseTx) IsSigned() bool { return len(b.tx().GetSignature()) > 0 }

// Signers recovers the signer addresses from the attached signatures, in
// signature order. Deriving them from the bytes (rather than recording them
// at Sign time) is what makes a decoded portable transaction report the same
// signers as the one that was encoded, and it re-verifies every signature it
// reports: a signature that does not recover is key.invalid, never a silently
// skipped entry.
func (b *baseTx) Signers() ([]tron.Address, error) {
	return recoverSigners(b.raw(), b.tx().GetSignature(), "tx.Signers")
}

// FeeLimit reports the effective fee limit in SUN. The builders apply the
// documented default (150_000_000) at build time, so the value read here is
// always what the node will see.
func (b *baseTx) FeeLimit() tron.SUN { return tron.SUN(b.raw().GetFeeLimit()) }

// Expiration reports the effective expiration. The build RPC sets it
// server-side (head + 60 s); WithExpiration mutates raw_data.expiration
// post-build, and this accessor always reports the current value.
func (b *baseTx) Expiration() time.Time {
	ms := b.raw().GetExpiration()
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// PermissionID reports the permission id on the wrapped contract message
// (0 = owner; multi-sig under active permissions needs 2–9).
func (b *baseTx) PermissionID() int32 {
	c := b.raw().GetContract()
	if len(c) == 0 {
		return 0
	}
	return c[0].GetPermissionId()
}

// cloneBase deep-copies the shared state for the copy-on-write With*/Sign
// methods. The pb extention is proto.Clone'd so a mutation of the copy can
// never touch the receiver's transaction (the receiver's ID and signature
// stay valid).
func (b *baseTx) cloneBase() *baseTx {
	nb := &baseTx{
		kind: b.kind,
		cp:   b.cp,
	}
	if b.ext != nil {
		nb.ext = proto.Clone(b.ext).(*api.TransactionExtention)
	}
	return nb
}

// requireRaw errors when a node build response carries no transaction raw
// data — the precondition every accessor and copy-on-write method relies on.
// Builders call it before constructing a kind.
func requireRaw(ext *api.TransactionExtention, op string) error {
	if ext.GetTransaction() == nil || ext.GetTransaction().GetRawData() == nil {
		return &tron.Error{
			Code: tron.CodeRPCMethodFailed,
			Op:   op,
			Hint: "the node's build response carries no transaction raw data; check node version and connectivity",
		}
	}
	return nil
}

// ensureUnsigned rejects an option mutation on an already-signed transaction.
// Every option (fee limit, expiration, permission id, resource percent) lives
// inside raw_data, and every attached signature covers raw_data's hash, so a
// post-sign mutation silently detaches the signatures from the bytes they
// authorize; the node then rejects the broadcast with SIGERROR. Failing here
// names the cause while the caller can still fix it, and it makes the rule
// uniform: options are set before Sign, never after.
func ensureUnsigned(signed bool, op string) error {
	if signed {
		return &tron.Error{
			Code: tron.CodeTxAlreadySigned,
			Op:   op,
			Hint: "options are part of raw_data and change the signed hash; call With* before Sign, or rebuild the transaction",
		}
	}
	return nil
}

// noConnection reports an operation that needs the node but was called on a
// transaction that carries no connection — a value reconstructed by Decode,
// whose baseTx has no rpc.ConnProvider. Simulation and estimation are the
// only such operations; they fail loudly here instead of dereferencing a nil
// provider.
func noConnection(op string) *tron.Error {
	return &tron.Error{
		Code: tron.CodeChainConnection,
		Op:   op,
		Hint: "this transaction carries no connection; simulation and estimation need a transaction returned by the builders — rebuild it against a client",
	}
}

// requireOneContract errors unless the transaction wraps exactly one contract
// message — the shape Broadcast and every parameter reader rely on.
func requireOneContract(raw *core.TransactionRaw, op string) error {
	if n := len(raw.GetContract()); n != 1 {
		return &tron.Error{
			Code: tron.CodeTxInvalidArgument,
			Op:   op,
			Hint: "transaction must contain exactly one contract message",
		}
	}
	return nil
}
