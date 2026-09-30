// Package account is the account-scoped facade: everything that acts on, or
// reads the state of, ONE account.
//
// The root Client is the chain handle — dial, broadcast, wait, read chain-wide
// data. This package is what you get from Client.Account(owner): a handle bound
// to a single address that owns the operations whose first argument would
// otherwise always be that address (transfers, deployment, staking,
// delegation, permissions, voting) and the reads that answer account-shaped
// questions.
//
// The separation is not cosmetic. TRON separates the ACCOUNT being operated on
// from the KEY that signs: in a multi-signature setup the signer's address is
// not the account's address, and a permission update changes which keys
// authorize the account. A handle that stored a private key, or that signed
// implicitly, would hide that distinction; this one never holds a key and never
// signs. Every state-changing method returns an unsigned transaction that the
// caller signs and broadcasts through the shared pipeline:
//
//	acct := cli.Account(owner)
//	req, err := acct.Resources().Stake(ctx, tronlib.Energy, tronlib.TRX(100))
//	signed, err := req.Sign(signer)      // signer may be a different address
//	rec, err := cli.Broadcast(ctx, signed)
//
// Naming follows what the chain does, not what is convenient: Unstake starts
// the cooldown and WithdrawUnstaked claims matured TRX, while ClaimRewards
// claims voting rewards. Those are three different flows and share no verb.
package account

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// Handle is the account-scoped entry point. It is a cheap value: New performs
// no I/O and the handle stores no key material, so it can be created per
// operation or kept for the life of the program.
type Handle struct {
	cp    rpc.ConnProvider
	owner tron.Address
}

// New returns a handle for owner on the client's connection. It performs no
// network I/O.
func New(cp rpc.ConnProvider, owner tron.Address) *Handle {
	return &Handle{cp: cp, owner: owner}
}

// Address returns the account this handle is bound to.
func (h *Handle) Address() tron.Address { return h.owner }

// State is a decoded account read: balances, stakes, pending unstakes, votes
// and the delegated totals, in SUN and canonical units. It is the answer to
// "what does this account look like right now"; Permissions().Current answers
// the permission half separately.
type State struct {
	// Address is the queried account.
	Address tron.Address
	// Exists reports whether the node knows the account at all. An address
	// with no account yet reports false with an otherwise zero state — this
	// matters because sending TRX to it creates the account and costs the
	// creation fee.
	Exists bool
	// Balance is the spendable TRX in SUN.
	Balance tron.SUN
	// Name is the account's optional on-chain name.
	Name string
	// IsWitness reports whether the account is an SR candidate, and
	// IsCommittee whether it is in the current active set.
	IsWitness   bool
	IsCommittee bool
	// CreatedAt is the account creation time; zero when unknown.
	CreatedAt time.Time
	// Allowance is an SR's remaining block-production allowance in SUN.
	Allowance tron.SUN
	// Votes is the account's current vote list (whole-list, not additive).
	Votes []tx.Vote
	// Stakes are the Stake 2.0 positions currently staked.
	Stakes []Stake
	// Unstakes are the Stake 2.0 unstake operations in their cooldown.
	Unstakes []Unstake
	// DelegatedOut is this account's staked TRX lent to other accounts.
	DelegatedOutBandwidth tron.SUN
	DelegatedOutEnergy    tron.SUN
	// DelegatedIn is other accounts' staked TRX lent to this account.
	DelegatedInBandwidth tron.SUN
	DelegatedInEnergy    tron.SUN
}

// Stake is one staked position: the TRX (SUN) staked for one resource.
type Stake struct {
	// Resource is the resource the stake produces.
	Resource tx.Resource
	// Amount is the staked TRX in SUN.
	Amount tron.SUN
}

// Unstake is one pending unstake operation. Its TRX is neither staked nor
// spendable until ExpiresAt, then it can be withdrawn (or re-staked with
// CancelUnstake).
type Unstake struct {
	// Resource is the resource the stake produced.
	Resource tx.Resource
	// Amount is the TRX in SUN in the cooldown.
	Amount tron.SUN
	// ExpiresAt is when the amount becomes withdrawable.
	ExpiresAt time.Time
}

