package tx

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// Account permission management (TIP-16 multi-signature).
//
// A TRON account has three permission slots — owner, witness, and up to eight
// active permissions — each holding up to five weighted keys and a threshold.
// A signature contributes its key's weight; the permission is satisfied when
// the weights of the recovered signers reach the threshold. That is the whole
// model, and it is expressed here as data (PermissionSet) rather than as
// protobufs, because a permission update REPLACES the complete configuration:
// the node never merges, so a partially-filled update silently wipes the
// slots it omitted. The safe pattern is read (Permissions().Current),
// modify the slot you mean, submit the whole set.
//
// Signing is separate from permission configuration: a transaction built with
// BuildAccountPermissionUpdate is signed like any other, with an active
// permission selected via WithPermissionID.

// ContractType identifies a protocol contract type for an active
// permission's operations bitmap. The values are the protocol's
// Transaction.Contract.ContractType numbers; the named constants cover the
// contract types this SDK can build, and OperationsBitmap accepts any value in
// the protocol's range.
type ContractType int32

// Contract types this SDK can construct (and therefore the operations an
// active permission most commonly needs to enable). The numbers are the
// protocol's enum values and must not be renumbered.
const (
	TypeAccountCreate           ContractType = 0
	TypeTransfer                ContractType = 1
	TypeTransferAsset           ContractType = 2
	TypeVoteWitness             ContractType = 4
	TypeCreateWitness           ContractType = 5
	TypeUpdateWitness           ContractType = 8
	TypeFreezeBalance           ContractType = 11
	TypeUnfreezeBalance         ContractType = 12
	TypeWithdrawBalance         ContractType = 13
	TypeProposalCreate          ContractType = 16
	TypeProposalApprove         ContractType = 17
	TypeProposalDelete          ContractType = 18
	TypeCreateSmartContract     ContractType = 30
	TypeTriggerSmartContract    ContractType = 31
	TypeUpdateSetting           ContractType = 33
	TypeUpdateEnergyLimit       ContractType = 45
	TypeAccountPermissionUpdate ContractType = 46
	TypeClearABI                ContractType = 48
	TypeUpdateBrokerage         ContractType = 49
	TypeFreezeBalanceV2         ContractType = 54
	TypeUnfreezeBalanceV2       ContractType = 55
	TypeWithdrawExpireUnfreeze  ContractType = 56
	TypeDelegateResource        ContractType = 57
	TypeUnDelegateResource      ContractType = 58
	TypeCancelAllUnfreezeV2     ContractType = 59
)

// String names the contract type where this package has a name for it, and
// renders the number otherwise.
func (t ContractType) String() string {
	if name, ok := contractTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("ContractType(%d)", int32(t))
}

var contractTypeNames = map[ContractType]string{
	TypeAccountCreate:           "AccountCreateContract",
	TypeTransfer:                "TransferContract",
	TypeTransferAsset:           "TransferAssetContract",
	TypeVoteWitness:             "VoteWitnessContract",
	TypeCreateWitness:           "WitnessCreateContract",
	TypeUpdateWitness:           "WitnessUpdateContract",
	TypeFreezeBalance:           "FreezeBalanceContract",
	TypeUnfreezeBalance:         "UnfreezeBalanceContract",
	TypeWithdrawBalance:         "WithdrawBalanceContract",
	TypeProposalCreate:          "ProposalCreateContract",
	TypeProposalApprove:         "ProposalApproveContract",
	TypeProposalDelete:          "ProposalDeleteContract",
	TypeCreateSmartContract:     "CreateSmartContract",
	TypeTriggerSmartContract:    "TriggerSmartContract",
	TypeUpdateSetting:           "UpdateSettingContract",
	TypeUpdateEnergyLimit:       "UpdateEnergyLimitContract",
	TypeAccountPermissionUpdate: "AccountPermissionUpdateContract",
	TypeClearABI:                "ClearABIContract",
	TypeUpdateBrokerage:         "UpdateBrokerageContract",
	TypeFreezeBalanceV2:         "FreezeBalanceV2Contract",
	TypeUnfreezeBalanceV2:       "UnfreezeBalanceV2Contract",
	TypeWithdrawExpireUnfreeze:  "WithdrawExpireUnfreezeContract",
	TypeDelegateResource:        "DelegateResourceContract",
	TypeUnDelegateResource:      "UnDelegateResourceContract",
	TypeCancelAllUnfreezeV2:     "CancelAllUnfreezeV2Contract",
}

