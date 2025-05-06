package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

//BLOCK

// GetBlockByNum returns a block by its number. it contains tron contract data
func (c *Client) GetBlockByNum(blockNumber int64) (*api.BlockExtention, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()

	var block *api.BlockExtention
	err := c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		block, err = c.wallet.GetBlockByNum2(ctx, &api.NumberMessage{
			Num: blockNumber,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get block by number: %v", err)
	}
	return block, nil
}

// GetBlockById returns a block by its ID. it contains smart contract log
func (c *Client) GetTransactionInfoByBlockNum(blockNumber int64) (*api.TransactionInfoList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()

	var txInfo *api.TransactionInfoList
	err := c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		txInfo, err = c.wallet.GetTransactionInfoByBlockNum(ctx, &api.NumberMessage{
			Num: blockNumber,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by block number: %v", err)
	}
	return txInfo, nil
}

func (c *Client) GetNowBlock() (*api.BlockExtention, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()

	var block *api.BlockExtention
	err := c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		block, err = c.wallet.GetNowBlock2(ctx, &api.EmptyMessage{})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %v", err)
	}
	return block, nil
}

// TRANSACTION
func (c *Client) WaitForTransactionInfo(txId string, timeout int) (*core.TransactionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	var tx *core.TransactionInfo
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %v", err)
	}
	for time.Now().Before(deadline) && tx.GetBlockNumber() == 0 {
		err := c.executeWithFailover(ctx, func(ctx context.Context) error {
			var err error

			tx, err = c.wallet.GetTransactionInfoById(ctx, &api.BytesMessage{
				Value: hashBytes,
			})
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("failed to wait for transaction info: %v", err)
		}
		time.Sleep(1 * time.Second)
	}
	return tx, nil
}

func (c *Client) GetTransactionById(txId string) (*core.Transaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()
	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %v", err)
	}
	var tx *core.Transaction
	err = c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		tx, err = c.wallet.GetTransactionById(ctx, &api.BytesMessage{
			Value: hashBytes,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction by ID: %v", err)
	}
	return tx, nil
}

func (c *Client) GetTransactionInfoById(txId string) (*core.TransactionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()
	hashBytes, err := hex.DecodeString(txId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %v", err)
	}
	var txInfo *core.TransactionInfo
	err = c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		txInfo, err = c.wallet.GetTransactionInfoById(ctx, &api.BytesMessage{
			Value: hashBytes,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info by ID: %v", err)
	}
	return txInfo, nil
}

// NETWORK PARAMETERS

func (c *Client) GetChainParameters() (*core.ChainParameters, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()

	var chainParams *core.ChainParameters
	err := c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		chainParams, err = c.wallet.GetChainParameters(ctx, &api.EmptyMessage{})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get chain parameters: %v", err)
	}
	return chainParams, nil
}
