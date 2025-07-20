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
	if c.isClosed() {
		return nil, fmt.Errorf("get block by num failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get block by num failed: %w", ErrContextCancelled)
	default:
	}

	// Validate input
	if blockNumber < 0 {
		return nil, fmt.Errorf("get block by num failed: invalid block number %d", blockNumber)
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get block by num: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	client := api.NewWalletClient(conn)
	result, err := client.GetBlockByNum2(ctx, &api.NumberMessage{
		Num: blockNumber,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get block by number: %w", err)
	}

	return result, nil
}

// GetTransactionInfoByBlockNum returns transaction info for a block.
func (c *Client) GetTransactionInfoByBlockNum(ctx context.Context, blockNumber int64) (*api.TransactionInfoList, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("get transaction info by block num failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get transaction info by block num failed: %w", ErrContextCancelled)
	default:
	}

	// Validate input
	if blockNumber < 0 {
		return nil, fmt.Errorf("get transaction info by block num failed: invalid block number %d", blockNumber)
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get transaction info by block num: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	client := api.NewWalletClient(conn)
	result, err := client.GetTransactionInfoByBlockNum(ctx, &api.NumberMessage{
		Num: blockNumber,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by block number: %w", err)
	}

	return result, nil
}

func (c *Client) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("get now block failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get now block failed: %w", ErrContextCancelled)
	default:
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get now block: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	client := api.NewWalletClient(conn)
	result, err := client.GetNowBlock2(ctx, &api.EmptyMessage{})

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
	if c.isClosed() {
		return nil, fmt.Errorf("wait for transaction info failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for transaction info failed: %w", ErrContextCancelled)
	default:
	}

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
		conn, err := c.GetConnection(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get connection for wait transaction info: %w", err)
		}

		// Create wallet client
		client := api.NewWalletClient(conn)
		result, err := client.GetTransactionInfoById(ctx, &api.BytesMessage{
			Value: hashBytes,
		})

		// Return connection to pool
		c.ReturnConnection(conn)

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
	if c.isClosed() {
		return nil, fmt.Errorf("get transaction by id failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get transaction by id failed: %w", ErrContextCancelled)
	default:
	}

	// Validate input
	if txId == "" {
		return nil, fmt.Errorf("get transaction by id failed: transaction ID is empty")
	}

	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %w", err)
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get transaction by id: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	client := api.NewWalletClient(conn)
	result, err := client.GetTransactionById(ctx, &api.BytesMessage{
		Value: hashBytes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction by ID: %w", err)
	}

	return result, nil
}

func (c *Client) GetTransactionInfoById(ctx context.Context, txId string) (*core.TransactionInfo, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("get transaction info by id failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get transaction info by id failed: %w", ErrContextCancelled)
	default:
	}

	// Validate input
	if txId == "" {
		return nil, fmt.Errorf("get transaction info by id failed: transaction ID is empty")
	}

	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %w", err)
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get transaction info by id: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	client := api.NewWalletClient(conn)
	result, err := client.GetTransactionInfoById(ctx, &api.BytesMessage{
		Value: hashBytes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by ID: %w", err)
	}

	return result, nil
}

func (c *Client) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("get chain parameters failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get chain parameters failed: %w", ErrContextCancelled)
	default:
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get chain parameters: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	client := api.NewWalletClient(conn)
	result, err := client.GetChainParameters(ctx, &api.EmptyMessage{})

	if err != nil {
		return nil, fmt.Errorf("failed to get chain parameters: %w", err)
	}

	return result, nil
}
