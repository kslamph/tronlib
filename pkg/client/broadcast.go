package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

func (c *Client) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer c.pool.Put(conn)

	// Create wallet client
	walletClient := api.NewWalletClient(conn)

	// Apply client timeout to context
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()

	result, err := walletClient.BroadcastTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	return result, nil
}
