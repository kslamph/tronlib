// package client provides 1:1 wrappers around WalletClient gRPC methods
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
)

// Miscellaneous gRPC calls (Exchange, Market, Storage, etc.)

// Exchange functions
func ListExchanges(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.ExchangeList, error) {
	return grpcGenericCallWrapper(c, ctx, "list exchanges", func(client api.WalletClient, ctx context.Context) (*api.ExchangeList, error) {
		return client.ListExchanges(ctx, req)
	})
}

func GetPaginatedExchangeList(c *client.Client, ctx context.Context, req *api.PaginatedMessage) (*api.ExchangeList, error) {
	return grpcGenericCallWrapper(c, ctx, "get paginated exchange list", func(client api.WalletClient, ctx context.Context) (*api.ExchangeList, error) {
		return client.GetPaginatedExchangeList(ctx, req)
	})
}

func GetExchangeById(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.Exchange, error) {
	return grpcGenericCallWrapper(c, ctx, "get exchange by id", func(client api.WalletClient, ctx context.Context) (*core.Exchange, error) {
		return client.GetExchangeById(ctx, req)
	})
}

func ExchangeCreate(c *client.Client, ctx context.Context, req *core.ExchangeCreateContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "exchange create", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ExchangeCreate(ctx, req)
	})
}

func ExchangeInject(c *client.Client, ctx context.Context, req *core.ExchangeInjectContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "exchange inject", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ExchangeInject(ctx, req)
	})
}

func ExchangeWithdraw(c *client.Client, ctx context.Context, req *core.ExchangeWithdrawContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "exchange withdraw", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ExchangeWithdraw(ctx, req)
	})
}

func ExchangeTransaction(c *client.Client, ctx context.Context, req *core.ExchangeTransactionContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "exchange transaction", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ExchangeTransaction(ctx, req)
	})
}

// Market functions
func MarketSellAsset(c *client.Client, ctx context.Context, req *core.MarketSellAssetContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "market sell asset", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.MarketSellAsset(ctx, req)
	})
}

func MarketCancelOrder(c *client.Client, ctx context.Context, req *core.MarketCancelOrderContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "market cancel order", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.MarketCancelOrder(ctx, req)
	})
}

func GetMarketOrderById(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.MarketOrder, error) {
	return grpcGenericCallWrapper(c, ctx, "get market order by id", func(client api.WalletClient, ctx context.Context) (*core.MarketOrder, error) {
		return client.GetMarketOrderById(ctx, req)
	})
}

func GetMarketOrderByAccount(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.MarketOrderList, error) {
	return grpcGenericCallWrapper(c, ctx, "get market order by account", func(client api.WalletClient, ctx context.Context) (*core.MarketOrderList, error) {
		return client.GetMarketOrderByAccount(ctx, req)
	})
}

func GetMarketPriceByPair(c *client.Client, ctx context.Context, req *core.MarketOrderPair) (*core.MarketPriceList, error) {
	return grpcGenericCallWrapper(c, ctx, "get market price by pair", func(client api.WalletClient, ctx context.Context) (*core.MarketPriceList, error) {
		return client.GetMarketPriceByPair(ctx, req)
	})
}

func GetMarketOrderListByPair(c *client.Client, ctx context.Context, req *core.MarketOrderPair) (*core.MarketOrderList, error) {
	return grpcGenericCallWrapper(c, ctx, "get market order list by pair", func(client api.WalletClient, ctx context.Context) (*core.MarketOrderList, error) {
		return client.GetMarketOrderListByPair(ctx, req)
	})
}

func GetMarketPairList(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*core.MarketOrderPairList, error) {
	return grpcGenericCallWrapper(c, ctx, "get market pair list", func(client api.WalletClient, ctx context.Context) (*core.MarketOrderPairList, error) {
		return client.GetMarketPairList(ctx, req)
	})
}

// Storage functions
func BuyStorage(c *client.Client, ctx context.Context, req *core.BuyStorageContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "buy storage", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.BuyStorage(ctx, req)
	})
}

func BuyStorageBytes(c *client.Client, ctx context.Context, req *core.BuyStorageBytesContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "buy storage bytes", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.BuyStorageBytes(ctx, req)
	})
}

func SellStorage(c *client.Client, ctx context.Context, req *core.SellStorageContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "sell storage", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.SellStorage(ctx, req)
	})
}