// State reads and decodes the account (GetAccount).
func (h *Handle) State(ctx context.Context) (*State, error) {
	const op = "account.State"
	acc, err := rpc.GetAccount(h.cp, ctx, &core.Account{Address: h.owner.Bytes()})
	if err != nil {
		return nil, err
	}
	st := &State{
		Address:     h.owner,
		Exists:      len(acc.GetAddress()) > 0,
		Balance:     tron.SUN(acc.GetBalance()),
		Name:        string(acc.GetAccountName()),
		IsWitness:   acc.GetIsWitness(),
		IsCommittee: acc.GetIsCommittee(),
		Allowance:   tron.SUN(acc.GetAllowance()),
	}
	if ct := acc.GetCreateTime(); ct > 0 {
		st.CreatedAt = time.UnixMilli(ct)
	}
	for _, v := range acc.GetVotes() {
		addr, err := tron.AddressFromBytes(v.GetVoteAddress())
		if err != nil {
			return nil, &tron.Error{Code: tron.CodeAddressInvalid, Op: op, Cause: err,
				Hint: "the node returned a vote whose witness address is not a 0x41-prefixed 21-byte value"}
		}
		st.Votes = append(st.Votes, tx.Vote{Witness: addr, Count: v.GetVoteCount()})
	}
	for _, f := range acc.GetFrozenV2() {
		st.Stakes = append(st.Stakes, Stake{Resource: resourceFromProto(f.GetType()), Amount: tron.SUN(f.GetAmount())})
	}
	for _, u := range acc.GetUnfrozenV2() {
		st.Unstakes = append(st.Unstakes, Unstake{
			Resource:  resourceFromProto(u.GetType()),
			Amount:    tron.SUN(u.GetUnfreezeAmount()),
			ExpiresAt: millisOrZero(u.GetUnfreezeExpireTime()),
		})
	}
	ar := acc.GetAccountResource()
	st.DelegatedOutBandwidth = tron.SUN(acc.GetDelegatedFrozenV2BalanceForBandwidth())
	st.DelegatedInBandwidth = tron.SUN(acc.GetAcquiredDelegatedFrozenV2BalanceForBandwidth())
	st.DelegatedOutEnergy = tron.SUN(ar.GetDelegatedFrozenV2BalanceForEnergy())
	st.DelegatedInEnergy = tron.SUN(ar.GetAcquiredDelegatedFrozenV2BalanceForEnergy())
	return st, nil
}

// Balance returns the spendable TRX in SUN. An account the node does not know
// reports zero with no error (the TransferTRX recipient-creation case).
func (h *Handle) Balance(ctx context.Context) (tron.SUN, error) {
	acc, err := rpc.GetAccount(h.cp, ctx, &core.Account{Address: h.owner.Bytes()})
	if err != nil {
		return 0, err
	}
	return tron.SUN(acc.GetBalance()), nil
}

// TransferTRX builds a TRX transfer from this account (NativeTx). The node
// fills raw_data; sign the result and broadcast it. Sending to an address that
// has no account yet creates it and costs the creation fee — read the
// recipient's State first, or let TotalCost account for it.
func (h *Handle) TransferTRX(ctx context.Context, to tron.Address, amt tron.SUN) (*tx.NativeTx, error) {
	return tx.BuildTransfer(ctx, h.cp, h.owner, to, amt)
}

// TransferTRC10 builds a TRC-10 asset transfer (AssetTx) of qty raw units of
// assetName. TRC-10 is TRON's legacy asset system; TRC-20 (USDT and friends)
// goes through the token handle: cli.Token(ctx, addr).Transfer(...). qty is
// raw asset units — the SDK does not read the asset's issuance precision,
// so scale whole-token input before calling.
func (h *Handle) TransferTRC10(ctx context.Context, to tron.Address, assetName string, qty int64) (*tx.AssetTx, error) {
	return tx.BuildAssetTransfer(ctx, h.cp, h.owner, to, assetName, qty)
}

// Deploy builds a CreateSmartContract transaction owned by this account.
func (h *Handle) Deploy(ctx context.Context, p tx.DeployParams) (*tx.DeployTx, error) {
	return tx.BuildDeploy(ctx, h.cp, h.owner, p)
}

// CostPreview predicts the cost of a contract call for this account,
// combining the simulated energy, the account's staked energy and
// bandwidth, and both governance unit prices (tx.PreviewCost): energy to
// burn, bandwidth to burn on a single-signature estimate of the broadcast
// bytes, and TotalFloor — the all-in floor with no governance fees.
func (h *Handle) CostPreview(ctx context.Context, t *tx.ContractTx) (*tx.CostPreview, error) {
	return tx.PreviewCost(ctx, h.cp, t, h.owner)
}

// TotalCost predicts the all-in cost of broadcasting the fully-signed t for
// this account: energy, bandwidth, account-creation and the governance fees a
// multi-signature or permission-update transaction carries (tx.TotalCostOf).
// Call it after Sign — the bandwidth half is measured on the broadcast bytes.
func (h *Handle) TotalCost(ctx context.Context, t tx.Tx) (*tx.TotalCost, error) {
	return tx.TotalCostOf(ctx, h.cp, t, h.owner)
}

// Resources returns the staking and delegation handle for this account.
func (h *Handle) Resources() *Resources { return &Resources{cp: h.cp, owner: h.owner} }

// Permissions returns the multi-signature permission handle for this account.
func (h *Handle) Permissions() *Permissions { return &Permissions{cp: h.cp, owner: h.owner} }

// Voting returns the SR voting and reward handle for this account.
func (h *Handle) Voting() *Voting { return &Voting{cp: h.cp, owner: h.owner} }

// nowFunc is the clock used for the local time comparisons this package makes
// (matured unstakes). It is a variable so tests can pin it; production code
// never overrides it.
var nowFunc = time.Now

// resourceFromProto maps the wire resource enum onto the curated one. The wire
// also carries TRON_POWER (2), which is only meaningful under the new resource
// model; it maps to the zero value here so it cannot be mistaken for Energy.
func resourceFromProto(r core.ResourceCode) tx.Resource {
	if r == core.ResourceCode_ENERGY {
		return tx.ResourceEnergy
	}
	if r == core.ResourceCode_BANDWIDTH {
		return tx.ResourceBandwidth
	}
	return tx.Resource(-1)
}

func millisOrZero(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}
