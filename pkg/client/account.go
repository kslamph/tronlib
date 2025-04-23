package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) GetAccount(account *types.Address) (*core.Account, error) {

	acc, err := c.wallet.GetAccount(context.Background(), &core.Account{
		Address: account.Bytes(),
	})

	return acc, err
}

func (c *Client) GetAccountNet(account *types.Address) (*api.AccountNetMessage, error) {
	accNet, err := c.wallet.GetAccountNet(context.Background(), &core.Account{
		Address: account.Bytes(),
	})

	return accNet, err
}

func (c *Client) GetAccountResource(account *types.Address) (*api.AccountResourceMessage, error) {
	accRes, err := c.wallet.GetAccountResource(context.Background(), &core.Account{
		Address: account.Bytes(),
	})

	return accRes, err
}
