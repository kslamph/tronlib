package client

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

func (c *Client) NewContractFromAddress(ctx context.Context, address *types.Address) (*smartcontract.Contract, error) {
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

	return smartcontract.NewContractFromABI(result.Abi, address.String())
}

func (c *Client) TriggerConstantSmartContract(ctx context.Context, contract *smartcontract.Contract, ownerAddress *types.Address, data []byte) ([][]byte, error) {
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

func (c *Client) EstimateEnergy(ctx context.Context, contract *smartcontract.Contract, ownerAddress *types.Address, data []byte) (int64, error) {
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

func (c *Client) TriggerSmartContract(ctx context.Context, owner *types.Address, contract *smartcontract.Contract, data []byte, callValue int64) (*api.TransactionExtention, error) {
	return c.CreateTriggerSmartContractTransaction(ctx, owner.Bytes(), contract.AddressBytes, data, callValue)
}

func (c *Client) DeployContract(ctx context.Context, owner *types.Address, bytecode []byte, abi string, name string, originEnergyLimit int64, consumeUserResourcePercent int64, constructorParams ...interface{}) (*api.TransactionExtention, error) {
	if len(bytecode) == 0 {
		return nil, fmt.Errorf("bytecode cannot be empty")
	}
	if len(abi) == 0 {
		return nil, fmt.Errorf("abi cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("contract name cannot be empty")
	}
	if originEnergyLimit <= 0 {
		return nil, fmt.Errorf("origin energy limit must be greater than 0")
	}
	if consumeUserResourcePercent < 0 || consumeUserResourcePercent > 100 {
		return nil, fmt.Errorf("consume user resource percent must be between 0 and 100")
	}
	if owner == nil {
		return nil, fmt.Errorf("owner address must be set before deploying contract")
	}

	finalBytecode := bytecode

	contractABI, err := smartcontract.DecodeABI(abi)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ABI: %v", err)
	}
	if len(constructorParams) > 0 {
		contract, err := smartcontract.NewContractFromABI(contractABI, owner.String())
		if err != nil {
			return nil, fmt.Errorf("failed to create contract instance: %v", err)
		}
		encodedParams, err := contract.EncodeInput("", constructorParams...)
		if err != nil {
			return nil, fmt.Errorf("failed to encode constructor parameters: %v", err)
		}
		finalBytecode = append(bytecode, encodedParams...)
	}

	createReq := &core.CreateSmartContract{
		OwnerAddress: owner.Bytes(),
		NewContract: &core.SmartContract{
			Name:                       name,
			Bytecode:                   finalBytecode,
			Abi:                        contractABI,
			OriginAddress:              owner.Bytes(),
			OriginEnergyLimit:          originEnergyLimit,
			ConsumeUserResourcePercent: consumeUserResourcePercent,
		},
	}

	return c.CreateDeployContractTransaction(ctx, createReq)
}

// CreateTriggerSmartContractTransaction creates a smart contract trigger transaction
func (c *Client) CreateTriggerSmartContractTransaction(ctx context.Context, ownerAddress, contractAddress []byte, data []byte, callValue int64) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "smart contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerContract(ctx, &core.TriggerSmartContract{
			OwnerAddress:    ownerAddress,
			ContractAddress: contractAddress,
			Data:            data,
			CallValue:       callValue,
		})
	})
}

// CreateDeployContractTransaction creates a deploy contract transaction
func (c *Client) CreateDeployContractTransaction(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error) {
	return c.grpcCallWrapper(ctx, "deploy contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.DeployContract(ctx, contract)
	})
}

// UpdateSetting updates smart contract settings
func (c *Client) UpdateSetting(ctx context.Context, contract *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	if contract == nil {
		return nil, fmt.Errorf("UpdateSetting failed: contract is nil")
	}
	return c.grpcCallWrapper(ctx, "update setting", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateSetting(ctx, contract)
	})
}

// UpdateEnergyLimit updates the energy limit of a smart contract
func (c *Client) UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	if contract == nil {
		return nil, fmt.Errorf("UpdateEnergyLimit failed: contract is nil")
	}
	return c.grpcCallWrapper(ctx, "update energy limit", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateEnergyLimit(ctx, contract)
	})
}

// GetContractInfo retrieves contract info by address
func (c *Client) GetContractInfo(ctx context.Context, address []byte) (*core.SmartContractDataWrapper, error) {
	if len(address) == 0 {
		return nil, fmt.Errorf("GetContractInfo failed: address is empty")
	}
	conn, err := c.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection for get contract info: %w", err)
	}
	defer c.pool.Put(conn)
	walletClient := api.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(ctx, c.GetTimeout())
	defer cancel()
	return walletClient.GetContractInfo(ctx, &api.BytesMessage{Value: address})
}

// ClearContractABI clears the ABI of a smart contract
func (c *Client) ClearContractABI(ctx context.Context, contract *core.ClearABIContract) (*api.TransactionExtention, error) {
	if contract == nil {
		return nil, fmt.Errorf("ClearContractABI failed: contract is nil")
	}
	return c.grpcCallWrapper(ctx, "clear contract abi", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ClearContractABI(ctx, contract)
	})
}
