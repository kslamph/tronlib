package rpc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/v2/tron"
)

// runWrapperGroup runs the happy path (default canned fake response comes
// back untouched) and the node-error path (fake handler returns a raw Go
// error -> rpc.method_failed with the wrapper's operation string in Op) for
// every wrapper in a group.
func runWrapperGroup(t *testing.T, cases []wrapperCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name+"/happy", func(t *testing.T) {
			c := newBufconnClient(t, &testWalletServer{}, time.Second)
			if err := tc.call(c, context.Background()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		t.Run(tc.name+"/node-error", func(t *testing.T) {
			srv := &testWalletServer{
				Handlers: map[string]func(ctx context.Context, in any) (any, error){
					tc.method: func(ctx context.Context, in any) (any, error) {
						return nil, errors.New("node exploded")
					},
				},
			}
			c := newBufconnClient(t, srv, time.Second)
			err := tc.call(c, context.Background())
			if !tron.HasCode(err, tron.CodeRPCMethodFailed) {
				t.Fatalf("err = %v, want rpc.method_failed", err)
			}
			var te *tron.Error
			if !errors.As(err, &te) {
				t.Fatalf("err = %T, want *tron.Error", err)
			}
			if te.Op != tc.op {
				t.Fatalf("Op = %q, want %q", te.Op, tc.op)
			}
			if te.Cause == nil || !strings.Contains(te.Cause.Error(), "node exploded") {
				t.Fatalf("Cause = %v, want an error carrying the fake's message", te.Cause)
			}
		})
	}
}

type wrapperCase struct {
	name   string
	method string // fake dispatch key
	op     string // expected operation string in Op
	call   func(cp ConnProvider, ctx context.Context) error
}

func TestBlockWrappers(t *testing.T) {
	runWrapperGroup(t, []wrapperCase{
		{"GetNowBlock2", "GetNowBlock2", "get now block2", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetNowBlock2(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetBlockByNum2", "GetBlockByNum2", "get block by num2", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetBlockByNum2(cp, ctx, &api.NumberMessage{})
			return err
		}},
		{"GetBlockById", "GetBlockById", "get block by id", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetBlockById(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetBlockByLimitNext2", "GetBlockByLimitNext2", "get block by limit next2", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetBlockByLimitNext2(cp, ctx, &api.BlockLimit{})
			return err
		}},
		{"GetBlockByLatestNum2", "GetBlockByLatestNum2", "get block by latest num2", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetBlockByLatestNum2(cp, ctx, &api.NumberMessage{})
			return err
		}},
		{"GetTransactionInfoByBlockNum", "GetTransactionInfoByBlockNum", "get transaction info by block num", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetTransactionInfoByBlockNum(cp, ctx, &api.NumberMessage{})
			return err
		}},
		{"ListNodes", "ListNodes", "list nodes", func(cp ConnProvider, ctx context.Context) error {
			_, err := ListNodes(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetNodeInfo", "GetNodeInfo", "get node info", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetNodeInfo(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetChainParameters", "GetChainParameters", "get chain parameters", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetChainParameters(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetBandwidthPrices", "GetBandwidthPrices", "get bandwidth prices", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetBandwidthPrices(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetEnergyPrices", "GetEnergyPrices", "get energy prices", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetEnergyPrices(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetMemoFee", "GetMemoFee", "get memo fee", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetMemoFee(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetNextMaintenanceTime", "GetNextMaintenanceTime", "get next maintenance time", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetNextMaintenanceTime(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"TotalTransaction", "TotalTransaction", "total transaction", func(cp ConnProvider, ctx context.Context) error {
			_, err := TotalTransaction(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetBurnTrx", "GetBurnTrx", "get burn trx", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetBurnTrx(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetBlock", "GetBlock", "get block", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetBlock(cp, ctx, &api.BlockReq{})
			return err
		}},
	})
}

// TestGetNowBlock2HappyValue pins that the fake's canned response is returned
// as-is (not just non-nil).
func TestGetNowBlock2HappyValue(t *testing.T) {
	want := &api.BlockExtention{Blockid: []byte("fake-block-id")}
	srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
		"GetNowBlock2": func(ctx context.Context, in any) (any, error) { return want, nil },
	}}
	c := newBufconnClient(t, srv, time.Second)
	got, err := GetNowBlock2(c, context.Background(), &api.EmptyMessage{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got.GetBlockid()) != "fake-block-id" {
		t.Fatalf("got %+v, want the fake's block (blockid fake-block-id)", got)
	}
}
