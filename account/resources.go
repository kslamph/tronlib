package account

import (
	"context"

	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// Resources is the staking and delegation handle for one account.
//
// The unstake lifecycle has three states, and this API keeps them separate so
// a caller always knows which one their TRX is in:
//
//	Stake           TRX is staked, producing a resource
//	Unstake         TRX is unstaked, in the chain's cooldown (not spendable)
//	WithdrawUnstaked  matured TRX moves to the spendable balance
//	CancelUnstake   anything still in cooldown goes back to staked
//
// Amounts are TRX in SUN throughout: the staked principal, never an Energy or
// Bandwidth quantity. A stake of 100 TRX produces a resource share that
// depends on the network-wide staked total, so read it back with State rather
// than assuming a rate.
type Resources struct {
	cp    rpc.ConnProvider
	owner tron.Address
}

// Owner returns the account this handle is bound to.
func (r *Resources) Owner() tron.Address { return r.owner }

// State reads the account's resource state: what limits each resource has, how
// much is currently used, and the TRON Power available for voting
// (tx.ResourceStateOf).
func (r *Resources) State(ctx context.Context) (*tx.ResourceState, error) {
	return tx.ResourceStateOf(ctx, r.cp, r.owner)
}

// Stake stakes amount TRX for res (FreezeBalanceV2). The TRX leaves the
// spendable balance immediately and produces a share of the network's
// Bandwidth or Energy, plus TRON Power (1 per TRX under the current Mainnet
// resource model) that voting can use.
func (r *Resources) Stake(ctx context.Context, res tx.Resource, amount tron.SUN) (*tx.NativeTx, error) {
	return tx.BuildFreezeBalanceV2(ctx, r.cp, r.owner, res, amount)
}

// Unstake starts the cooldown for amount TRX staked for res
// (UnfreezeBalanceV2). The TRX is NOT spendable afterwards: it becomes
// withdrawable only after the chain's unstake delay (14 days on Mainnet, 1 on
// Nile — read it from tx.ChainParamsOf rather than assuming). TRX currently
// delegated to another account cannot be unstaked; undelegate first.
//
// An unstake also reclaims TRON Power: idle (unvoted) power first, then
// proportionally from the voted witnesses.
func (r *Resources) Unstake(ctx context.Context, res tx.Resource, amount tron.SUN) (*tx.NativeTx, error) {
	return tx.BuildUnfreezeBalanceV2(ctx, r.cp, r.owner, res, amount)
}

// WithdrawUnstaked moves every unstake that has cleared its cooldown into the
// spendable balance (WithdrawExpireUnfreeze). It takes no amount: the node
// withdraws everything matured. Read Withdrawable first to see how much that
// is.
func (r *Resources) WithdrawUnstaked(ctx context.Context) (*tx.NativeTx, error) {
	return tx.BuildWithdrawExpireUnfreeze(ctx, r.cp, r.owner)
}

// CancelUnstake re-stakes every unstake still inside its cooldown and
// auto-withdraws the ones that have already matured (CancelAllUnfreezeV2). It
// restores the same resources and TRON Power the stake held. The operation is
// gated by a chain parameter (getAllowCancelAllUnfreezeV2); if the node has it
// disabled, the broadcast is rejected.
func (r *Resources) CancelUnstake(ctx context.Context) (*tx.NativeTx, error) {
	return tx.BuildCancelAllUnfreezeV2(ctx, r.cp, r.owner)
}

// Withdrawable reads how much TRX (SUN) has already cleared its cooldown and
// can be withdrawn now (tx.WithdrawableUnfreezeOf).
func (r *Resources) Withdrawable(ctx context.Context) (tron.SUN, error) {
	return tx.WithdrawableUnfreezeOf(ctx, r.cp, r.owner)
}

// UnstakeSlots reads how many more unstake operations can be started
// (tx.UnfreezeSlotsOf). At most 32 may be pending at once; the node's limit is
// fixed, not a chain parameter.
func (r *Resources) UnstakeSlots(ctx context.Context) (int64, error) {
	return tx.UnfreezeSlotsOf(ctx, r.cp, r.owner)
}

// Delegatable reads how much stake of res this account can still delegate
// right now (tx.DelegatableOf) — the staked balance minus any resource still
// inside its 24-hour recovery window.
func (r *Resources) Delegatable(ctx context.Context, res tx.Resource) (tron.SUN, error) {
	return tx.DelegatableOf(ctx, r.cp, r.owner, res)
}

// Delegate lends amount TRX (SUN) of staked res to receiver
// (DelegateResource). The TRX stays staked under this account; the recipient
// gets to spend the resource share. The protocol requires at least 1 TRX per
// delegation.
//
// DelegateOptions{Lock: true, LockBlocks: n} makes the delegation
// non-cancellable for n BLOCKS (~3 s each), which is how a DApp operator
// guarantees a user keeps the resource. An unlocked delegation
// (DelegateOptions{}) can be undelegated at any time.
func (r *Resources) Delegate(ctx context.Context, res tx.Resource, receiver tron.Address, amount tron.SUN, opts tx.DelegateOptions) (*tx.NativeTx, error) {
	return tx.BuildDelegateResource(ctx, r.cp, r.owner, receiver, res, amount, opts)
}

// Undelegate returns the resource share to this account (UnDelegateResource).
// It is immediate for an unlocked delegation; a locked one can only be
// undelegated after its lock expires.
//
// The recipient loses a proportional share of their still-recovering resources
// as a side effect, so a recipient's available resources can drop without
// their doing anything.
func (r *Resources) Undelegate(ctx context.Context, res tx.Resource, receiver tron.Address, amount tron.SUN) (*tx.NativeTx, error) {
	return tx.BuildUnDelegateResource(ctx, r.cp, r.owner, receiver, res, amount)
}

// DelegationsGrantedTo reads the delegation records from this account to
// receiver (tx.DelegationsOf). An empty result means no delegation exists;
// the records carry the locked/unlocked expiry per resource.
func (r *Resources) DelegationsGrantedTo(ctx context.Context, receiver tron.Address) ([]tx.Delegation, error) {
	return tx.DelegationsOf(ctx, r.cp, r.owner, receiver)
}

// DelegationsReceivedFrom reads the delegation records from delegator to this
// account (tx.DelegationsOf).
func (r *Resources) DelegationsReceivedFrom(ctx context.Context, delegator tron.Address) ([]tx.Delegation, error) {
	return tx.DelegationsOf(ctx, r.cp, delegator, r.owner)
}

// DelegationIndex reads the account's delegation graph — who delegates to it
// and whom it delegates to — without paying one read per counterparty
// (tx.DelegationIndexOf). Use the returned addresses with
// DelegationsGrantedTo / DelegationsReceivedFrom for the amounts.
func (r *Resources) DelegationIndex(ctx context.Context) (*tx.DelegationIndex, error) {
	return tx.DelegationIndexOf(ctx, r.cp, r.owner)
}

// Summary is a one-call overview of what an account holds and owes in
// resources: the staked and pending TRX, the delegation totals and the
// currently available resource, all decoded from a single account read plus a
// single resource read.
type Summary struct {
	// StakedByResource maps each resource to the TRX (SUN) staked for it.
	StakedByResource map[tx.Resource]tron.SUN
	// UnstakePending is the TRX (SUN) inside the cooldown, and
	// UnstakeWithdrawable is the part that can be withdrawn now.
	UnstakePending      tron.SUN
	UnstakeWithdrawable tron.SUN
	// DelegatedOut and DelegatedIn are the TRX (SUN) lent and borrowed.
	DelegatedOut tron.SUN
	DelegatedIn  tron.SUN
	// UnstakeSlots is the number of unstake operations still allowed.
	UnstakeSlots int64
	// TronPowerAvailable is the unused voting power.
	TronPowerAvailable int64
}

// Summary reads the account state and resource state in two reads and folds
// them into the numbers a staking UI or a top-up decision needs.
func (r *Resources) Summary(ctx context.Context) (*Summary, error) {
	st, err := (&Handle{cp: r.cp, owner: r.owner}).State(ctx)
	if err != nil {
		return nil, err
	}
	rs, err := tx.ResourceStateOf(ctx, r.cp, r.owner)
	if err != nil {
		return nil, err
	}
	slots, err := tx.UnfreezeSlotsOf(ctx, r.cp, r.owner)
	if err != nil {
		return nil, err
	}
	out := &Summary{
		StakedByResource:   make(map[tx.Resource]tron.SUN, 2),
		UnstakeSlots:       slots,
		TronPowerAvailable: rs.TronPowerAvailable(),
	}
	for _, s := range st.Stakes {
		out.StakedByResource[s.Resource] += s.Amount
	}
	for _, u := range st.Unstakes {
		out.UnstakePending += u.Amount
		if !u.ExpiresAt.IsZero() && !u.ExpiresAt.After(nowFunc()) {
			out.UnstakeWithdrawable += u.Amount
		}
	}
	// Withdrawable is also reported by the node evaluated at a timestamp;
	// prefer the node's number when it is larger (it accounts for unstakes
	// the account read may not have listed).
	if w, err := tx.WithdrawableUnfreezeOf(ctx, r.cp, r.owner); err == nil && w > out.UnstakeWithdrawable {
		out.UnstakeWithdrawable = w
	}
	out.DelegatedOut = st.DelegatedOutBandwidth + st.DelegatedOutEnergy
	out.DelegatedIn = st.DelegatedInBandwidth + st.DelegatedInEnergy
	return out, nil
}
