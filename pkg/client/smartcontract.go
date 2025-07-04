package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) NewContractFromAddress(ctx context.Context, address *types.Address) (*types.Contract, error) {
	if address == nil {
		return nil, fmt.Errorf("failed to get contract: contract address is nil")
	}

	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.GetContract(ctx, &api.BytesMessage{
		Value: address.Bytes(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %v", err)
	}

	if result.Abi == nil {
		return nil, fmt.Errorf("failed to get contract: contract ABI is nil")
	}

	return types.NewContractFromABI(result.Abi, address.String())
}

func (c *Client) TriggerConstantSmartContract(ctx context.Context, contract *types.Contract, ownerAddress *types.Address, data []byte) ([][]byte, error) {
	if err := c.ensureConnection(); err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: contract.AddressBytes,
		Data:            data,
	}

	client := api.NewWalletClient(c.conn)
	result, err := client.TriggerConstantContract(ctx, trigger)

	if err != nil {
		return nil, err
	}

	return result.GetConstantResult(), nil
}
