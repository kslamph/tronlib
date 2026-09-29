package tx

import (
	"context"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// DefaultFeeLimit is the fee limit every builder applies at build time
// (150_000_000 SUN = 150 TRX, v1's DefaultBroadcastOptions value, carried
// over deliberately rather than invented). A transaction broadcast with
// fee_limit 0 cannot purchase energy and fails; the default is a floor, and
// CostPreview compares it against the energy estimate before broadcast
// (architecture §6.4).
const DefaultFeeLimit = tron.SUN(150_000_000)

// validateAddress errors on the unset (zero) address — v2's value-type
// replacement for v1's nil-address rejection ("nil/zero addresses ->
// address.invalid via IsZero").
func validateAddress(op, field string, a tron.Address) error {
	if a.IsZero() {
		return &tron.Error{
			Code: tron.CodeAddressInvalid,
			Op:   op,
			Hint: field + " is the zero address; pass a parsed tron.Address (tron.ParseAddress)",
		}
	}
	return nil
}

// validateAmount errors on a negative amount (amount.negative) and a zero
// amount (amount.invalid, v1 requires strictly positive). SUN is already
// scaled: construct with tron.TRX(n) or tron.ParseTRX("1.5").
func validateAmount(op string, amt tron.SUN) error {
	if amt < 0 {
		return &tron.Error{
			Code: tron.CodeAmountNegative,
			Op:   op,
			Hint: "amount cannot be negative",
		}
	}
	if amt == 0 {
		return &tron.Error{
			Code: tron.CodeAmountInvalid,
			Op:   op,
			Hint: "amount must be positive; construct with tron.TRX(n) or tron.ParseTRX(\"1.5\")",
		}
	}
	return nil
}

// applyBuildDefaults enforces the documented default (architecture §6.4: "every
// builder applies a documented default") on a fresh build response: the
// node's CreateTransaction2-family responses may report fee_limit 0, which
// cannot purchase energy, so the builder floors it at DefaultFeeLimit
// (150_000_000). Expiration is NOT touched: the node sets it server-side
// (head + 60 s); WithExpiration mutates it post-build when a longer
// multi-signer circulation is needed. It mutates the response the builder
// just received and validated.
func applyBuildDefaults(ext *api.TransactionExtention) {
	raw := ext.GetTransaction().GetRawData()
	if raw == nil {
		return // requireRaw produces the typed error for this shape
	}
	if raw.GetFeeLimit() == 0 {
		raw.FeeLimit = int64(DefaultFeeLimit)
	}
}

// wrapBuildResult converts a validated node build response into a
// transaction kind. A failed Return never reaches it: the TxCall wrappers
// already map it through the typed node-return cause (rpc.NodeReturnCode
// seam, Task 5) onto the v2 code with the raw node code preserved in the
// error's Cause.
func wrapBuildResult[T any](mk func(*baseTx) T, ext *api.TransactionExtention, kind Kind, cp rpc.ConnProvider, op string) (T, error) {
	var zero T
	if err := requireRaw(ext, op); err != nil {
		return zero, err
	}
	return mk(&baseTx{ext: ext, kind: kind, cp: cp}), nil
}

// BuildTransfer builds a TRX transfer (NativeTx) via the node's
// CreateTransaction2 — the node fills raw_data (TAPOS reference, timestamp,
// expiration); I/O is real.
func BuildTransfer(cp rpc.ConnProvider, ctx context.Context, from, to tron.Address, amt tron.SUN) (*NativeTx, error) {
	const op = "tx.BuildTransfer"
	if err := validateAddress(op, "from", from); err != nil {
		return nil, err
	}
	if err := validateAddress(op, "to", to); err != nil {
		return nil, err
	}
	if err := validateAmount(op, amt); err != nil {
		return nil, err
	}
	req := &core.TransferContract{
		OwnerAddress: from.Bytes(),
		ToAddress:    to.Bytes(),
		Amount:       int64(amt),
	}
	ext, err := rpc.CreateTransaction2(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	applyBuildDefaults(ext)
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// BuildTriggerSmartContract builds a TriggerSmartContract transaction
// (ContractTx) via the node's TriggerContract build RPC. data is the ABI
// encoded selector + arguments (contract-layer concern, architecture §3); callValue
// is the SUN sent with the call and may be zero.
func BuildTriggerSmartContract(cp rpc.ConnProvider, ctx context.Context, owner, contract tron.Address, data []byte, callValue tron.SUN) (*ContractTx, error) {
	const op = "tx.BuildTriggerSmartContract"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	if err := validateAddress(op, "contract", contract); err != nil {
		return nil, err
	}
	if callValue < 0 {
		return nil, &tron.Error{Code: tron.CodeAmountNegative, Op: op, Hint: "call value cannot be negative"}
	}
	req := &core.TriggerSmartContract{
		OwnerAddress:    owner.Bytes(),
		ContractAddress: contract.Bytes(),
		Data:            data,
		CallValue:       int64(callValue),
	}
	ext, err := rpc.TriggerContract(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	applyBuildDefaults(ext)
	return wrapBuildResult(func(b *baseTx) *ContractTx {
		return &ContractTx{baseTx: *b}
	}, ext, KindContract, cp, op)
}

// BuildDeploy builds a CreateSmartContract transaction (DeployTx) via the
// node's DeployContract build RPC. It validates the DeployParams the way
// v1's smartcontract.Manager.Deploy does (bytecode non-empty, resource
// percent 0–100, origin energy limit >= 0, call value >= 0, visible-character
// contract name). ABI parsing is a contract-layer concern: p.ABI is the
// already-parsed pb ABI, and p.Bytecode is the final bytecode (constructor
// arguments already appended).
func BuildDeploy(cp rpc.ConnProvider, ctx context.Context, owner tron.Address, p DeployParams) (*DeployTx, error) {
	const op = "tx.BuildDeploy"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	if !validContractName(p.Name) {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "contract name contains non-visible (control) characters"}
	}
	if len(p.Bytecode) == 0 {
		return nil, &tron.Error{Code: tron.CodeContractBadABI, Op: op, Hint: "bytecode cannot be empty"}
	}
	if p.CallValue < 0 {
		return nil, &tron.Error{Code: tron.CodeAmountNegative, Op: op, Hint: "call value cannot be negative"}
	}
	if p.ConsumeUserResourcePercent < 0 || p.ConsumeUserResourcePercent > 100 {
		return nil, &tron.Error{
			Code: tron.CodeTxInvalidArgument,
			Op:   op,
			Hint: "consume user resource percent must be between 0 and 100",
		}
	}
	if p.OriginEnergyLimit < 0 {
		return nil, &tron.Error{
			Code: tron.CodeTxInvalidArgument,
			Op:   op,
			Hint: "origin energy limit cannot be negative",
		}
	}
	req := &core.CreateSmartContract{
		OwnerAddress: owner.Bytes(),
		NewContract: &core.SmartContract{
			OriginAddress:              owner.Bytes(),
			Abi:                        p.ABI,
			Bytecode:                   p.Bytecode,
			CallValue:                  int64(p.CallValue),
			ConsumeUserResourcePercent: p.ConsumeUserResourcePercent,
			Name:                       p.Name,
			OriginEnergyLimit:          p.OriginEnergyLimit,
		},
	}
	ext, err := rpc.DeployContract(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	applyBuildDefaults(ext)
	return wrapBuildResult(func(b *baseTx) *DeployTx {
		return &DeployTx{baseTx: *b}
	}, ext, KindDeploy, cp, op)
}

// BuildAssetTransfer builds a TransferAssetContract transaction (AssetTx)
// via the node's TransferAsset2 build RPC. assetName is the TRC-10 token
// identifier (its id or name form as the node expects it) and must be
// non-empty; qty must be positive; a self-transfer is rejected like v1's
// TransferAsset2 does.
func BuildAssetTransfer(cp rpc.ConnProvider, ctx context.Context, from, to tron.Address, assetName string, qty int64) (*AssetTx, error) {
	const op = "tx.BuildAssetTransfer"
	if err := validateAddress(op, "from", from); err != nil {
		return nil, err
	}
	if err := validateAddress(op, "to", to); err != nil {
		return nil, err
	}
	if assetName == "" {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "asset name cannot be empty"}
	}
	if qty < 0 {
		return nil, &tron.Error{Code: tron.CodeAmountNegative, Op: op, Hint: "quantity cannot be negative"}
	}
	if qty == 0 {
		return nil, &tron.Error{Code: tron.CodeAmountInvalid, Op: op, Hint: "quantity must be positive"}
	}
	if from == to {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "from and to are the same address"}
	}
	req := &core.TransferAssetContract{
		OwnerAddress: from.Bytes(),
		ToAddress:    to.Bytes(),
		AssetName:    []byte(assetName),
		Amount:       qty,
	}
	ext, err := rpc.TransferAsset2(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	applyBuildDefaults(ext)
	return wrapBuildResult(func(b *baseTx) *AssetTx {
		return &AssetTx{baseTx: *b}
	}, ext, KindAssetTransfer, cp, op)
}
