// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package voting_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/voting"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const voteBufSize = 1024 * 1024

type voteServer struct{ api.UnimplementedWalletServer }

func (s *voteServer) VoteWitnessAccount2(ctx context.Context, in *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}, nil
}
func (s *voteServer) WithdrawBalance2(ctx context.Context, in *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}, nil
}
func (s *voteServer) CreateWitness2(ctx context.Context, in *core.WitnessCreateContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}, nil
}
func (s *voteServer) UpdateWitness2(ctx context.Context, in *core.WitnessUpdateContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}, nil
}
func (s *voteServer) ListWitnesses(ctx context.Context, in *api.EmptyMessage) (*api.WitnessList, error) {
	return &api.WitnessList{}, nil
}
func (s *voteServer) GetRewardInfo(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
	return &api.NumberMessage{Num: 42}, nil
}
func (s *voteServer) GetBrokerageInfo(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
	return &api.NumberMessage{Num: 12}, nil
}
func (s *voteServer) UpdateBrokerage(ctx context.Context, in *core.UpdateBrokerageContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}, nil
}

func newVoteBufServer(t *testing.T, impl api.WalletServer) (*bufconn.Listener, *grpc.Server, func()) {
	t.Helper()
	lis := bufconn.Listen(voteBufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, impl)
	go func() { _ = srv.Serve(lis) }()
	cleanup := func() { _ = lis.Close(); srv.Stop() }
	return lis, srv, cleanup
}

func TestVotingManager_ValidationsAndCalls(t *testing.T) {
	lis, _, cleanup := newVoteBufServer(t, &voteServer{})
	t.Cleanup(cleanup)

	c, err := client.NewClientWithDialer("passthrough:///bufnet", func(ctx context.Context, s string) (net.Conn, error) { return lis.DialContext(ctx) }, client.WithTimeout(500*time.Millisecond), client.WithPool(1, 1))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer c.Close()

	m := voting.NewManager(c)
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
	m := voting.NewManager(&client.Client{})
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
