package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

//BLOCK

// GetBlockByNum returns a block by its number. it contains tron contract data
func (c *Client) GetBlockByNum(blockNumber int64) (*api.BlockExtention, error) {
	block, err := c.wallet.GetBlockByNum2(context.Background(), &api.NumberMessage{
		Num: blockNumber,
	})

	return block, err
}

// GetBlockById returns a block by its ID. it contains smart contract log
func (c *Client) GetTransactionInfoByBlockNum(blockNumber int64) (*api.TransactionInfoList, error) {
	txInfo, err := c.wallet.GetTransactionInfoByBlockNum(context.Background(), &api.NumberMessage{
		Num: blockNumber,
	})

	return txInfo, err
}

func (c *Client) GetNowBlock() (*api.BlockExtention, error) {
	block, err := c.wallet.GetNowBlock2(context.Background(), &api.EmptyMessage{})

	return block, err
}

//TRANSACTION

func (c *Client) GetTransactionById(txId string) (*core.Transaction, error) {
	tx, err := c.wallet.GetTransactionById(context.Background(), &api.BytesMessage{
		Value: []byte(txId),
	})

	return tx, err
}

func (c *Client) GetTransactionInfoById(txId string) (*core.TransactionInfo, error) {
	txInfo, err := c.wallet.GetTransactionInfoById(context.Background(), &api.BytesMessage{
		Value: []byte(txId),
	})

	return txInfo, err
}

// NETWORK PARAMETERS

func (c *Client) GetChainParameters() (*core.ChainParameters, error) {
	chainParams, err := c.wallet.GetChainParameters(context.Background(), &api.EmptyMessage{})

	return chainParams, err
}
