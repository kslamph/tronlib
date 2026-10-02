package tx

import (
	"bytes"

	"google.golang.org/protobuf/proto"

	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// Portable transactions (offline multi-signer handoff).
//
// A TRON multi-signature flow often has to move a transaction between
// machines: one process builds it, each signer signs on its own host, and
// only the last one broadcasts. Copy-on-write Sign makes that easy inside one
// process, but the protobuf wire types are not a safe interchange format:
// a *core.Transaction alone does not say which Go kind produced it, so an
// importer would have to guess — and guessing wrong re-creates the F1 defect
// (a TransferContract decoded as a TriggerSmartContract is wire-compatible on
// fields 1–3 and produces plausible numbers with no error).
//
// Encode therefore writes a self-describing envelope: magic, format version,
// the statically decided kind, then the marshalled transaction (including any
// signatures collected so far). Decode reconstructs the correct concrete type
// and refuses anything whose declared kind disagrees with the wrapped
// contract type, so the importer's type assertion is always sound.
//
// The format is deliberately small and additive:
//
//	offset 0..3   magic "TLTX"
//	offset 4      version byte (PortableVersion)
//	offset 5      kind byte (Kind, as declared by the builder)
//	offset 6..    proto.Marshal(*core.Transaction)
//
// Signatures already attached are carried verbatim; Decode additionally
// verifies that every signature is well formed and recovers to some address,
// so a corrupt handoff fails at import rather than at broadcast.

// PortableVersion is the envelope format version written by Encode and
// accepted by Decode.
const PortableVersion byte = 1

// portableHeaderLen is magic (4) + version (1) + kind (1).
const portableHeaderLen = 6

// portableMagic identifies a tronlib portable transaction envelope.
var portableMagic = [4]byte{'T', 'L', 'T', 'X'}

// Encode renders t as a portable, versioned envelope that carries the
// declared kind and every signature attached so far. The result is safe to
// persist or hand to another signer: Decode reconstructs the same concrete
// type and the same signatures.
//
// Encode rejects a transaction that is not a builder-produced shape (no raw
// data, or a contract list whose type does not match the transaction's kind),
// so an envelope that exists is always importable by the same version of this
// package.
func Encode(t Tx) ([]byte, error) {
	const op = "tx.Encode"
	if t == nil {
		return nil, nilTx(op)
	}
	c, err := singleContract(t.Transaction(), op)
	if err != nil {
		return nil, err
	}
	declared := t.Kind()
	want, ok := kindOfContract(c.GetType())
	if !ok {
		return nil, &tron.Error{Code: tron.CodeTxUnknownContract, Op: op,
			Hint: "the transaction wraps a contract type this SDK does not model; it has no portable kind"}
	}
	if want != declared {
		return nil, kindMismatch(op, declared, c.GetType())
	}
	kb, ok := kindByte(declared)
	if !ok {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "the transaction's kind is not encodable; it was not produced by this package's builders"}
	}
	if _, err := t.Signers(); err != nil {
		return nil, err
	}
	data, err := proto.Marshal(t.Transaction())
	if err != nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Cause: err,
			Hint: "the wrapped transaction could not be marshalled"}
	}
	out := make([]byte, 0, portableHeaderLen+len(data))
	out = append(out, portableMagic[:]...)
	out = append(out, PortableVersion, kb)
	return append(out, data...), nil
}

// Decode reconstructs a transaction from an Encode envelope. The returned
// value is the concrete kind the builder decided (NativeTx, ContractTx,
// DeployTx or AssetTx), carrying any signatures the envelope held; use Sign
// to add more. Every failure mode — wrong magic, unknown version, an
// envelope whose declared kind contradicts its contract type, malformed
// protobuf, or a signature that does not recover — is a typed *tron.Error,
// never a partially valid transaction.
func Decode(data []byte) (Tx, error) {
	const op = "tx.Decode"
	if len(data) < portableHeaderLen {
		return nil, corruptEnvelope(op, "the envelope is shorter than its 6-byte header")
	}
	if !bytes.Equal(data[:4], portableMagic[:]) {
		return nil, corruptEnvelope(op, "the envelope magic is not TLTX; this is not a tronlib transaction")
	}
	if data[4] != PortableVersion {
		return nil, corruptEnvelope(op,
			"envelope version "+itoa(int64(data[4]))+" is not supported by this build (want "+itoa(int64(PortableVersion))+")")
	}
	declared, ok := kindFromByte(data[5])
	if !ok {
		return nil, corruptEnvelope(op, "envelope kind byte is not a known transaction kind")
	}
	txpb := new(core.Transaction)
	if err := proto.Unmarshal(data[portableHeaderLen:], txpb); err != nil {
		return nil, corruptEnvelope(op, "the envelope body is not a valid transaction: "+err.Error())
	}
	base := &baseTx{ext: &api.TransactionExtention{Transaction: txpb}, kind: declared}
	c, err := singleContract(txpb, op)
	if err != nil {
		return nil, err
	}
	want, ok := kindOfContract(c.GetType())
	if !ok {
		return nil, &tron.Error{Code: tron.CodeTxUnknownContract, Op: op,
			Hint: "the envelope wraps a contract type this SDK does not model"}
	}
	if want != declared {
		return nil, kindMismatch(op, declared, c.GetType())
	}
	digest, err := rawDigest(base.raw())
	if err != nil || digest == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Cause: err,
			Hint: "the envelope's transaction carries no usable raw data"}
	}
	base.ext.Txid = digest
	if _, err := base.Signers(); err != nil {
		return nil, err
	}
	return leafOf(base), nil
}

