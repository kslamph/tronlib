package tx

import (
	"context"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// Contract management operations: setting updates the deployer-controlled
// parameters of a deployed contract. They consume bandwidth only — no TVM
// execution, no energy, no simulation path — so all three build NativeTx
// (the "other non-contract operations" bucket), each carrying only the
// options meaningful for it: expiration and permission id, never a fee
// limit. Only the contract's owner (deployer) account may broadcast them;
// the node rejects anyone else.

// BuildUpdateSetting builds an UpdateSettingContract transaction setting
// the contract's consume_user_resource_percent (0..100): the deployer's
// share of every call's energy cost (0 = deployer pays all, 100 =
// caller pays all). Out of range is tx.invalid_argument.
func BuildUpdateSetting(ctx context.Context, cp rpc.ConnProvider, owner, contract tron.Address, percent int64) (*NativeTx, error) {
	const op = "tx.BuildUpdateSetting"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	if err := validateAddress(op, "contract", contract); err != nil {
		return nil, err
	}
	if percent < 0 || percent > 100 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "consume_user_resource_percent must be between 0 and 100 (0 = deployer pays all energy, 100 = caller pays all)"}
	}
	req := &core.UpdateSettingContract{
		OwnerAddress:               owner.Bytes(),
		ContractAddress:            contract.Bytes(),
		ConsumeUserResourcePercent: percent,
	}
	ext, err := rpc.UpdateSetting(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// BuildUpdateEnergyLimit builds an UpdateEnergyLimitContract transaction
// setting the contract's origin_energy_limit: the cap on energy the
// contract itself may spend per call. Negative is tx.invalid_argument
// (v1 parity); zero is accepted — the node interprets it.
func BuildUpdateEnergyLimit(ctx context.Context, cp rpc.ConnProvider, owner, contract tron.Address, limit int64) (*NativeTx, error) {
	const op = "tx.BuildUpdateEnergyLimit"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	if err := validateAddress(op, "contract", contract); err != nil {
		return nil, err
	}
	if limit < 0 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "origin_energy_limit cannot be negative"}
	}
	req := &core.UpdateEnergyLimitContract{
		OwnerAddress:      owner.Bytes(),
		ContractAddress:   contract.Bytes(),
		OriginEnergyLimit: limit,
	}
	ext, err := rpc.UpdateEnergyLimit(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// BuildClearABI builds a ClearABIContract transaction removing the
// contract's published ABI. The contract keeps running — only its
// on-chain interface description is dropped.
func BuildClearABI(ctx context.Context, cp rpc.ConnProvider, owner, contract tron.Address) (*NativeTx, error) {
	const op = "tx.BuildClearABI"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	if err := validateAddress(op, "contract", contract); err != nil {
		return nil, err
	}
	req := &core.ClearABIContract{
		OwnerAddress:    owner.Bytes(),
		ContractAddress: contract.Bytes(),
	}
	ext, err := rpc.ClearContractABI(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}
