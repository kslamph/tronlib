package account

import (
	"context"
	"strconv"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// Permissions is the multi-signature handle for one account: reading the
// current permission configuration, submitting a complete replacement, and
// asking the node how much signature weight a partially signed transaction
// has collected.
//
// Configuration and signing are separate. This handle configures who may
// authorize and how much weight is needed; the transaction pipeline collects
// the signatures. Two consequences are worth knowing before using it:
//
//   - An update REPLACES the whole configuration. Permissions().Current gives
//     you the exact value to edit; submit the edited copy back.
//   - If the new owner permission cannot meet its own threshold, the account's
//     permissions can never be changed again. Rehearse on a testnet.
type Permissions struct {
	cp    rpc.ConnProvider
	owner tron.Address
}

// Owner returns the account this handle is bound to.
func (p *Permissions) Owner() tron.Address { return p.owner }

// Current reads the account's complete permission configuration (owner,
// witness when present, and every active permission) from GetAccount. It is
// the value to edit before calling Update.
func (p *Permissions) Current(ctx context.Context) (tx.PermissionSet, error) {
	const op = "account.Permissions.Current"
	acc, err := rpc.GetAccount(p.cp, ctx, &core.Account{Address: p.owner.Bytes()})
	if err != nil {
		return tx.PermissionSet{}, err
	}
	if acc.GetOwnerPermission() == nil && len(acc.GetActivePermission()) == 0 {
		return tx.PermissionSet{}, &tron.Error{
			Code: tron.CodeContractBadMetadata, Op: op,
			Hint: "the node reported no permission configuration for this account; the address may not exist yet",
		}
	}
	set := tx.PermissionSet{}
	if op := acc.GetOwnerPermission(); op != nil {
		set.Owner, err = permissionFromProto(op, "owner")
		if err != nil {
			return tx.PermissionSet{}, err
		}
	}
	if wp := acc.GetWitnessPermission(); wp != nil {
		w, err := permissionFromProto(wp, "witness")
		if err != nil {
			return tx.PermissionSet{}, err
		}
		set.Witness = &w
	}
	for i, ap := range acc.GetActivePermission() {
		a, err := permissionFromProto(ap, "actives["+itoa(int64(i))+"]")
		if err != nil {
			return tx.PermissionSet{}, err
		}
		set.Actives = append(set.Actives, a)
	}
	return set, nil
}

// Update builds an AccountPermissionUpdateContract for a complete set
// (tx.BuildAccountPermissionUpdate). The transaction must be signed under the
// account's current owner permission, or under an active permission whose
// operations bitmap enables tx.TypeAccountPermissionUpdate.
//
// The node charges a fixed permission-update fee (getUpdateAccountPermissionFee)
// — read it from tx.ChainParamsOf, or let TotalCost include it.
func (p *Permissions) Update(ctx context.Context, set tx.PermissionSet) (*tx.NativeTx, error) {
	return tx.BuildAccountPermissionUpdate(ctx, p.cp, p.owner, set)
}

// SignatureStatus is the node's verdict on a partially signed transaction
// (GetTransactionSignWeight): which permission it is being signed under, who
// has signed so far, how much weight that adds up to, and whether the
// threshold is met.
//
// Use it before broadcasting a multi-signature transaction. "At least one
// signature" is not authorization; Enough is.
type SignatureStatus struct {
	// Permission is the permission the transaction is signed under (from the
	// transaction's permission id).
	Permission *tx.Permission
	// Approved lists the addresses whose signatures the node recognized.
	Approved []tron.Address
	// Weight is the summed weight of Approved; Threshold is what it must
	// reach.
	Weight    int64
	Threshold int64
	// Enough reports whether the transaction can be broadcast under this
	// permission.
	Enough bool
	// Result is the node's raw result code name (ENOUGH_PERMISSION,
	// NOT_ENOUGH_PERMISSION, SIGNATURE_FORMAT_ERROR, …), preserved because
	// the distinction has different remedies.
	Result string
	// Message is the node's message, when any.
	Message string
}

// String renders the status as one line for logs.
func (s *SignatureStatus) String() string {
	return "signature weight " + itoa(s.Weight) + "/" + itoa(s.Threshold) + " (" + s.Result + ")"
}

// SignWeight asks the node to weigh the transaction's collected signatures
// (GetTransactionSignWeight). It performs no broadcast.
func (p *Permissions) SignWeight(ctx context.Context, t tx.Tx) (*SignatureStatus, error) {
	const op = "account.Permissions.SignWeight"
	if t == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	msg, err := rpc.GetTransactionSignWeight(p.cp, ctx, t.Transaction())
	if err != nil {
		return nil, err
	}
	approved, err := decodeApproved(msg.GetApprovedList(), op)
	if err != nil {
		return nil, err
	}
	out := &SignatureStatus{
		Approved:  approved,
		Weight:    msg.GetCurrentWeight(),
		Result:    msg.GetResult().GetCode().String(),
		Message:   msg.GetResult().GetMessage(),
		Threshold: msg.GetPermission().GetThreshold(),
	}
	if perm := msg.GetPermission(); perm != nil {
		decoded, err := permissionFromProto(perm, "permission")
		if err != nil {
			return nil, err
		}
		out.Permission = &decoded
	}
	out.Enough = msg.GetResult().GetCode() == api.TransactionSignWeight_Result_ENOUGH_PERMISSION &&
		(out.Threshold == 0 || out.Weight >= out.Threshold)
	return out, nil
}

// Approvals asks the node which addresses have signed the transaction
// (GetTransactionApprovedList). Unlike SignWeight it does not weigh them
// against a permission; it is the "who has signed so far" question.
func (p *Permissions) Approvals(ctx context.Context, t tx.Tx) ([]tron.Address, error) {
	const op = "account.Permissions.Approvals"
	if t == nil {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "transaction is nil"}
	}
	msg, err := rpc.GetTransactionApprovedList(p.cp, ctx, t.Transaction())
	if err != nil {
		return nil, err
	}
	if res := msg.GetResult(); res != nil && res.GetCode() != api.TransactionApprovedList_Result_SUCCESS {
		return nil, &tron.Error{
			Code: tron.CodeKeyInvalid, Op: op,
			Hint: "the node rejected the transaction's signatures: " + res.GetMessage() + " (" + res.GetCode().String() + ")",
		}
	}
	return decodeApproved(msg.GetApprovedList(), op)
}

