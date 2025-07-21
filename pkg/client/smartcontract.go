package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) NewContractFromAddress(ctx context.Context, address *types.Address) (*types.Contract, error) {
	// Validate input
	if address == nil {
		return nil, fmt.Errorf("failed to get contract: contract address is nil")
	}

	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for new contract from address: %w", err)
	}
	defer c.pool.Put(conn)

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.GetContract(ctx, &api.BytesMessage{
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
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for trigger constant smart contract: %w", err)
	}
	defer c.pool.Put(conn)

	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: contract.AddressBytes,
		Data:            data,
	}

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.TriggerConstantContract(ctx, trigger)

	if err != nil {
		return nil, fmt.Errorf("failed to trigger constant contract: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("failed to trigger constant contract: nil response")
	}

	return result.GetConstantResult(), nil
}

func (c *Client) EstimateEnergy(ctx context.Context, contract *types.Contract, ownerAddress *types.Address, data []byte) (int64, error) {
	// Validate inputs
	if contract == nil {
		return 0, fmt.Errorf("estimate energy failed: contract is nil")
	}
	if ownerAddress == nil {
		return 0, fmt.Errorf("estimate energy failed: owner address is nil")
	}
	if data == nil {
		return 0, fmt.Errorf("estimate energy failed: data is nil")
	}

	// Get connection from pool
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get connection for estimate energy: %w", err)
	}
	defer c.pool.Put(conn)

	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: contract.AddressBytes,
		Data:            data,
	}

	walletClient := api.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	result, err := walletClient.EstimateEnergy(ctx, trigger)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate energy: %w", err)
	}
	if result == nil {
		return 0, fmt.Errorf("failed to estimate energy: nil response")
	}
	return result.EnergyRequired, nil
}
