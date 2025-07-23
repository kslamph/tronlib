package client

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// GetBlockByNum returns a block by its number. it contains tron contract data
func (c *Client) GetBlockByNum(ctx context.Context, blockNumber int64) (*api.BlockExtention, error) {
	// Validate input
	if blockNumber < 0 {
		return nil, fmt.Errorf("get block by num failed: invalid block number %d", blockNumber)
	}

	return grpcGenericCallWrapper(c, ctx, "get block by num", func(client api.WalletClient, ctx context.Context) (*api.BlockExtention, error) {
		return client.GetBlockByNum2(ctx, &api.NumberMessage{
			Num: blockNumber,
		})
	})
}

// GetTransactionInfoByBlockNum returns transaction info for a block.
func (c *Client) GetTransactionInfoByBlockNum(ctx context.Context, blockNumber int64) (*api.TransactionInfoList, error) {
	// Validate input
	if blockNumber < 0 {
		return nil, fmt.Errorf("get transaction info by block num failed: invalid block number %d", blockNumber)
	}

	return grpcGenericCallWrapper(c, ctx, "get transaction info by block num", func(client api.WalletClient, ctx context.Context) (*api.TransactionInfoList, error) {
		return client.GetTransactionInfoByBlockNum(ctx, &api.NumberMessage{
			Num: blockNumber,
		})
	})
}

func (c *Client) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	return grpcGenericCallWrapper(c, ctx, "get now block", func(client api.WalletClient, ctx context.Context) (*api.BlockExtention, error) {
		return client.GetNowBlock2(ctx, &api.EmptyMessage{})
	})
}

// WaitForTransactionInfo waits for a transaction to be confirmed by checking its info.
// User can use *core.TransactionInfo.Result to check the transaction result.
// *core.TransactionInfo.Result is a int32
// 0 = success
// 1 = failed
// when error occurs, the transaction status should be considered as unknown
func (c *Client) WaitForTransactionInfo(ctx context.Context, txId string) (*core.TransactionInfo, error) {
	// Validate inputs
	if txId == "" {
		return nil, fmt.Errorf("wait for transaction info failed: transaction ID is empty")
	}

	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %w", err)
	}

	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for transaction info failed: %w", ctx.Err())
		default:
		}

		// Use generic wrapper for individual calls but handle retry logic here
		result, err := grpcGenericCallWrapper(c, ctx, "wait for transaction info", func(client api.WalletClient, ctx context.Context) (*core.TransactionInfo, error) {
			return client.GetTransactionInfoById(ctx, &api.BytesMessage{
				Value: hashBytes,
			})
		})

		if err != nil {
			// If the context was canceled during the gRPC call, return that error.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("wait for transaction info failed: %w", err)
			}
			// For other gRPC errors, you might want to log them and continue retrying
			// or return immediately depending on the desired behavior.
			// For this example, we'll continue retrying on other errors.
			log.Printf("failed to get transaction info, will retry: %v", err)
		}

		if result != nil && result.GetBlockNumber() != 0 {
			return result, nil
		}

		// Sleep before next attempt, but respect context cancellation
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for transaction info failed: %w", ctx.Err())
		}
	}
}

func (c *Client) GetTransactionById(ctx context.Context, txId []byte) (*core.Transaction, error) {
	// Validate input
	if len(txId) == 0 {
		return nil, fmt.Errorf("get transaction by id failed: transaction ID is empty")
	}

	return grpcGenericCallWrapper(c, ctx, "get transaction by id", func(client api.WalletClient, ctx context.Context) (*core.Transaction, error) {
		return client.GetTransactionById(ctx, &api.BytesMessage{
			Value: txId,
		})
	})
}

func (c *Client) GetTransactionInfoById(ctx context.Context, txId []byte) (*core.TransactionInfo, error) {
	// Validate input
	if len(txId) == 0 {
		return nil, fmt.Errorf("get transaction info by id failed: transaction ID is empty")
	}

	return grpcGenericCallWrapper(c, ctx, "get transaction info by id", func(client api.WalletClient, ctx context.Context) (*core.TransactionInfo, error) {
		return client.GetTransactionInfoById(ctx, &api.BytesMessage{
			Value: txId,
		})
	})
}

func (c *Client) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
	return grpcGenericCallWrapper(c, ctx, "get node info", func(client api.WalletClient, ctx context.Context) (*core.NodeInfo, error) {
		return client.GetNodeInfo(ctx, &api.EmptyMessage{})
	})
}

func (c *Client) ListNodes(ctx context.Context) (*api.NodeList, error) {
	return grpcGenericCallWrapper(c, ctx, "list nodes", func(client api.WalletClient, ctx context.Context) (*api.NodeList, error) {
		return client.ListNodes(ctx, &api.EmptyMessage{})
	})
}

func (c *Client) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	return grpcGenericCallWrapper(c, ctx, "get chain parameters", func(client api.WalletClient, ctx context.Context) (*core.ChainParameters, error) {
		return client.GetChainParameters(ctx, &api.EmptyMessage{})
	})
}

// GetBlockById retrieves a block by its ID
func (c *Client) GetBlockById(ctx context.Context, blockId []byte) (*core.Block, error) {
	if len(blockId) == 0 {
		return nil, fmt.Errorf("GetBlockById failed: blockId is empty")
	}

	return grpcGenericCallWrapper(c, ctx, "get block by id", func(client api.WalletClient, ctx context.Context) (*core.Block, error) {
		return client.GetBlockById(ctx, &api.BytesMessage{Value: blockId})
	})
}
