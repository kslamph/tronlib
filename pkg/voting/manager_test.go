package voting_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/internal/testutil"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/voting"
	"google.golang.org/grpc/test/bufconn"
)

// voteServer is the voting fake: every RPC returns a minimal valid response
// unless the corresponding error field is set (for failure-path tests).
type voteServer struct {
	api.UnimplementedWalletServer

	voteErr            error
	withdrawErr        error
	createWitnessErr   error
	updateWitnessErr   error
	listWitnessesErr   error
	rewardInfoErr      error
	brokerageInfoErr   error
	updateBrokerageErr error
}

func okExt() *api.TransactionExtention {
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}
}

func (s *voteServer) VoteWitnessAccount2(ctx context.Context, in *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	return okExt(), s.voteErr
}
func (s *voteServer) WithdrawBalance2(ctx context.Context, in *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	return okExt(), s.withdrawErr
}
func (s *voteServer) CreateWitness2(ctx context.Context, in *core.WitnessCreateContract) (*api.TransactionExtention, error) {
	return okExt(), s.createWitnessErr
}
func (s *voteServer) UpdateWitness2(ctx context.Context, in *core.WitnessUpdateContract) (*api.TransactionExtention, error) {
	return okExt(), s.updateWitnessErr
}
func (s *voteServer) ListWitnesses(ctx context.Context, in *api.EmptyMessage) (*api.WitnessList, error) {
	if s.listWitnessesErr != nil {
		return nil, s.listWitnessesErr
	}
	return &api.WitnessList{}, nil
}
func (s *voteServer) GetRewardInfo(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
	if s.rewardInfoErr != nil {
		return nil, s.rewardInfoErr
	}
	return &api.NumberMessage{Num: 42}, nil
}
func (s *voteServer) GetBrokerageInfo(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
	if s.brokerageInfoErr != nil {
		return nil, s.brokerageInfoErr
	}
	return &api.NumberMessage{Num: 12}, nil
}
func (s *voteServer) UpdateBrokerage(ctx context.Context, in *core.UpdateBrokerageContract) (*api.TransactionExtention, error) {
	return okExt(), s.updateBrokerageErr
}

func newVoteBufServer(t *testing.T, impl api.WalletServer) *bufconn.Listener {
	t.Helper()
	return testutil.NewBufconnServer(t, impl)
}

func TestVotingManager_ValidationsAndCalls(t *testing.T) {
	lis := newVoteBufServer(t, &voteServer{})

	m := voting.NewManager(testutil.NewMockConnProvider(testutil.DialBufconn(t, lis)))
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	witness := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")

	ctx := context.Background()
	// Vote
	if _, err := m.VoteWitnessAccount2(ctx, owner, []voting.Vote{{WitnessAddress: witness, VoteCount: 1}}); err != nil {
		t.Fatalf("VoteWitnessAccount2: %v", err)
	}
	// Withdraw
	if _, err := m.WithdrawBalance2(ctx, owner); err != nil {
		t.Fatalf("WithdrawBalance2: %v", err)
	}
	// Create witness
	if _, err := m.CreateWitness2(ctx, owner, "https://example.com"); err != nil {
		t.Fatalf("CreateWitness2: %v", err)
	}
	// Update witness
	if _, err := m.UpdateWitness2(ctx, owner, "https://example.com/u"); err != nil {
		t.Fatalf("UpdateWitness2: %v", err)
	}
	// List
	if _, err := m.ListWitnesses(ctx); err != nil {
		t.Fatalf("ListWitnesses: %v", err)
	}
	// Get reward/brokerage
	if _, err := m.GetRewardInfo(ctx, owner); err != nil {
		t.Fatalf("GetRewardInfo: %v", err)
	}
	if _, err := m.GetBrokerageInfo(ctx, owner); err != nil {
		t.Fatalf("GetBrokerageInfo: %v", err)
	}
	// Update brokerage
	if _, err := m.UpdateBrokerage(ctx, owner, 20); err != nil {
		t.Fatalf("UpdateBrokerage: %v", err)
	}
}

func TestVotingManager_InputValidationErrors(t *testing.T) {
	m := voting.NewManager(&testutil.MockConnProvider{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Empty votes
	if _, err := m.VoteWitnessAccount2(ctx, types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o"), nil); err == nil {
		t.Fatalf("expected error for empty votes")
	}
	// Invalid vote count
	if _, err := m.VoteWitnessAccount2(ctx, types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o"), []voting.Vote{{WitnessAddress: nil, VoteCount: 0}}); err == nil {
		t.Fatalf("expected error for invalid vote count/address")
	}
	// Nil owner for WithdrawBalance2
	if _, err := m.WithdrawBalance2(ctx, nil); err == nil {
		t.Fatalf("expected error for nil owner")
	}
	// Empty URL for Create/Update
	if _, err := m.CreateWitness2(ctx, types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o"), ""); err == nil {
		t.Fatalf("expected error for empty url")
	}
	if _, err := m.UpdateWitness2(ctx, types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o"), ""); err == nil {
		t.Fatalf("expected error for empty update url")
	}
	// GetReward/Brokerage nil address
	if _, err := m.GetRewardInfo(ctx, nil); err == nil {
		t.Fatalf("expected error for nil addr")
	}
	if _, err := m.GetBrokerageInfo(ctx, nil); err == nil {
		t.Fatalf("expected error for nil addr")
	}
	// UpdateBrokerage invalid percent
	if _, err := m.UpdateBrokerage(ctx, types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o"), 101); err == nil {
		t.Fatalf("expected error for >100")
	}
}
