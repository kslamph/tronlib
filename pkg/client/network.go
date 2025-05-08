package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"google.golang.org/grpc"
)

// GetBlockByNum returns a block by its number. it contains tron contract data
func (c *TronClient) GetBlockByNum(blockNumber int64) (*api.BlockExtention, error) {
	var block *api.BlockExtention

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetBlockByNum2(ctx, &api.NumberMessage{
			Num: blockNumber,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get block by number: %v", err)
	}

	block = result.(*api.BlockExtention)
	return block, nil
}

// GetTransactionInfoByBlockNum returns transaction info for a block.
func (c *TronClient) GetTransactionInfoByBlockNum(blockNumber int64) (*api.TransactionInfoList, error) {
	var txInfo *api.TransactionInfoList

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetTransactionInfoByBlockNum(ctx, &api.NumberMessage{
			Num: blockNumber,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by block number: %v", err)
	}

	txInfo = result.(*api.TransactionInfoList)
	return txInfo, nil
}

func (c *TronClient) GetNowBlock() (*api.BlockExtention, error) {
	var block *api.BlockExtention

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetNowBlock2(ctx, &api.EmptyMessage{})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %v", err)
	}

	block = result.(*api.BlockExtention)
	return block, nil
}

func (c *TronClient) WaitForTransactionInfo(txId string, timeoutSeconds int) (*core.TransactionInfo, error) {
	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %v", err)
	}

	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	var tx *core.TransactionInfo

	for time.Now().Before(deadline) {
		result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
			walletClient := api.NewWalletClient(conn)
			return walletClient.GetTransactionInfoById(ctx, &api.BytesMessage{
				Value: hashBytes,
			})
		})

		if err != nil {
			return nil, fmt.Errorf("failed to wait for transaction info: %v", err)
		}

		tx = result.(*core.TransactionInfo)
		if tx.GetBlockNumber() != 0 {
			return tx, nil
		}

		time.Sleep(time.Second)
	}

	return nil, fmt.Errorf("transaction not found after %d seconds", timeoutSeconds)
}

func (c *TronClient) GetTransactionById(txId string) (*core.Transaction, error) {
	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %v", err)
	}

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetTransactionById(ctx, &api.BytesMessage{
			Value: hashBytes,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction by ID: %v", err)
	}

	tx := result.(*core.Transaction)
	return tx, nil
}

func (c *TronClient) GetTransactionInfoById(txId string) (*core.TransactionInfo, error) {
	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %v", err)
	}

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetTransactionInfoById(ctx, &api.BytesMessage{
			Value: hashBytes,
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by ID: %v", err)
	}

	txInfo := result.(*core.TransactionInfo)
	return txInfo, nil
}

func (c *TronClient) GetChainParameters() (*core.ChainParameters, error) {
	var chainParams *core.ChainParameters

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetChainParameters(ctx, &api.EmptyMessage{})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get chain parameters: %v", err)
	}

	chainParams = result.(*core.ChainParameters)
	return chainParams, nil
}
