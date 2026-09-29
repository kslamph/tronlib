package account

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

// Voting is the Super Representative voting and reward handle for one
// account.
//
// Voting consumes TRON Power, which staking grants (Resources().Stake) and
// unstaking reclaims. Two properties shape the API:
//
//   - SetVotes replaces the complete vote list. A witness omitted from the
//     call loses the votes it held, so the method is named for the whole-list
//     semantics rather than "Vote".
//   - Tallies are applied at the chain's maintenance period (6 hours on
//     Mainnet), not immediately, and rewards accrue against the snapshot
//     taken at the start of each cycle. NextTally reports when the next
//     update happens.
type Voting struct {
	cp    rpc.ConnProvider
	owner tron.Address
}

// Owner returns the account this handle is bound to.
func (v *Voting) Owner() tron.Address { return v.owner }

// SetVotes builds a vote transaction whose vote list REPLACES the account's
// current list (tx.BuildVoteWitness). votes must be non-empty and name each
// witness once; a witness left out loses the votes it held.
func (v *Voting) SetVotes(ctx context.Context, votes []tx.Vote) (*tx.NativeTx, error) {
	return tx.BuildVoteWitness(ctx, v.cp, v.owner, votes)
}

// Votes reads the account's current vote list.
func (v *Voting) Votes(ctx context.Context) ([]tx.Vote, error) {
	st, err := (&Handle{cp: v.cp, owner: v.owner}).State(ctx)
	if err != nil {
		return nil, err
	}
	return st.Votes, nil
}

// Rewards reads the accrued but unclaimed voting rewards in SUN
// (tx.VotingRewardsOf).
func (v *Voting) Rewards(ctx context.Context) (tron.SUN, error) {
	return tx.VotingRewardsOf(ctx, v.cp, v.owner)
}

// ClaimRewards builds the transaction that moves the accrued voting rewards
// into the spendable balance (tx.BuildWithdrawRewards). It is deliberately not
// called "Withdraw": withdrawing UNSTAKED TRX is
// Resources().WithdrawUnstaked, a different balance and a different cooldown.
func (v *Voting) ClaimRewards(ctx context.Context) (*tx.NativeTx, error) {
	return tx.BuildWithdrawRewards(ctx, v.cp, v.owner)
}

// NextTally returns when the next maintenance period starts, i.e. when votes
// submitted now take effect in the SR ranking (GetNextMaintenanceTime).
func (v *Voting) NextTally(ctx context.Context) (time.Time, error) {
	const op = "account.Voting.NextTally"
	msg, err := rpc.GetNextMaintenanceTime(v.cp, ctx, &api.EmptyMessage{})
	if err != nil {
		return time.Time{}, err
	}
	if msg.GetNum() <= 0 {
		return time.Time{}, &tron.Error{Code: tron.CodeContractBadMetadata, Op: op,
			Hint: "the node returned no next maintenance time"}
	}
	return time.UnixMilli(msg.GetNum()), nil
}
