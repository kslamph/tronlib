package tx

import (
	"context"
	"fmt"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// Stake 2.0 (TIP-467) and resource delegation.
//
// Staking and delegation are independent operations on TRON. A stake turns
// TRX into one of two resources — Bandwidth or Energy — and grants TRON Power
// (1 TP per TRX under the current Mainnet resource model). Delegation lends
// the staked resource to another account without unstaking: the TRX stays
// staked under the owner, the recipient gets to spend the resource.
//
// Every amount in this file is TRX in SUN — the staked principal, never an
// Energy or Bandwidth quantity. How much Energy a given stake buys depends on
// the network-wide staked total, so the resource amount is a chain state, not
// a property of the transaction; read it back with ResourceStateOf.
//
// The unstake lifecycle is deliberately three verbs, because it has three
// distinct states and conflating them is how callers lose track of their TRX:
//
//	Stake ──► staked (earning resources)
//	Unstake ──► pending unfreeze (cooldown, chain-configured delay)
//	WithdrawUnstaked ──► spendable balance
//	CancelUnstake ──► back to staked, or auto-withdraws anything already matured
//
// WithdrawBalanceContract (voting rewards) is a different operation and is
// never called "withdraw" here; see BuildWithdrawRewards.

// Resource names one of the two delegatable, stakeable resources. TRON Power
// is deliberately absent: it is not delegatable, and under the current
// Mainnet resource model it is granted by the same stake rather than staked
// separately.
type Resource int

const (
	// ResourceBandwidth pays for transaction byte size.
	ResourceBandwidth Resource = iota
	// ResourceEnergy pays for smart-contract execution.
	ResourceEnergy
)

// ResourceUnknown marks a resource code the curated set does not cover — TRON
// Power, or any code a future node adds. It is deliberately outside the
// [ResourceBandwidth, ResourceEnergy] range and is rejected by every builder,
// so a decode of an unmapped code can be told apart from Bandwidth (which is
// the zero value and would otherwise read as "this account staked Bandwidth").
// Decoders return it; nothing accepts it.
const ResourceUnknown Resource = -1

// String returns the resource name as the protocol's ResourceCode spells it.
func (r Resource) String() string {
	switch r {
	case ResourceBandwidth:
		return "BANDWIDTH"
	case ResourceEnergy:
		return "ENERGY"
	default:
		return "UNKNOWN"
	}
}

// proto maps the resource onto the wire enum.
func (r Resource) proto() core.ResourceCode {
	if r == ResourceEnergy {
		return core.ResourceCode_ENERGY
	}
	return core.ResourceCode_BANDWIDTH
}

// validateResource rejects any value outside the two delegatable resources,
// naming the TRON Power case explicitly: it exists on the wire but is only
// meaningful under the new resource model, and silently mapping it to
// Bandwidth would stake the wrong thing.
func validateResource(op string, r Resource) error {
	switch r {
	case ResourceBandwidth, ResourceEnergy:
		return nil
	default:
		return &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "resource must be ResourceEnergy or ResourceBandwidth; TRON Power cannot be staked or delegated through this API"}
	}
}

// DelegateOptions configures a delegation lock.
//
// A locked delegation guarantees the recipient keeps the resource for a
// period the delegator cannot shorten; an unlocked delegation can be
// undelegated at any time. LockBlocks is measured in BLOCKS, not wall-clock
// time (TRON produces one block every ~3 seconds): lock=true with 0 asks the
// node for its default of 86,400 blocks (~3 days), and the network's maximum
// is a governance parameter (getMaxDelegateLockPeriod).
type DelegateOptions struct {
	// Lock, when true, makes the delegation non-cancellable until LockBlocks.
	Lock bool
	// LockBlocks is the lock length in blocks. It is meaningful only when
	// Lock is true.
	LockBlocks int64
}