// operationsLen is the protocol's fixed bitmap width: 256 bits, one per
// contract type id.
const operationsLen = 32

// OperationsBitmap builds the 32-byte little-endian operations bitmap for an
// active permission: contract type id n is bit (n % 8) of byte (n / 8). Pass
// the contract types the permission may execute; everything else is denied.
//
// Build it rather than hand-rolling hex: a wrong byte order silently grants or
// denies the wrong operations, and the node reports only "Permission denied!"
// at broadcast.
func OperationsBitmap(types ...ContractType) ([]byte, error) {
	const op = "tx.OperationsBitmap"
	out := make([]byte, operationsLen)
	for _, t := range types {
		if t < 0 || int(t) >= operationsLen*8 {
			return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
				Hint: fmt.Sprintf("contract type id %d is outside the 0..255 range the operations bitmap covers", int32(t))}
		}
		out[t/8] |= 1 << (t % 8)
	}
	return out, nil
}

// OperationsList decodes an operations bitmap back into the contract types it
// allows, in ascending id order.
func OperationsList(bitmap []byte) ([]ContractType, error) {
	const op = "tx.OperationsList"
	if len(bitmap) != operationsLen {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: fmt.Sprintf("an operations bitmap is %d bytes, got %d; build it with OperationsBitmap", operationsLen, len(bitmap))}
	}
	var out []ContractType
	for i, b := range bitmap {
		for bit := 0; bit < 8; bit++ {
			if b>>bit&1 == 1 {
				out = append(out, ContractType(i*8+bit))
			}
		}
	}
	return out, nil
}

// PermissionKey is one address authorized under a permission, with the weight
// its signature contributes.
type PermissionKey struct {
	// Address is the key holder's TRON address.
	Address tron.Address
	// Weight is the signature's contribution toward the threshold; it must be
	// positive.
	Weight int64
}

// Permission is one permission slot (owner, witness, or an active
// permission): the signers it accepts, the total weight required, and — for
// active permissions — the operations it may execute.
type Permission struct {
	// ID is the permission id (0 owner, 1 witness, 2–9 active). Builders
	// assign it from position; a value read from an account round-trips.
	ID int32
	// Name is the permission's label; at most 32 bytes. It may be empty.
	Name string
	// Threshold is the total signature weight required. It must be positive,
	// and the sum of key weights must reach it.
	Threshold int64
	// Keys are the authorized signers (at most 5, addresses distinct).
	Keys []PermissionKey
	// Operations is the 32-byte contract-type bitmap for an active
	// permission; ignored for owner and witness. Build it with
	// OperationsBitmap.
	Operations []byte
}

// PermissionSet is an account's complete permission configuration.
//
// AccountPermissionUpdate REPLACES all three slots at once — the node does
// not merge. The safe editing pattern is therefore: read the current set with
// Permissions().Current, change the one slot you mean to change, and submit
// the whole set back. Changing only one field and submitting the rest zeroed
// is the mistake this type exists to make obvious.
type PermissionSet struct {
	// Owner is the account's owner permission; it is required and is the
	// permission that can change permissions.
	Owner Permission
	// Witness is the block-production permission. It is set only on SR
	// accounts (exactly one key) and must be nil for everyone else.
	Witness *Permission
	// Actives are the account's active permissions, 1–8 of them. The node
	// assigns their ids in order (2, 3, …); the Operations bitmap scopes each
	// one to specific contract types.
	Actives []Permission
}

