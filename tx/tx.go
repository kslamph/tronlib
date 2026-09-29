package tx

import (
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// Tx is the sealed interface satisfied by every transaction kind.
//
// Sealing is deliberate: without txInternal the interface would be
// satisfiable by any caller-supplied type, and Broadcast would accept a
// hand-rolled Tx that bypasses the builders which set fee_limit, expiration
// and permission id (architecture §6.1). The same technique seals contract.Arg.
type Tx interface {
	// txInternal seals the set: only the four kinds in this package can
	// implement Tx.
	txInternal()
	// ID returns the hex transaction id (sha256 of raw_data), or "" when
	// the node did not populate one.
	ID() string
	// Kind reports which TRON contract this transaction wraps.
	Kind() Kind
	// Extension returns the raw node build response (escape hatch).
	Extension() *api.TransactionExtention
	// Transaction returns the wrapped pb transaction (escape hatch).
	Transaction() *core.Transaction
	// Signers recovers the signer addresses from the attached signatures.
	Signers() ([]tron.Address, error)
	// IsSigned reports whether at least one signature is attached.
	IsSigned() bool
	// FeeLimit reports the effective fee limit in SUN (default applied).
	FeeLimit() tron.SUN
	// Expiration reports the effective expiration (default applied).
	Expiration() time.Time
	// PermissionID reports the effective permission id on the wrapped
	// contract message (0 = owner).
	PermissionID() int32
}
