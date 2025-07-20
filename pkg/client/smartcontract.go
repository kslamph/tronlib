package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) NewContractFromAddress(ctx context.Context, address *types.Address) (*types.Contract, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("new contract from address failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("new contract from address failed: %w", ErrContextCancelled)
	default:
	}

	// Validate input
	if address == nil {
		return nil, fmt.Errorf("failed to get contract: contract address is nil")
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for new contract from address: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	client := api.NewWalletClient(conn)
	result, err := client.GetContract(ctx, &api.BytesMessage{
		Value: address.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("failed to get contract: nil response")
	}

	if result.Abi == nil {
		return nil, fmt.Errorf("failed to get contract: contract ABI is nil")
	}

	return types.NewContractFromABI(result.Abi, address.String())
}

func (c *Client) TriggerConstantSmartContract(ctx context.Context, contract *types.Contract, ownerAddress *types.Address, data []byte) ([][]byte, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("trigger constant smart contract failed: %w", ErrClientClosed)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("trigger constant smart contract failed: %w", ErrContextCancelled)
	default:
	}

	// Validate inputs
	if contract == nil {
		return nil, fmt.Errorf("trigger constant smart contract failed: contract is nil")
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("trigger constant smart contract failed: owner address is nil")
	}
	if data == nil {
		return nil, fmt.Errorf("trigger constant smart contract failed: data is nil")
	}

	// Get connection from pool
	conn, err := c.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for trigger constant smart contract: %w", err)
	}

	// Ensure connection is returned to pool
	defer c.ReturnConnection(conn)

	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: contract.AddressBytes,
		Data:            data,
	}

	client := api.NewWalletClient(conn)
	result, err := client.TriggerConstantContract(ctx, trigger)

	if err != nil {
		return nil, fmt.Errorf("failed to trigger constant contract: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("failed to trigger constant contract: nil response")
	}

	return result.GetConstantResult(), nil
}
