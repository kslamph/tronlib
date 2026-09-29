package rpc

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
)

func TestTransactionWrappers(t *testing.T) {
	runWrapperGroup(t, []wrapperCase{
		{"CreateTransaction2", "CreateTransaction2", "create transaction2", func(cp ConnProvider, ctx context.Context) error {
			_, err := CreateTransaction2(cp, ctx, &core.TransferContract{})
			return err
		}},
		{"BroadcastTransaction", "BroadcastTransaction", "broadcast transaction", func(cp ConnProvider, ctx context.Context) error {
			_, err := BroadcastTransaction(cp, ctx, &core.Transaction{})
			return err
		}},
		{"GetTransactionById", "GetTransactionById", "get transaction by id", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetTransactionById(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetTransactionInfoById", "GetTransactionInfoById", "get transaction info by id", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetTransactionInfoById(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetTransactionCountByBlockNum", "GetTransactionCountByBlockNum", "get transaction count by block num", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetTransactionCountByBlockNum(cp, ctx, &api.NumberMessage{})
			return err
		}},
		{"GetTransactionSignWeight", "GetTransactionSignWeight", "get transaction sign weight", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetTransactionSignWeight(cp, ctx, &core.Transaction{})
			return err
		}},
		{"GetTransactionApprovedList", "GetTransactionApprovedList", "get transaction approved list", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetTransactionApprovedList(cp, ctx, &core.Transaction{})
			return err
		}},
		{"CreateCommonTransaction", "CreateCommonTransaction", "create common transaction", func(cp ConnProvider, ctx context.Context) error {
			_, err := CreateCommonTransaction(cp, ctx, &core.Transaction{})
			return err
		}},
		{"GetTransactionFromPending", "GetTransactionFromPending", "get transaction from pending", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetTransactionFromPending(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetTransactionListFromPending", "GetTransactionListFromPending", "get transaction list from pending", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetTransactionListFromPending(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetPendingSize", "GetPendingSize", "get pending size", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetPendingSize(cp, ctx, &api.EmptyMessage{})
			return err
		}},
	})
}