// Sign adds a signature from each signer to t and returns the same concrete
// kind. It exists so a transaction recovered by Decode — whose static type is
// the Tx interface — can be signed without a type assertion. Signers already
// present are rejected (duplicate signatures invalidate the transaction), and
// the receiver is left untouched, exactly like the per-kind Sign method.
func Sign(t Tx, signers ...key.Signer) (Tx, error) {
	const op = "tx.Sign"
	switch v := t.(type) {
	case *NativeTx:
		if v == nil {
			return nil, nilTx(op)
		}
		return v.Sign(signers...)
	case *ContractTx:
		if v == nil {
			return nil, nilTx(op)
		}
		return v.Sign(signers...)
	case *DeployTx:
		if v == nil {
			return nil, nilTx(op)
		}
		return v.Sign(signers...)
	case *AssetTx:
		if v == nil {
			return nil, nilTx(op)
		}
		return v.Sign(signers...)
	default:
		return nil, nilTx(op)
	}
}

// leafOf returns the concrete named type wrapping base. Decode needs it
// because baseTx alone does not satisfy the sealed Tx interface.
func leafOf(b *baseTx) Tx {
	switch b.kind {
	case KindContract:
		return &ContractTx{baseTx: *b}
	case KindDeploy:
		return &DeployTx{baseTx: *b}
	case KindAssetTransfer:
		return &AssetTx{baseTx: *b}
	default:
		return &NativeTx{baseTx: *b}
	}
}

// singleContractRaw validates the shared portable preconditions and returns
// the one wrapped contract message.
func singleContract(txp *core.Transaction, op string) (*core.Transaction_Contract, error) {
	if txp == nil || txp.GetRawData() == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "transaction has no raw data; build it with the tx.Build* functions"}
	}
	raw := txp.GetRawData()
	if err := requireOneContract(raw, op); err != nil {
		return nil, err
	}
	return raw.GetContract()[0], nil
}

// kindOfContract maps a protocol contract type onto the Go kind that models
// it. The second result is false for a contract type this SDK cannot build or
// import.
func kindOfContract(ct core.Transaction_Contract_ContractType) (Kind, bool) {
	switch ct {
	case core.Transaction_Contract_TriggerSmartContract:
		return KindContract, true
	case core.Transaction_Contract_CreateSmartContract:
		return KindDeploy, true
	case core.Transaction_Contract_TransferAssetContract:
		return KindAssetTransfer, true
	case core.Transaction_Contract_TransferContract,
		core.Transaction_Contract_AccountCreateContract,
		core.Transaction_Contract_AccountUpdateContract,
		core.Transaction_Contract_SetAccountIdContract,
		core.Transaction_Contract_FreezeBalanceContract,
		core.Transaction_Contract_UnfreezeBalanceContract,
		core.Transaction_Contract_FreezeBalanceV2Contract,
		core.Transaction_Contract_UnfreezeBalanceV2Contract,
		core.Transaction_Contract_WithdrawBalanceContract,
		core.Transaction_Contract_WithdrawExpireUnfreezeContract,
		core.Transaction_Contract_CancelAllUnfreezeV2Contract,
		core.Transaction_Contract_DelegateResourceContract,
		core.Transaction_Contract_UnDelegateResourceContract,
		core.Transaction_Contract_VoteWitnessContract,
		core.Transaction_Contract_AccountPermissionUpdateContract,
		core.Transaction_Contract_UpdateSettingContract,
		core.Transaction_Contract_UpdateEnergyLimitContract,
		core.Transaction_Contract_ClearABIContract:
		return KindNative, true
	default:
		return 0, false
	}
}

// kindByte renders a Kind as its one-byte envelope code. The switch makes the
// conversion total: an out-of-range Kind cannot be silently truncated.
func kindByte(k Kind) (byte, bool) {
	switch k {
	case KindNative:
		return 1, true
	case KindContract:
		return 2, true
	case KindDeploy:
		return 3, true
	case KindAssetTransfer:
		return 4, true
	default:
		return 0, false
	}
}

func kindFromByte(b byte) (Kind, bool) {
	k := Kind(b)
	switch k {
	case KindNative, KindContract, KindDeploy, KindAssetTransfer:
		return k, true
	default:
		return 0, false
	}
}

func kindMismatch(op string, declared Kind, ct core.Transaction_Contract_ContractType) *tron.Error {
	return &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
		Hint: "the envelope declares kind " + declared.String() + " but wraps contract type " + ct.String() +
			"; the envelope is corrupt or was written by an incompatible tool"}
}

func corruptEnvelope(op, hint string) *tron.Error {
	return &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: hint}
}

func nilTx(op string) *tron.Error {
	return &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
}
