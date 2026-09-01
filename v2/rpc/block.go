package rpc

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Block and network related gRPC calls (1:1 port of lowlevel/block.go).

// GetNowBlock2 gets the latest block (v2 - preferred)
func GetNowBlock2(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.BlockExtention, error) {
	return Call(cp, ctx, "get now block2", func(client api.WalletClient, ctx context.Context) (*api.BlockExtention, error) {
		return client.GetNowBlock2(ctx, req)
	})
}

// GetBlockByNum2 gets block by number (v2 - preferred)
func GetBlockByNum2(cp ConnProvider, ctx context.Context, req *api.NumberMessage) (*api.BlockExtention, error) {
	return Call(cp, ctx, "get block by num2", func(client api.WalletClient, ctx context.Context) (*api.BlockExtention, error) {
		return client.GetBlockByNum2(ctx, req)
	})
}

// GetBlockById gets block by ID
func GetBlockById(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.Block, error) {
	return Call(cp, ctx, "get block by id", func(client api.WalletClient, ctx context.Context) (*core.Block, error) {
		return client.GetBlockById(ctx, req)
	})
}

// GetBlockByLimitNext2 gets blocks by limit and next (v2 - preferred)
func GetBlockByLimitNext2(cp ConnProvider, ctx context.Context, req *api.BlockLimit) (*api.BlockListExtention, error) {
	return Call(cp, ctx, "get block by limit next2", func(client api.WalletClient, ctx context.Context) (*api.BlockListExtention, error) {
		return client.GetBlockByLimitNext2(ctx, req)
	})
}

// GetBlockByLatestNum2 gets latest blocks by number (v2 - preferred)
func GetBlockByLatestNum2(cp ConnProvider, ctx context.Context, req *api.NumberMessage) (*api.BlockListExtention, error) {
	return Call(cp, ctx, "get block by latest num2", func(client api.WalletClient, ctx context.Context) (*api.BlockListExtention, error) {
		return client.GetBlockByLatestNum2(ctx, req)
	})
}

// GetTransactionInfoByBlockNum gets transaction info by block number
func GetTransactionInfoByBlockNum(cp ConnProvider, ctx context.Context, req *api.NumberMessage) (*api.TransactionInfoList, error) {
	return Call(cp, ctx, "get transaction info by block num", func(client api.WalletClient, ctx context.Context) (*api.TransactionInfoList, error) {
		return client.GetTransactionInfoByBlockNum(ctx, req)
	})
}

// Network information functions

// ListNodes lists the node's peers
func ListNodes(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.NodeList, error) {
	return Call(cp, ctx, "list nodes", func(client api.WalletClient, ctx context.Context) (*api.NodeList, error) {
		return client.ListNodes(ctx, req)
	})
}

// GetNodeInfo gets the node's information
func GetNodeInfo(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*core.NodeInfo, error) {
	return Call(cp, ctx, "get node info", func(client api.WalletClient, ctx context.Context) (*core.NodeInfo, error) {
		return client.GetNodeInfo(ctx, req)
	})
}

// GetChainParameters gets the chain parameters
func GetChainParameters(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*core.ChainParameters, error) {
	return Call(cp, ctx, "get chain parameters", func(client api.WalletClient, ctx context.Context) (*core.ChainParameters, error) {
		return client.GetChainParameters(ctx, req)
	})
}

// GetBandwidthPrices gets bandwidth price history
func GetBandwidthPrices(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	return Call(cp, ctx, "get bandwidth prices", func(client api.WalletClient, ctx context.Context) (*api.PricesResponseMessage, error) {
		return client.GetBandwidthPrices(ctx, req)
	})
}

// GetEnergyPrices gets energy price history
func GetEnergyPrices(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	return Call(cp, ctx, "get energy prices", func(client api.WalletClient, ctx context.Context) (*api.PricesResponseMessage, error) {
		return client.GetEnergyPrices(ctx, req)
	})
}

// GetMemoFee gets memo fee price history
func GetMemoFee(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	return Call(cp, ctx, "get memo fee", func(client api.WalletClient, ctx context.Context) (*api.PricesResponseMessage, error) {
		return client.GetMemoFee(ctx, req)
	})
}

// GetNextMaintenanceTime gets the next maintenance time
func GetNextMaintenanceTime(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.NumberMessage, error) {
	return Call(cp, ctx, "get next maintenance time", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.GetNextMaintenanceTime(ctx, req)
	})
}

// TotalTransaction gets the total transaction count
func TotalTransaction(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.NumberMessage, error) {
	return Call(cp, ctx, "total transaction", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.TotalTransaction(ctx, req)
	})
}

// GetBurnTrx gets the burned TRX count
func GetBurnTrx(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.NumberMessage, error) {
	return Call(cp, ctx, "get burn trx", func(client api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return client.GetBurnTrx(ctx, req)
	})
}

// GetBlock gets block by ID or number
func GetBlock(cp ConnProvider, ctx context.Context, req *api.BlockReq) (*api.BlockExtention, error) {
	return Call(cp, ctx, "get block", func(client api.WalletClient, ctx context.Context) (*api.BlockExtention, error) {
		return client.GetBlock(ctx, req)
	})
}

// ChainTip returns the latest known block number (GetNowBlock2's block
// header number). Facade-facing convenience over GetNowBlock2; added for
// Task 9 (spec §10 Client.ChainTip delegates here).
func ChainTip(cp ConnProvider, ctx context.Context) (uint64, error) {
	b, err := GetNowBlock2(cp, ctx, &api.EmptyMessage{})
	if err != nil {
		return 0, err
	}
	return uint64(b.GetBlockHeader().GetRawData().GetNumber()), nil
}
