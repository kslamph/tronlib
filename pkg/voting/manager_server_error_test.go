package voting_test

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/internal/testutil"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/voting"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestVotingManager_ServerError verifies every RPC surfaces an error when the
// node returns a transport error (here: gRPC Unavailable).
func TestVotingManager_ServerError(t *testing.T) {
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	witness := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	nodeErr := status.Error(codes.Unavailable, "node unavailable")
	ctx := context.Background()

	tests := []struct {
		name string
		fake *voteServer
		call func(m *voting.Manager) error
	}{
		{"VoteWitnessAccount2", &voteServer{voteErr: nodeErr}, func(m *voting.Manager) error {
			_, err := m.VoteWitnessAccount2(ctx, owner, []voting.Vote{{WitnessAddress: witness, VoteCount: 1}})
			return err
		}},
		{"WithdrawBalance2", &voteServer{withdrawErr: nodeErr}, func(m *voting.Manager) error {
			_, err := m.WithdrawBalance2(ctx, owner)
			return err
		}},
		{"CreateWitness2", &voteServer{createWitnessErr: nodeErr}, func(m *voting.Manager) error {
			_, err := m.CreateWitness2(ctx, owner, "https://example.com")
			return err
		}},
		{"UpdateWitness2", &voteServer{updateWitnessErr: nodeErr}, func(m *voting.Manager) error {
			_, err := m.UpdateWitness2(ctx, owner, "https://example.com")
			return err
		}},
		{"ListWitnesses", &voteServer{listWitnessesErr: nodeErr}, func(m *voting.Manager) error {
			_, err := m.ListWitnesses(ctx)
			return err
		}},
		{"GetRewardInfo", &voteServer{rewardInfoErr: nodeErr}, func(m *voting.Manager) error {
			_, err := m.GetRewardInfo(ctx, owner)
			return err
		}},
		{"GetBrokerageInfo", &voteServer{brokerageInfoErr: nodeErr}, func(m *voting.Manager) error {
			_, err := m.GetBrokerageInfo(ctx, owner)
			return err
		}},
		{"UpdateBrokerage", &voteServer{updateBrokerageErr: nodeErr}, func(m *voting.Manager) error {
			_, err := m.UpdateBrokerage(ctx, owner, 5)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lis := testutil.NewBufconnServer(t, tt.fake)
			conn := testutil.DialBufconn(t, lis)
			m := voting.NewManager(testutil.NewMockConnProvider(conn))
			if err := tt.call(m); err == nil {
				t.Fatal("expected error from server")
			}
		})
	}
}
