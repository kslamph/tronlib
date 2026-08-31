package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// TestOtherWrappers exercises the 1:1 port of lowlevel/other.go (exchange,
// market and storage RPCs) through the bufconn fake (happy path + node-error
// path per wrapper).
func TestOtherWrappers(t *testing.T) {
	runWrapperGroup(t, []wrapperCase{
		{"ListExchanges", "ListExchanges", "list exchanges", func(cp ConnProvider, ctx context.Context) error {
			_, err := ListExchanges(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetPaginatedExchangeList", "GetPaginatedExchangeList", "get paginated exchange list", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetPaginatedExchangeList(cp, ctx, &api.PaginatedMessage{})
			return err
		}},
		{"GetExchangeById", "GetExchangeById", "get exchange by id", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetExchangeById(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"ExchangeCreate", "ExchangeCreate", "exchange create", func(cp ConnProvider, ctx context.Context) error {
			_, err := ExchangeCreate(cp, ctx, &core.ExchangeCreateContract{})
			return err
		}},
		{"ExchangeInject", "ExchangeInject", "exchange inject", func(cp ConnProvider, ctx context.Context) error {
			_, err := ExchangeInject(cp, ctx, &core.ExchangeInjectContract{})
			return err
		}},
		{"ExchangeWithdraw", "ExchangeWithdraw", "exchange withdraw", func(cp ConnProvider, ctx context.Context) error {
			_, err := ExchangeWithdraw(cp, ctx, &core.ExchangeWithdrawContract{})
			return err
		}},
		{"ExchangeTransaction", "ExchangeTransaction", "exchange transaction", func(cp ConnProvider, ctx context.Context) error {
			_, err := ExchangeTransaction(cp, ctx, &core.ExchangeTransactionContract{})
			return err
		}},
		{"MarketSellAsset", "MarketSellAsset", "market sell asset", func(cp ConnProvider, ctx context.Context) error {
			_, err := MarketSellAsset(cp, ctx, &core.MarketSellAssetContract{})
			return err
		}},
		{"MarketCancelOrder", "MarketCancelOrder", "market cancel order", func(cp ConnProvider, ctx context.Context) error {
			_, err := MarketCancelOrder(cp, ctx, &core.MarketCancelOrderContract{})
			return err
		}},
		{"GetMarketOrderById", "GetMarketOrderById", "get market order by id", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetMarketOrderById(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetMarketOrderByAccount", "GetMarketOrderByAccount", "get market order by account", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetMarketOrderByAccount(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetMarketPriceByPair", "GetMarketPriceByPair", "get market price by pair", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetMarketPriceByPair(cp, ctx, &core.MarketOrderPair{})
			return err
		}},
		{"GetMarketOrderListByPair", "GetMarketOrderListByPair", "get market order list by pair", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetMarketOrderListByPair(cp, ctx, &core.MarketOrderPair{})
			return err
		}},
		{"GetMarketPairList", "GetMarketPairList", "get market pair list", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetMarketPairList(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"BuyStorage", "BuyStorage", "buy storage", func(cp ConnProvider, ctx context.Context) error {
			_, err := BuyStorage(cp, ctx, &core.BuyStorageContract{})
			return err
		}},
		{"BuyStorageBytes", "BuyStorageBytes", "buy storage bytes", func(cp ConnProvider, ctx context.Context) error {
			_, err := BuyStorageBytes(cp, ctx, &core.BuyStorageBytesContract{})
			return err
		}},
		{"SellStorage", "SellStorage", "sell storage", func(cp ConnProvider, ctx context.Context) error {
			_, err := SellStorage(cp, ctx, &core.SellStorageContract{})
			return err
		}},
	})
}

// TestGetExchangeByIdHappyValue pins that the fake's canned response is
// returned as-is (not just non-nil) for a Call-style wrapper.
func TestGetExchangeByIdHappyValue(t *testing.T) {
	srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
		"GetExchangeById": func(ctx context.Context, in any) (any, error) {
			return &core.Exchange{ExchangeId: 42}, nil
		},
	}}
	c := newBufconnClient(t, srv, time.Second)
	got, err := GetExchangeById(c, context.Background(), &api.BytesMessage{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GetExchangeId() != 42 {
		t.Fatalf("got %+v, want the fake's exchange (id 42)", got)
	}
}
