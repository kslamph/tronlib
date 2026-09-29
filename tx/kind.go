package tx

// Kind reports which TRON contract a built transaction wraps. It is decided
// statically by the builder that constructed the transaction and is
// informational only — safety comes from the distinct Go types, not from
// this value (architecture §6.1). Open enum: a further kind is a new type, not a
// change here.
type Kind int

const (
	// KindNative is a non-contract transaction (TRX transfer and other
	// account operations); it consumes no energy.
	KindNative Kind = iota + 1
	// KindContract is a TriggerSmartContract transaction; Simulate and
	// EstimateEnergy apply (and exist only on *ContractTx).
	KindContract
	// KindDeploy is a CreateSmartContract transaction; no simulate path
	// exists for it in the protocol.
	KindDeploy
	// KindAssetTransfer is a TransferAssetContract transaction (TRC-10
	// transfer).
	KindAssetTransfer
)

// String returns the lower-case kind name, or "unknown" for values outside
// the enum.
func (k Kind) String() string {
	switch k {
	case KindNative:
		return "native"
	case KindContract:
		return "contract"
	case KindDeploy:
		return "deploy"
	case KindAssetTransfer:
		return "asset_transfer"
	default:
		return "unknown"
	}
}
