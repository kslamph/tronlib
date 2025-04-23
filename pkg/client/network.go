package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

func (c *Client) GetBlockByNum(blockNumber int64) (*core.Block, error) {
	block, err := c.wallet.GetBlockByNum(context.Background(), &api.NumberMessage{
		Num: blockNumber,
	})

	if err != nil {
		return nil, err
	}

	return block, nil
}

func (c *Client) GetTransactionInfoByBlockNum(blockNumber int64) (*api.TransactionInfoList, error) {
	txInfo, err := c.wallet.GetTransactionInfoByBlockNum(context.Background(), &api.NumberMessage{
		Num: blockNumber,
	})

	if err != nil {
		return nil, err
	}

	return txInfo, nil
}

func (c *Client) GetNowBlock() (*core.Block, error) {
	block, err := c.wallet.GetNowBlock(context.Background(), &api.EmptyMessage{})

	if err != nil {
		return nil, err
	}

	return block, nil
}
