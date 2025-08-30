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

package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Exchange and market related gRPC calls

func ListExchanges(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.ExchangeList, error) {
	return Call(cp, ctx, "list exchanges", func(cl api.WalletClient, ctx context.Context) (*api.ExchangeList, error) {
		return cl.ListExchanges(ctx, req)
	})
}

func GetPaginatedExchangeList(cp ConnProvider, ctx context.Context, req *api.PaginatedMessage) (*api.ExchangeList, error) {
	return Call(cp, ctx, "get paginated exchange list", func(cl api.WalletClient, ctx context.Context) (*api.ExchangeList, error) {
		return cl.GetPaginatedExchangeList(ctx, req)
	})
}

func GetExchangeById(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.Exchange, error) {
	return Call(cp, ctx, "get exchange by id", func(cl api.WalletClient, ctx context.Context) (*core.Exchange, error) {
		return cl.GetExchangeById(ctx, req)
	})
}

func ExchangeCreate(cp ConnProvider, ctx context.Context, req *core.ExchangeCreateContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "exchange create", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ExchangeCreate(ctx, req)
	})
}

func ExchangeInject(cp ConnProvider, ctx context.Context, req *core.ExchangeInjectContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "exchange inject", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ExchangeInject(ctx, req)
	})
}

func ExchangeWithdraw(cp ConnProvider, ctx context.Context, req *core.ExchangeWithdrawContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "exchange withdraw", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ExchangeWithdraw(ctx, req)
	})
}

func ExchangeTransaction(cp ConnProvider, ctx context.Context, req *core.ExchangeTransactionContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "exchange transaction", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ExchangeTransaction(ctx, req)
	})
}

func MarketSellAsset(cp ConnProvider, ctx context.Context, req *core.MarketSellAssetContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "market sell asset", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.MarketSellAsset(ctx, req)
	})
}

func MarketCancelOrder(cp ConnProvider, ctx context.Context, req *core.MarketCancelOrderContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "market cancel order", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.MarketCancelOrder(ctx, req)
	})
}

func GetMarketOrderById(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.MarketOrder, error) {
	return Call(cp, ctx, "get market order by id", func(cl api.WalletClient, ctx context.Context) (*core.MarketOrder, error) {
		return cl.GetMarketOrderById(ctx, req)
	})
}

func GetMarketOrderByAccount(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.MarketOrderList, error) {
	return Call(cp, ctx, "get market order by account", func(cl api.WalletClient, ctx context.Context) (*core.MarketOrderList, error) {
		return cl.GetMarketOrderByAccount(ctx, req)
	})
}

func GetMarketPriceByPair(cp ConnProvider, ctx context.Context, req *core.MarketOrderPair) (*core.MarketPriceList, error) {
	return Call(cp, ctx, "get market price by pair", func(cl api.WalletClient, ctx context.Context) (*core.MarketPriceList, error) {
		return cl.GetMarketPriceByPair(ctx, req)
	})
}

func GetMarketOrderListByPair(cp ConnProvider, ctx context.Context, req *core.MarketOrderPair) (*core.MarketOrderList, error) {
	return Call(cp, ctx, "get market order list by pair", func(cl api.WalletClient, ctx context.Context) (*core.MarketOrderList, error) {
		return cl.GetMarketOrderListByPair(ctx, req)
	})
}

func GetMarketPairList(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*core.MarketOrderPairList, error) {
	return Call(cp, ctx, "get market pair list", func(cl api.WalletClient, ctx context.Context) (*core.MarketOrderPairList, error) {
		return cl.GetMarketPairList(ctx, req)
	})
}

// Storage functions

func BuyStorage(cp ConnProvider, ctx context.Context, req *core.BuyStorageContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "buy storage", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.BuyStorage(ctx, req)
	})
}

func BuyStorageBytes(cp ConnProvider, ctx context.Context, req *core.BuyStorageBytesContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "buy storage bytes", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.BuyStorageBytes(ctx, req)
	})
}

func SellStorage(cp ConnProvider, ctx context.Context, req *core.SellStorageContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "sell storage", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.SellStorage(ctx, req)
	})
}
