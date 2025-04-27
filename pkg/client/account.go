package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) GetAccount(account *types.Address) (*core.Account, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()
	var acc *core.Account
	err := c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		acc, err = c.wallet.GetAccount(ctx, &core.Account{
			Address: account.Bytes(),
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %v", err)
	}
	return acc, nil
}

func (c *Client) GetAccountNet(account *types.Address) (*api.AccountNetMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()

	var accNet *api.AccountNetMessage
	err := c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		accNet, err = c.wallet.GetAccountNet(ctx, &core.Account{
			Address: account.Bytes(),
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get account net: %v", err)
	}
	return accNet, nil
}

func (c *Client) GetAccountResource(account *types.Address) (*api.AccountResourceMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
	defer cancel()

	var accRes *api.AccountResourceMessage
	err := c.executeWithFailover(ctx, func(ctx context.Context) error {
		var err error
		accRes, err = c.wallet.GetAccountResource(ctx, &core.Account{
			Address: account.Bytes(),
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get account resource: %v", err)
	}
	return accRes, nil
}