// BuildAccountPermissionUpdate builds an AccountPermissionUpdateContract from
// a complete PermissionSet. The transaction must be signed under the
// account's current owner permission, or under an active permission whose
// bitmap enables TypeAccountPermissionUpdate (select one with
// WithPermissionID).
//
// The node charges a fixed permission-update fee (getUpdateAccountPermissionFee,
// 100 TRX on Mainnet today) on top of bandwidth; TotalCostOf includes it.
//
// Two failure modes are worth stating because they are unrecoverable:
// permissions are validated structurally, but the node cannot verify that the
// operator controls the private keys in Keys; and if the new owner permission
// cannot meet its threshold, the account's permissions can never be changed
// again.
func BuildAccountPermissionUpdate(ctx context.Context, cp rpc.ConnProvider, owner tron.Address, set PermissionSet) (*NativeTx, error) {
	const op = "tx.BuildAccountPermissionUpdate"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	ownerPb, err := permissionPB(op, "owner", set.Owner, core.Permission_Owner, 0, false)
	if err != nil {
		return nil, err
	}
	var witnessPb *core.Permission
	if set.Witness != nil {
		witnessPb, err = permissionPB(op, "witness", *set.Witness, core.Permission_Witness, 1, true)
		if err != nil {
			return nil, err
		}
	}
	if len(set.Actives) == 0 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "actives must hold 1–8 active permissions; the contract replaces the whole configuration, so the node rejects an empty list"}
	}
	if len(set.Actives) > 8 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: fmt.Sprintf("an account may hold at most 8 active permissions, got %d", len(set.Actives))}
	}
	activesPb := make([]*core.Permission, 0, len(set.Actives))
	for i, active := range set.Actives {
		p, err := permissionPB(op, "actives["+itoa(int64(i))+"]", active, core.Permission_Active, int32(2+i), false)
		if err != nil {
			return nil, err
		}
		if len(active.Operations) != operationsLen {
			return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
				Hint: "active permission " + itoa(int64(i)) + " needs a 32-byte operations bitmap; build it with OperationsBitmap(TypeTransfer, …)"}
		}
		p.Operations = active.Operations
		activesPb = append(activesPb, p)
	}
	req := &core.AccountPermissionUpdateContract{
		OwnerAddress: owner.Bytes(),
		Owner:        ownerPb,
		Witness:      witnessPb,
		Actives:      activesPb,
	}
	ext, err := rpc.AccountPermissionUpdate(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// permissionPB validates one permission and renders it as the wire message.
// expectedID is the id the node assigns by position; a caller-supplied id that
// disagrees is rejected rather than silently overridden.
func permissionPB(op, field string, p Permission, ptype core.Permission_PermissionType, expectedID int32, witness bool) (*core.Permission, error) {
	if p.ID != 0 && p.ID != expectedID {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: fmt.Sprintf("%s.id is %d but its position assigns id %d; the node numbers permissions by position", field, p.ID, expectedID)}
	}
	if len(p.Name) > 32 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: fmt.Sprintf("%s.permission_name is %d bytes; the protocol limit is 32", field, len(p.Name))}
	}
	if witness && len(p.Keys) != 1 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: fmt.Sprintf("%s (witness permission) must hold exactly one key, got %d", field, len(p.Keys))}
	}
	if len(p.Keys) == 0 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: field + " needs at least one key; an account locked out of every permission cannot be recovered"}
	}
	if len(p.Keys) > 5 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: fmt.Sprintf("%s holds %d keys; the protocol allows at most 5 per permission", field, len(p.Keys))}
	}
	if p.Threshold <= 0 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: field + ".threshold must be positive; the node rejects a permission whose threshold is 0"}
	}
	var sum int64
	seen := make(map[tron.Address]struct{}, len(p.Keys))
	keys := make([]*core.Key, 0, len(p.Keys))
	for _, k := range p.Keys {
		if err := validateAddress(op, field+".keys", k.Address); err != nil {
			return nil, err
		}
		if _, dup := seen[k.Address]; dup {
			return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
				Hint: field + " lists " + k.Address.String() + " twice; the node rejects duplicate keys"}
		}
		seen[k.Address] = struct{}{}
		if k.Weight <= 0 {
			return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
				Hint: field + " has a key with non-positive weight; every key weight must be positive"}
		}
		sum += k.Weight
		keys = append(keys, &core.Key{Address: k.Address.Bytes(), Weight: k.Weight})
	}
	if sum < p.Threshold {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: fmt.Sprintf("%s.threshold is %d but its key weights sum to only %d; the threshold would be unreachable", field, p.Threshold, sum)}
	}
	return &core.Permission{
		Type:           ptype,
		Id:             expectedID,
		PermissionName: p.Name,
		Threshold:      p.Threshold,
		ParentId:       0,
		Keys:           keys,
	}, nil
}
