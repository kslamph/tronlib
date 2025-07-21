package client

import (
	"context"
	"encoding/hex"
	"fmt"
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

	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get block by num: %w", err)
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.GetBlockByNum2(ctx, &api.NumberMessage{
		Num: blockNumber,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get block by number: %w", err)
	}

	return result, nil
}

// GetTransactionInfoByBlockNum returns transaction info for a block.
func (c *Client) GetTransactionInfoByBlockNum(ctx context.Context, blockNumber int64) (*api.TransactionInfoList, error) {
	// Validate input
	if blockNumber < 0 {
		return nil, fmt.Errorf("get transaction info by block num failed: invalid block number %d", blockNumber)
	}

	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get transaction info by block num: %w", err)
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.GetTransactionInfoByBlockNum(ctx, &api.NumberMessage{
		Num: blockNumber,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by block number: %w", err)
	}

	return result, nil
}

func (c *Client) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get now block: %w", err)
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.GetNowBlock2(ctx, &api.EmptyMessage{})

	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %w", err)
	}

	return result, nil
}

// WaitForTransactionInfo waits for a transaction to be confirmed by checking its info.
// User can use *core.TransactionInfo.Result to check the transaction result.
// *core.TransactionInfo.Result is a int32
// 0 = success
// 1 = failed
// when error occurs, the transaction status should be considered as unknown
func (c *Client) WaitForTransactionInfo(ctx context.Context, txId string, timeoutSeconds int) (*core.TransactionInfo, error) {
	// Validate inputs
	if txId == "" {
		return nil, fmt.Errorf("wait for transaction info failed: transaction ID is empty")
	}
	if timeoutSeconds <= 0 {
		return nil, fmt.Errorf("wait for transaction info failed: invalid timeout %d", timeoutSeconds)
	}

	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %w", err)
	}

	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	for time.Now().Before(deadline) {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for transaction info failed: %w", ErrContextCancelled)
		default:
		}

		// Get connection from pool
		conn, err := c.pool.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get connection for wait transaction info: %w", err)
		}

		// Create wallet client
		walletClient := api.NewWalletClient(conn)

		callCtx, cancel := context.WithTimeout(ctx, c.GetTimeout())
		result, err := walletClient.GetTransactionInfoById(callCtx, &api.BytesMessage{
			Value: hashBytes,
		})
		cancel()

		// Return connection to pool
		c.pool.Put(conn)

		if err != nil {
			return nil, fmt.Errorf("failed to wait for transaction info: %w", err)
		}

		if result.GetBlockNumber() != 0 {
			return result, nil
		}

		// Sleep before next attempt, but respect context cancellation
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for transaction info failed: %w", ErrContextCancelled)
		}
	}

	return nil, fmt.Errorf("transaction not found after %d seconds", timeoutSeconds)
}

func (c *Client) GetTransactionById(ctx context.Context, txId string) (*core.Transaction, error) {
	// Validate input
	if txId == "" {
		return nil, fmt.Errorf("get transaction by id failed: transaction ID is empty")
	}

	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %w", err)
	}

	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get transaction by id: %w", err)
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.GetTransactionById(ctx, &api.BytesMessage{
		Value: hashBytes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction by ID: %w", err)
	}

	return result, nil
}

func (c *Client) GetTransactionInfoById(ctx context.Context, txId string) (*core.TransactionInfo, error) {
	// Validate input
	if txId == "" {
		return nil, fmt.Errorf("get transaction info by id failed: transaction ID is empty")
	}

	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %w", err)
	}

	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get transaction info by id: %w", err)
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.GetTransactionInfoById(ctx, &api.BytesMessage{
		Value: hashBytes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by ID: %w", err)
	}

	return result, nil
}

func (c *Client) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	return walletClient.GetNodeInfo(ctx, &api.EmptyMessage{})
}

func (c *Client) ListNodes(ctx context.Context) (*api.NodeList, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	return walletClient.ListNodes(ctx, &api.EmptyMessage{})
}

func (c *Client) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	return walletClient.GetChainParameters(ctx, &api.EmptyMessage{})
}
