// Package lowlevel provides 1:1 wrappers around WalletClient gRPC methods
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
)

// Block and network related gRPC calls

// GetNowBlock2 gets the latest block (v2 - preferred)
func GetNowBlock2(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.BlockExtention, error) {
	return grpcGenericCallWrapper(c, ctx, "get now block2", func(client api.WalletClient, ctx context.Context) (*api.BlockExtention, error) {
		return client.GetNowBlock2(ctx, req)
	})
}

// GetBlockByNum2 gets block by number (v2 - preferred)
func GetBlockByNum2(c *client.Client, ctx context.Context, req *api.NumberMessage) (*api.BlockExtention, error) {
	return grpcGenericCallWrapper(c, ctx, "get block by num2", func(client api.WalletClient, ctx context.Context) (*api.BlockExtention, error) {
		return client.GetBlockByNum2(ctx, req)
	})
}

// GetBlockById gets block by ID
func GetBlockById(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.Block, error) {
	return grpcGenericCallWrapper(c, ctx, "get block by id", func(client api.WalletClient, ctx context.Context) (*core.Block, error) {
		return client.GetBlockById(ctx, req)
	})
}

// GetBlockByLimitNext2 gets blocks by limit and next (v2 - preferred)
func GetBlockByLimitNext2(c *client.Client, ctx context.Context, req *api.BlockLimit) (*api.BlockListExtention, error) {
	return grpcGenericCallWrapper(c, ctx, "get block by limit next2", func(client api.WalletClient, ctx context.Context) (*api.BlockListExtention, error) {
		return client.GetBlockByLimitNext2(ctx, req)
	})
}

// GetBlockByLatestNum2 gets latest blocks by number (v2 - preferred)
func GetBlockByLatestNum2(c *client.Client, ctx context.Context, req *api.NumberMessage) (*api.BlockListExtention, error) {
	return grpcGenericCallWrapper(c, ctx, "get block by latest num2", func(client api.WalletClient, ctx context.Context) (*api.BlockListExtention, error) {
		return client.GetBlockByLatestNum2(ctx, req)
	})
}

// GetTransactionInfoByBlockNum gets transaction info by block number
func GetTransactionInfoByBlockNum(c *client.Client, ctx context.Context, req *api.NumberMessage) (*api.TransactionInfoList, error) {
	return grpcGenericCallWrapper(c, ctx, "get transaction info by block num", func(client api.WalletClient, ctx context.Context) (*api.TransactionInfoList, error) {
		return client.GetTransactionInfoByBlockNum(ctx, req)
	})
}

// Network information functions
func ListNodes(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.NodeList, error) {
	return grpcGenericCallWrapper(c, ctx, "list nodes", func(client api.WalletClient, ctx context.Context) (*api.NodeList, error) {
		return client.ListNodes(ctx, req)
	})
}

func GetNodeInfo(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*core.NodeInfo, error) {
	return grpcGenericCallWrapper(c, ctx, "get node info", func(client api.WalletClient, ctx context.Context) (*core.NodeInfo, error) {
		return client.GetNodeInfo(ctx, req)
	})
}

func GetChainParameters(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*core.ChainParameters, error) {
	return grpcGenericCallWrapper(c, ctx, "get chain parameters", func(client api.WalletClient, ctx context.Context) (*core.ChainParameters, error) {
		return client.GetChainParameters(ctx, req)
	})
}

func GetBandwidthPrices(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get bandwidth prices", func(client api.WalletClient, ctx context.Context) (*api.PricesResponseMessage, error) {
		return client.GetBandwidthPrices(ctx, req)
	})
}

func GetEnergyPrices(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get energy prices", func(client api.WalletClient, ctx context.Context) (*api.PricesResponseMessage, error) {
		return client.GetEnergyPrices(ctx, req)
	})
}

func GetMemoFee(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get memo fee", func(client api.WalletClient, ctx context.Context) (*api.PricesResponseMessage, error) {
		return client.GetMemoFee(ctx, req)
	})
}

func GetNextMaintenanceTime(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.NumberMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get next maintenance time", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.GetNextMaintenanceTime(ctx, req)
	})
}

func TotalTransaction(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.NumberMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "total transaction", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.TotalTransaction(ctx, req)
	})
}

func GetBurnTrx(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.NumberMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "get burn trx", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.GetBurnTrx(ctx, req)
	})
}