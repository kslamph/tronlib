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

package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Exchange and market related gRPC calls

func (c *Client) ListExchanges(ctx context.Context, req *api.EmptyMessage) (*api.ExchangeList, error) {
	return grpcGenericCallWrapper(c, ctx, "list exchanges", func(cl api.WalletClient, ctx context.Context) (*api.ExchangeList, error) {
		return cl.ListExchanges(ctx, req)
	})
}

func (c *Client) GetPaginatedExchangeList(ctx context.Context, req *api.PaginatedMessage) (*api.ExchangeList, error) {
	return grpcGenericCallWrapper(c, ctx, "get paginated exchange list", func(cl api.WalletClient, ctx context.Context) (*api.ExchangeList, error) {
		return cl.GetPaginatedExchangeList(ctx, req)
	})
}

func (c *Client) GetExchangeById(ctx context.Context, req *api.BytesMessage) (*core.Exchange, error) {
	return grpcGenericCallWrapper(c, ctx, "get exchange by id", func(cl api.WalletClient, ctx context.Context) (*core.Exchange, error) {
		return cl.GetExchangeById(ctx, req)
	})
}

func (c *Client) ExchangeCreate(ctx context.Context, req *core.ExchangeCreateContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "exchange create", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ExchangeCreate(ctx, req)
	})
}

func (c *Client) ExchangeInject(ctx context.Context, req *core.ExchangeInjectContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "exchange inject", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ExchangeInject(ctx, req)
	})
}

func (c *Client) ExchangeWithdraw(ctx context.Context, req *core.ExchangeWithdrawContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "exchange withdraw", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ExchangeWithdraw(ctx, req)
	})
}

func (c *Client) ExchangeTransaction(ctx context.Context, req *core.ExchangeTransactionContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "exchange transaction", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ExchangeTransaction(ctx, req)
	})
}

func (c *Client) MarketSellAsset(ctx context.Context, req *core.MarketSellAssetContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "market sell asset", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.MarketSellAsset(ctx, req)
	})
}

func (c *Client) MarketCancelOrder(ctx context.Context, req *core.MarketCancelOrderContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "market cancel order", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.MarketCancelOrder(ctx, req)
	})
}

func (c *Client) GetMarketOrderById(ctx context.Context, req *api.BytesMessage) (*core.MarketOrder, error) {
	return grpcGenericCallWrapper(c, ctx, "get market order by id", func(cl api.WalletClient, ctx context.Context) (*core.MarketOrder, error) {
		return cl.GetMarketOrderById(ctx, req)
	})
}

func (c *Client) GetMarketOrderByAccount(ctx context.Context, req *api.BytesMessage) (*core.MarketOrderList, error) {
	return grpcGenericCallWrapper(c, ctx, "get market order by account", func(cl api.WalletClient, ctx context.Context) (*core.MarketOrderList, error) {
		return cl.GetMarketOrderByAccount(ctx, req)
	})
}
func (c *Client) GetMarketPriceByPair(ctx context.Context, req *core.MarketOrderPair) (*core.MarketPriceList, error) {
	return grpcGenericCallWrapper(c, ctx, "get market price by pair", func(cl api.WalletClient, ctx context.Context) (*core.MarketPriceList, error) {
		return cl.GetMarketPriceByPair(ctx, req)
	})
}

func (c *Client) GetMarketOrderListByPair(ctx context.Context, req *core.MarketOrderPair) (*core.MarketOrderList, error) {
	return grpcGenericCallWrapper(c, ctx, "get market order list by pair", func(cl api.WalletClient, ctx context.Context) (*core.MarketOrderList, error) {
		return cl.GetMarketOrderListByPair(ctx, req)
	})
}

func (c *Client) GetMarketPairList(ctx context.Context, req *api.EmptyMessage) (*core.MarketOrderPairList, error) {
	return grpcGenericCallWrapper(c, ctx, "get market pair list", func(cl api.WalletClient, ctx context.Context) (*core.MarketOrderPairList, error) {
		return cl.GetMarketPairList(ctx, req)
	})
}

// Storage functions

func (c *Client) BuyStorage(ctx context.Context, req *core.BuyStorageContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "buy storage", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.BuyStorage(ctx, req)
	})
}

func (c *Client) BuyStorageBytes(ctx context.Context, req *core.BuyStorageBytesContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "buy storage bytes", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.BuyStorageBytes(ctx, req)
	})
}

func (c *Client) SellStorage(ctx context.Context, req *core.SellStorageContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "sell storage", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.SellStorage(ctx, req)
	})
}