// BuildFreezeBalanceV2 builds a Stake 2.0 stake (FreezeBalanceV2Contract).
// amount is the TRX to stake, in SUN, and must be positive. The unstaked TRX
// leaves the spendable balance immediately; read the resulting resource share
// back with ResourceStateOf.
func BuildFreezeBalanceV2(ctx context.Context, cp rpc.ConnProvider, owner tron.Address, res Resource, amount tron.SUN) (*NativeTx, error) {
	const op = "tx.BuildFreezeBalanceV2"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	if err := validateResource(op, res); err != nil {
		return nil, err
	}
	if err := validateAmount(op, amount); err != nil {
		return nil, err
	}
	req := &core.FreezeBalanceV2Contract{
		OwnerAddress:  owner.Bytes(),
		FrozenBalance: int64(amount),
		Resource:      res.proto(),
	}
	ext, err := rpc.FreezeBalanceV2(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// BuildUnfreezeBalanceV2 builds a Stake 2.0 unstake (UnfreezeBalanceV2Contract),
// which starts the cooldown rather than returning TRX. The delay is a chain
// parameter (currently 14 days on Mainnet, 1 day on Nile) and the funds become
// claimable with BuildWithdrawExpireUnfreeze — or, for anything already
// matured, automatically inside the next unstake.
//
// At most 32 unstake operations may be pending per account; the remaining
// slots are UnfreezeSlotsOf. TRX that is currently delegated cannot be
// unstaked: undelegate first.
func BuildUnfreezeBalanceV2(ctx context.Context, cp rpc.ConnProvider, owner tron.Address, res Resource, amount tron.SUN) (*NativeTx, error) {
	const op = "tx.BuildUnfreezeBalanceV2"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	if err := validateResource(op, res); err != nil {
		return nil, err
	}
	if err := validateAmount(op, amount); err != nil {
		return nil, err
	}
	req := &core.UnfreezeBalanceV2Contract{
		OwnerAddress:    owner.Bytes(),
		UnfreezeBalance: int64(amount),
		Resource:        res.proto(),
	}
	ext, err := rpc.UnfreezeBalanceV2(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// BuildWithdrawExpireUnfreeze builds a WithdrawExpireUnfreezeContract: it
// moves every unstake that has cleared its cooldown into the spendable
// balance. It takes no amount — the node withdraws everything matured.
func BuildWithdrawExpireUnfreeze(ctx context.Context, cp rpc.ConnProvider, owner tron.Address) (*NativeTx, error) {
	const op = "tx.BuildWithdrawExpireUnfreeze"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	req := &core.WithdrawExpireUnfreezeContract{OwnerAddress: owner.Bytes()}
	ext, err := rpc.WithdrawExpireUnfreeze(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// BuildCancelAllUnfreezeV2 builds a CancelAllUnfreezeV2Contract: it re-stakes
// every unstake still inside its cooldown and auto-withdraws the ones that
// have already matured. Re-staking restores the same resources and TRON Power
// the stake held, which makes this the cheap way to undo a change of mind
// before the cooldown elapses.
func BuildCancelAllUnfreezeV2(ctx context.Context, cp rpc.ConnProvider, owner tron.Address) (*NativeTx, error) {
	const op = "tx.BuildCancelAllUnfreezeV2"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	req := &core.CancelAllUnfreezeV2Contract{OwnerAddress: owner.Bytes()}
	ext, err := rpc.CancelAllUnfreezeV2(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// BuildDelegateResource builds a DelegateResourceContract: it lends staked
// Bandwidth or Energy to receiver. amount is the TRX (in SUN) whose resource
// share is lent, not a resource quantity, and must be at least 1 TRX — the
// protocol's minimum delegation. The TRX stays staked under owner.
//
// The recipient must be an externally-owned account that is not the owner
// and not a contract address; the node enforces this (the check would cost an
// extra account/contract read here, and the node's rejection is precise).
func BuildDelegateResource(ctx context.Context, cp rpc.ConnProvider, owner, receiver tron.Address, res Resource, amount tron.SUN, opts DelegateOptions) (*NativeTx, error) {
	const op = "tx.BuildDelegateResource"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	if err := validateAddress(op, "receiver", receiver); err != nil {
		return nil, err
	}
	if owner == receiver {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "cannot delegate to yourself; receiver must be a different account"}
	}
	if err := validateResource(op, res); err != nil {
		return nil, err
	}
	if err := validateAmount(op, amount); err != nil {
		return nil, err
	}
	if amount < tron.TRX(1) {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "the protocol requires a delegation of at least 1 TRX (1_000_000 SUN) of stake"}
	}
	if opts.LockBlocks < 0 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op, Hint: "lock blocks cannot be negative"}
	}
	if !opts.Lock && opts.LockBlocks != 0 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "lock blocks is only meaningful with Lock: true; the node ignores lock_period on an unlocked delegation"}
	}
	req := &core.DelegateResourceContract{
		OwnerAddress:    owner.Bytes(),
		Resource:        res.proto(),
		Balance:         int64(amount),
		ReceiverAddress: receiver.Bytes(),
		Lock:            opts.Lock,
		LockPeriod:      opts.LockBlocks,
	}
	ext, err := rpc.DelegateResource(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// BuildUnDelegateResource builds an UnDelegateResourceContract: it returns
// the resource share to owner. amount is TRX in SUN, as in the delegate call.
// It is immediate for an unlocked delegation; a locked delegation can only be
// undelegated after its lock expires.
//
// Undelegating also reclaims a proportional share of the recipient's
// still-recovering (unrecovered) resources, so the recipient's available
// resources can drop for reasons they did not initiate.
func BuildUnDelegateResource(ctx context.Context, cp rpc.ConnProvider, owner, receiver tron.Address, res Resource, amount tron.SUN) (*NativeTx, error) {
	const op = "tx.BuildUnDelegateResource"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	if err := validateAddress(op, "receiver", receiver); err != nil {
		return nil, err
	}
	if owner == receiver {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "cannot undelegate from yourself; receiver must be the account this owner delegated to"}
	}
	if err := validateResource(op, res); err != nil {
		return nil, err
	}
	if err := validateAmount(op, amount); err != nil {
		return nil, err
	}
	req := &core.UnDelegateResourceContract{
		OwnerAddress:    owner.Bytes(),
		Resource:        res.proto(),
		Balance:         int64(amount),
		ReceiverAddress: receiver.Bytes(),
	}
	ext, err := rpc.UnDelegateResource(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// ResourceState is an account's decoded GetAccountResource answer: the
// per-resource limits and the usage currently charged against them. All
// values are in the resource's own unit (Energy units, Bandwidth bytes, TRON
// Power), except the network totals/weights which are SUN-denominated.
//
// Available resource is Limit − Used. Usage decays over a rolling 24-hour
// window, so a read is a snapshot, not a guarantee for a later transaction.
type ResourceState struct {
	// Address is the account the state belongs to.
	Address tron.Address

	// EnergyLimit and EnergyUsed are the account's Energy share and what is
	// currently charged against it.
	EnergyLimit int64
	EnergyUsed  int64
	// BandwidthLimit and BandwidthUsed are the staked Bandwidth share and
	// its current usage.
	BandwidthLimit int64
	BandwidthUsed  int64
	// FreeBandwidthLimit and FreeBandwidthUsed are the network-wide free
	// quota (600/day) and its current usage.
	FreeBandwidthLimit int64
	FreeBandwidthUsed  int64
	// TronPowerLimit and TronPowerUsed are the voting power granted by the
	// stake and the amount already voted.
	TronPowerLimit int64
	TronPowerUsed  int64

	// TotalEnergyLimit / TotalEnergyWeight describe the network-wide Energy
	// pool the account's stake divides into.
	TotalEnergyLimit  int64
	TotalEnergyWeight int64
	// TotalBandwidthLimit / TotalBandwidthWeight are the Bandwidth
	// equivalents.
	TotalBandwidthLimit  int64
	TotalBandwidthWeight int64
}

// EnergyAvailable is the Energy the account can spend right now.
func (s *ResourceState) EnergyAvailable() int64 { return s.EnergyLimit - s.EnergyUsed }

// BandwidthAvailable is staked Bandwidth plus the free quota minus their
// current usage — the order the node charges them in.
func (s *ResourceState) BandwidthAvailable() int64 {
	return (s.BandwidthLimit - s.BandwidthUsed) + (s.FreeBandwidthLimit - s.FreeBandwidthUsed)
}

// TronPowerAvailable is the unused voting power; voting is what turns TRON
// Power into rewards.
func (s *ResourceState) TronPowerAvailable() int64 { return s.TronPowerLimit - s.TronPowerUsed }

// ResourceStateOf reads an account's resource state (GetAccountResource).
// It works for any address; an unknown account yields a zero state with no
// error, like account.Handle.Balance.
func ResourceStateOf(ctx context.Context, cp rpc.ConnProvider, owner tron.Address) (*ResourceState, error) {
	const op = "tx.ResourceStateOf"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	msg, err := rpc.GetAccountResource(cp, ctx, &core.Account{Address: owner.Bytes()})
	if err != nil {
		return nil, err
	}
	return &ResourceState{
		Address:              owner,
		EnergyLimit:          msg.GetEnergyLimit(),
		EnergyUsed:           msg.GetEnergyUsed(),
		BandwidthLimit:       msg.GetNetLimit(),
		BandwidthUsed:        msg.GetNetUsed(),
		FreeBandwidthLimit:   msg.GetFreeNetLimit(),
		FreeBandwidthUsed:    msg.GetFreeNetUsed(),
		TronPowerLimit:       msg.GetTronPowerLimit(),
		TronPowerUsed:        msg.GetTronPowerUsed(),
		TotalEnergyLimit:     msg.GetTotalEnergyLimit(),
		TotalEnergyWeight:    msg.GetTotalEnergyWeight(),
		TotalBandwidthLimit:  msg.GetTotalNetLimit(),
		TotalBandwidthWeight: msg.GetTotalNetWeight(),
	}, nil
}

// DelegatableOf reads how much stake of res the owner can still delegate
// right now (GetCanDelegatedMaxSize). Recently consumed resources reduce the
// figure until their 24-hour recovery window clears.
func DelegatableOf(ctx context.Context, cp rpc.ConnProvider, owner tron.Address, res Resource) (tron.SUN, error) {
	const op = "tx.DelegatableOf"
	if err := validateAddress(op, "owner", owner); err != nil {
		return 0, err
	}
	if err := validateResource(op, res); err != nil {
		return 0, err
	}
	msg, err := rpc.GetCanDelegatedMaxSize(cp, ctx, &api.CanDelegatedMaxSizeRequestMessage{
		OwnerAddress: owner.Bytes(),
		Type:         int32(res.proto()),
	})
	if err != nil {
		return 0, err
	}
	return tron.SUN(msg.GetMaxSize()), nil
}

// UnfreezeSlotsOf reads how many unstake operations the account can still
// start (GetAvailableUnfreezeCount). The protocol caps concurrent pending
// unstakes at 32; this is the remaining count.
func UnfreezeSlotsOf(ctx context.Context, cp rpc.ConnProvider, owner tron.Address) (int64, error) {
	const op = "tx.UnfreezeSlotsOf"
	if err := validateAddress(op, "owner", owner); err != nil {
		return 0, err
	}
	msg, err := rpc.GetAvailableUnfreezeCount(cp, ctx, &api.GetAvailableUnfreezeCountRequestMessage{
		OwnerAddress: owner.Bytes(),
	})
	if err != nil {
		return 0, err
	}
	return msg.GetCount(), nil
}

// WithdrawableUnfreezeOf reads the TRX that has already cleared its unstake
// cooldown and can be moved to the spendable balance
// (GetCanWithdrawUnfreezeAmount, evaluated at the current time).
func WithdrawableUnfreezeOf(ctx context.Context, cp rpc.ConnProvider, owner tron.Address) (tron.SUN, error) {
	const op = "tx.WithdrawableUnfreezeOf"
	if err := validateAddress(op, "owner", owner); err != nil {
		return 0, err
	}
	msg, err := rpc.GetCanWithdrawUnfreezeAmount(cp, ctx, &api.CanWithdrawUnfreezeAmountRequestMessage{
		OwnerAddress: owner.Bytes(),
		Timestamp:    time.Now().UnixMilli(),
	})
	if err != nil {
		return 0, err
	}
	return tron.SUN(msg.GetAmount()), nil
}

// Delegation is one from→to resource delegation record, in TRX SUN.
type Delegation struct {
	// From is the delegating account.
	From tron.Address
	// To is the receiving account.
	To tron.Address
	// Bandwidth is the TRX (SUN) whose Bandwidth share is delegated.
	Bandwidth tron.SUN
	// Energy is the TRX (SUN) whose Energy share is delegated.
	Energy tron.SUN
	// BandwidthExpiresAt is when a locked Bandwidth delegation unlocks; the
	// zero time means unlocked (or already expired), i.e. undelegatable now.
	BandwidthExpiresAt time.Time
	// EnergyExpiresAt is the Energy equivalent.
	EnergyExpiresAt time.Time
}

// DelegationsOf reads the delegation records between from and to
// (GetDelegatedResourceV2). An empty result means no delegation exists. Both
// directions are queried by passing the accounts in the other order.
func DelegationsOf(ctx context.Context, cp rpc.ConnProvider, from, to tron.Address) ([]Delegation, error) {
	const op = "tx.DelegationsOf"
	if err := validateAddress(op, "from", from); err != nil {
		return nil, err
	}
	if err := validateAddress(op, "to", to); err != nil {
		return nil, err
	}
	list, err := rpc.GetDelegatedResourceV2(cp, ctx, &api.DelegatedResourceMessage{
		FromAddress: from.Bytes(),
		ToAddress:   to.Bytes(),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Delegation, 0, len(list.GetDelegatedResource()))
	for _, r := range list.GetDelegatedResource() {
		out = append(out, Delegation{
			From:               from,
			To:                 to,
			Bandwidth:          tron.SUN(r.GetFrozenBalanceForBandwidth()),
			Energy:             tron.SUN(r.GetFrozenBalanceForEnergy()),
			BandwidthExpiresAt: millisOrZero(r.GetExpireTimeForBandwidth()),
			EnergyExpiresAt:    millisOrZero(r.GetExpireTimeForEnergy()),
		})
	}
	return out, nil
}

// DelegationIndex is the account's delegation graph: who delegates to it and
// whom it delegates to (GetDelegatedResourceAccountIndexV2). Use it to
// discover counterparts and then DelegationsOf for the amounts, instead of
// paying one read per account.
type DelegationIndex struct {
	// From lists accounts that delegate resources TO this account.
	From []tron.Address
	// To lists accounts this account delegates resources to.
	To []tron.Address
}

// DelegationIndexOf reads the account's delegation index.
func DelegationIndexOf(ctx context.Context, cp rpc.ConnProvider, account tron.Address) (*DelegationIndex, error) {
	const op = "tx.DelegationIndexOf"
	if err := validateAddress(op, "account", account); err != nil {
		return nil, err
	}
	idx, err := rpc.GetDelegatedResourceAccountIndexV2(cp, ctx, &api.BytesMessage{Value: account.Bytes()})
	if err != nil {
		return nil, err
	}
	decode := func(raw [][]byte, field string) ([]tron.Address, error) {
		out := make([]tron.Address, 0, len(raw))
		for _, b := range raw {
			a, err := tron.AddressFromBytes(b)
			if err != nil {
				return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Cause: err,
					Hint: "the node returned a " + field + " address that is not a 0x41-prefixed 21-byte value"}
			}
			out = append(out, a)
		}
		return out, nil
	}
	from, err := decode(idx.GetFromAccounts(), "delegator")
	if err != nil {
		return nil, err
	}
	to, err := decode(idx.GetToAccounts(), "receiver")
	if err != nil {
		return nil, err
	}
	return &DelegationIndex{From: from, To: to}, nil
}

// millisOrZero converts a protocol millisecond timestamp into a Time,
// mapping the protocol's 0 ("no lock") and any non-positive value to the zero
// time.
func millisOrZero(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// ResourceStateString renders the state as one line for logs. Energy and
// Bandwidth are counts; TRON Power is votes.
func (s *ResourceState) String() string {
	return fmt.Sprintf("energy %d/%d (avail %d), bandwidth %d/%d + free %d/%d, tron power %d/%d",
		s.EnergyUsed, s.EnergyLimit, s.EnergyAvailable(),
		s.BandwidthUsed, s.BandwidthLimit, s.FreeBandwidthUsed, s.FreeBandwidthLimit,
		s.TronPowerUsed, s.TronPowerLimit)
}