// permissionFromProto decodes one permission, keeping the field name in the
// error so a corrupt key list names its slot.
func permissionFromProto(p *core.Permission, field string) (tx.Permission, error) {
	out := tx.Permission{
		ID:         p.GetId(),
		Name:       p.GetPermissionName(),
		Threshold:  p.GetThreshold(),
		Operations: p.GetOperations(),
	}
	for i, k := range p.GetKeys() {
		addr, err := tron.AddressFromBytes(k.GetAddress())
		if err != nil {
			return tx.Permission{}, &tron.Error{Code: tron.CodeAddressInvalid, Op: "account.Permissions.Current", Cause: err,
				Hint: field + ".keys[" + itoa(int64(i)) + "] is not a 0x41-prefixed 21-byte address"}
		}
		out.Keys = append(out.Keys, tx.PermissionKey{Address: addr, Weight: k.GetWeight()})
	}
	return out, nil
}

// decodeApproved converts the node's raw 21-byte approved addresses.
func decodeApproved(raw [][]byte, op string) ([]tron.Address, error) {
	out := make([]tron.Address, 0, len(raw))
	for _, b := range raw {
		a, err := tron.AddressFromBytes(b)
		if err != nil {
			return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Cause: err,
				Hint: "the node returned an approved address that is not a 0x41-prefixed 21-byte value"}
		}
		out = append(out, a)
	}
	return out, nil
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
