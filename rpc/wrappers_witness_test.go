package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
)

func TestWitnessWrappers(t *testing.T) {
	runWrapperGroup(t, []wrapperCase{
		{"VoteWitnessAccount2", "VoteWitnessAccount2", "vote witness account2", func(cp ConnProvider, ctx context.Context) error {
			_, err := VoteWitnessAccount2(cp, ctx, &core.VoteWitnessContract{})
			return err
		}},
		{"WithdrawBalance2", "WithdrawBalance2", "withdraw balance2", func(cp ConnProvider, ctx context.Context) error {
			_, err := WithdrawBalance2(cp, ctx, &core.WithdrawBalanceContract{})
			return err
		}},
		{"CreateWitness2", "CreateWitness2", "create witness2", func(cp ConnProvider, ctx context.Context) error {
			_, err := CreateWitness2(cp, ctx, &core.WitnessCreateContract{})
			return err
		}},
		{"UpdateWitness2", "UpdateWitness2", "update witness2", func(cp ConnProvider, ctx context.Context) error {
			_, err := UpdateWitness2(cp, ctx, &core.WitnessUpdateContract{})
			return err
		}},
		{"ListWitnesses", "ListWitnesses", "list witnesses", func(cp ConnProvider, ctx context.Context) error {
			_, err := ListWitnesses(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetRewardInfo", "GetRewardInfo", "get reward info", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetRewardInfo(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetBrokerageInfo", "GetBrokerageInfo", "get brokerage info", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetBrokerageInfo(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"UpdateBrokerage", "UpdateBrokerage", "update brokerage", func(cp ConnProvider, ctx context.Context) error {
			_, err := UpdateBrokerage(cp, ctx, &core.UpdateBrokerageContract{})
			return err
		}},
		{"GetPaginatedNowWitnessList", "GetPaginatedNowWitnessList", "get paginated now witness list", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetPaginatedNowWitnessList(cp, ctx, &api.PaginatedMessage{})
			return err
		}},
		{"GetPaginatedNowWitnessListSolidity", "GetPaginatedNowWitnessList", "get paginated now witness list", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetPaginatedNowWitnessListSolidity(cp, ctx, &api.PaginatedMessage{})
			return err
		}},
	})
}

// TestGetPaginatedNowWitnessListHappyValue pins the canned WitnessList return
// (Wallet variant), and TestGetPaginatedNowWitnessListSolidityHappyValue
// proves the request actually routes to the WalletSolidity service.
func TestGetPaginatedNowWitnessListHappyValue(t *testing.T) {
	want := &api.WitnessList{Witnesses: []*core.Witness{{Url: "w1"}}}
	srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
		"GetPaginatedNowWitnessList": func(ctx context.Context, in any) (any, error) { return want, nil },
	}}
	c := newBufconnClient(t, srv, time.Second)
	got, err := GetPaginatedNowWitnessList(c, context.Background(), &api.PaginatedMessage{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.GetWitnesses()) != 1 || got.GetWitnesses()[0].GetUrl() != "w1" {
		t.Fatalf("got %+v, want the fake's witness list", got)
	}
}

func TestGetPaginatedNowWitnessListSolidityHappyValue(t *testing.T) {
	want := &api.WitnessList{Witnesses: []*core.Witness{{Url: "solidity-w"}}}
	srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
		"GetPaginatedNowWitnessList": func(ctx context.Context, in any) (any, error) { return want, nil },
	}}
	c := newBufconnClient(t, srv, time.Second)
	got, err := GetPaginatedNowWitnessListSolidity(c, context.Background(), &api.PaginatedMessage{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.GetWitnesses()) != 1 || got.GetWitnesses()[0].GetUrl() != "solidity-w" {
		t.Fatalf("got %+v, want the fake's witness list", got)
	}
}
