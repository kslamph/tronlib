package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
)

func (c *Client) GetAccount(account *types.Address) (*core.Account, error) {
	var acc *core.Account

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (any, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetAccount(ctx, &core.Account{
			Address: account.Bytes(),
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account: %v", err)
	}

	acc = result.(*core.Account)
	return acc, nil
}

func (c *Client) GetAccountNet(account *types.Address) (*api.AccountNetMessage, error) {
	var accNet *api.AccountNetMessage

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetAccountNet(ctx, &core.Account{
			Address: account.Bytes(),
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account net: %v", err)
	}

	accNet = result.(*api.AccountNetMessage)
	return accNet, nil
}

func (c *Client) GetAccountResource(account *types.Address) (*api.AccountResourceMessage, error) {
	var accRes *api.AccountResourceMessage

	result, err := c.ExecuteWithClient(func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
		walletClient := api.NewWalletClient(conn)
		return walletClient.GetAccountResource(ctx, &core.Account{
			Address: account.Bytes(),
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get account resource: %v", err)
	}

	accRes = result.(*api.AccountResourceMessage)
	return accRes, nil
}
