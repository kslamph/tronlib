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
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetBlockByNum2(ctx, &api.NumberMessage{
		Num: blockNumber,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get block by number: %v", err)
	}

	return result, nil
}

// GetTransactionInfoByBlockNum returns transaction info for a block.
func (c *Client) GetTransactionInfoByBlockNum(ctx context.Context, blockNumber int64) (*api.TransactionInfoList, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetTransactionInfoByBlockNum(ctx, &api.NumberMessage{
		Num: blockNumber,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by block number: %v", err)
	}

	return result, nil
}

func (c *Client) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetNowBlock2(ctx, &api.EmptyMessage{})

	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %v", err)
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
	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %v", err)
	}

	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	for time.Now().Before(deadline) {
		if err := c.ensureConnection(); err != nil {
			return nil, fmt.Errorf("connection error: %v", err)
		}

		// Use the provided ctx for cancellation, but still respect the timeoutSeconds loop
		client := api.NewWalletClient(c.conn)
		result, err := client.GetTransactionInfoById(ctx, &api.BytesMessage{
			Value: hashBytes,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to wait for transaction info: %v", err)
		}

		if result.GetBlockNumber() != 0 {
			return result, nil
		}

		time.Sleep(time.Second)
	}

	return nil, fmt.Errorf("transaction not found after %d seconds", timeoutSeconds)
}

func (c *Client) GetTransactionById(ctx context.Context, txId string) (*core.Transaction, error) {
	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %v", err)
	}

	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetTransactionById(ctx, &api.BytesMessage{
		Value: hashBytes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction by ID: %v", err)
	}

	return result, nil
}

func (c *Client) GetTransactionInfoById(ctx context.Context, txId string) (*core.TransactionInfo, error) {
	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %v", err)
	}

	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetTransactionInfoById(ctx, &api.BytesMessage{
		Value: hashBytes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by ID: %v", err)
	}

	return result, nil
}

func (c *Client) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetChainParameters(ctx, &api.EmptyMessage{})

	if err != nil {
		return nil, fmt.Errorf("failed to get chain parameters: %v", err)
	}

	return result, nil
}
