package tx

import (
	"context"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// Voting: casting TRON Power for Super Representatives and claiming the
// resulting rewards.
//
// TRON Power is granted by staking (1 TRX staked = 1 TP under the current
// Mainnet resource model) and is not delegatable — it stays with the account
// that staked. Staking alone earns nothing: voting is what produces rewards.
//
// Two facts shape this API:
//
//   - A vote transaction REPLACES the account's whole vote list. There is no
//     "add a vote" operation; omitting a witness withdraws the votes it held.
//     BuildVoteWitness is named for that whole-list replacement, not "Vote".
//   - Vote tallies are applied at the chain's maintenance period (6 hours on
//     Mainnet), not per transaction, and rewards accrue against the snapshot
//     taken at the start of each cycle.
//
// Unstaking reclaims TRON Power — idle (unvoted) power first, then
// proportionally from the voted witnesses — so an unstake silently reduces
// votes. Stake 2.0 does this proportionally, unlike Stake 1.0 which revoked
// every vote at once.

// Vote allocates Count units of TRON Power to Witness. Count is measured in
// TRON Power (TRX staked under the current resource model), not in TRX moved
// anywhere: voting never transfers funds.
type Vote struct {
	// Witness is the SR candidate the votes go to.
	Witness tron.Address
	// Count is the number of TRON Power units allocated; it must be positive.
	Count int64
}

// BuildVoteWitness builds a VoteWitnessContract. votes is the COMPLETE list
// the account will have after the transaction — any witness left out loses
// the votes it held. votes must be non-empty, and each witness may appear
// only once.
//
// The vote takes effect in the ranking at the next maintenance period
// (rpc.GetNextMaintenanceTime), not immediately.
func BuildVoteWitness(ctx context.Context, cp rpc.ConnProvider, voter tron.Address, votes []Vote) (*NativeTx, error) {
	const op = "tx.BuildVoteWitness"
	if err := validateAddress(op, "voter", voter); err != nil {
		return nil, err
	}
	if len(votes) == 0 {
		return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
			Hint: "a vote transaction must name at least one witness; a vote transaction replaces the whole vote list"}
	}
	seen := make(map[tron.Address]struct{}, len(votes))
	pbVotes := make([]*core.VoteWitnessContract_Vote, 0, len(votes))
	for i, v := range votes {
		if err := validateAddress(op, "votes["+itoa(int64(i))+"]", v.Witness); err != nil {
			return nil, err
		}
		if _, dup := seen[v.Witness]; dup {
			return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
				Hint: "witness " + v.Witness.String() + " appears more than once; merge the counts into a single entry"}
		}
		seen[v.Witness] = struct{}{}
		if v.Count <= 0 {
			return nil, &tron.Error{Code: tron.CodeTxInvalidArgument, Op: op,
				Hint: "vote count for " + v.Witness.String() + " must be positive; leave the witness out to withdraw its votes"}
		}
		pbVotes = append(pbVotes, &core.VoteWitnessContract_Vote{
			VoteAddress: v.Witness.Bytes(),
			VoteCount:   v.Count,
		})
	}
	req := &core.VoteWitnessContract{OwnerAddress: voter.Bytes(), Votes: pbVotes}
	ext, err := rpc.VoteWitnessAccount2(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// BuildWithdrawRewards builds a WithdrawBalanceContract, which moves the
// account's accrued voting rewards into its spendable balance. It takes no
// amount: the node withdraws everything accrued. It is named for what it
// does — claiming voting rewards — and never "withdraw unstaked TRX", which is
// BuildWithdrawExpireUnfreeze.
func BuildWithdrawRewards(ctx context.Context, cp rpc.ConnProvider, owner tron.Address) (*NativeTx, error) {
	const op = "tx.BuildWithdrawRewards"
	if err := validateAddress(op, "owner", owner); err != nil {
		return nil, err
	}
	req := &core.WithdrawBalanceContract{OwnerAddress: owner.Bytes()}
	ext, err := rpc.WithdrawBalance2(cp, ctx, req)
	if err != nil {
		return nil, err
	}
	return wrapBuildResult(func(b *baseTx) *NativeTx {
		return &NativeTx{baseTx: *b}
	}, ext, KindNative, cp, op)
}

// VotingRewardsOf reads the rewards the account has accrued but not yet
// claimed (GetRewardInfo), in SUN. Claiming them is BuildWithdrawRewards.
func VotingRewardsOf(ctx context.Context, cp rpc.ConnProvider, owner tron.Address) (tron.SUN, error) {
	const op = "tx.VotingRewardsOf"
	if err := validateAddress(op, "owner", owner); err != nil {
		return 0, err
	}
	msg, err := rpc.GetRewardInfo(cp, ctx, &api.BytesMessage{Value: owner.Bytes()})
	if err != nil {
		return 0, err
	}
	return tron.SUN(msg.GetNum()), nil
}
