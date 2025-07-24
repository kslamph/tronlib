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

	result, err := grpcGenericCallWrapper(c, ctx, "new contract from address", func(client api.WalletClient, ctx context.Context) (*core.SmartContract, error) {
		return client.GetContract(ctx, &api.BytesMessage{
			Value: address.Bytes(),
		})
	})

	if err != nil {
		return nil, err
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

	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: contract.AddressBytes,
		Data:            data,
	}

	result, err := grpcGenericCallWrapper(c, ctx, "trigger constant smart contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerConstantContract(ctx, trigger)
	})

	if err != nil {
		return nil, err
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

	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: contract.AddressBytes,
		Data:            data,
	}

	result, err := grpcGenericCallWrapper(c, ctx, "estimate energy", func(client api.WalletClient, ctx context.Context) (*api.EstimateEnergyMessage, error) {
		return client.EstimateEnergy(ctx, trigger)
	})

	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, fmt.Errorf("failed to estimate energy: nil response")
	}
	return result.EnergyRequired, nil
}

func (c *Client) TriggerSmartContract(ctx context.Context, owner *types.Address, contract *smartcontract.Contract, data []byte, callValue int64) (*api.TransactionExtention, error) {
	return c.CreateTriggerSmartContractTransaction(ctx, owner, types.MustNewAddress(contract.Address), data, callValue)
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
func (c *Client) CreateTriggerSmartContractTransaction(ctx context.Context, ownerAddress, contractAddress *types.Address, data []byte, callValue int64) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("CreateTriggerSmartContractTransaction failed: owner address is nil")
	}
	if contractAddress == nil {
		return nil, fmt.Errorf("CreateTriggerSmartContractTransaction failed: contract address is nil")
	}

	return c.grpcCallWrapper(ctx, "smart contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerContract(ctx, &core.TriggerSmartContract{
			OwnerAddress:    ownerAddress.Bytes(),
			ContractAddress: contractAddress.Bytes(),
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
func (c *Client) UpdateSetting(ctx context.Context, ownerAddr, contractAddr *types.Address, ConsumeUserResourcePercent int64) (*api.TransactionExtention, error) {
	if ownerAddr == nil {
		return nil, fmt.Errorf("UpdateSetting failed: owner address is nil")
	}
	if contractAddr == nil {
		return nil, fmt.Errorf("UpdateSetting failed: contract address is nil")
	}
	if ConsumeUserResourcePercent < 0 || ConsumeUserResourcePercent > 100 {
		return nil, fmt.Errorf("UpdateSetting failed: consume user resource percent must be between 0 and 100")
	}
	return c.grpcCallWrapper(ctx, "update setting", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateSetting(ctx, &core.UpdateSettingContract{
			OwnerAddress:               ownerAddr.Bytes(),
			ContractAddress:            contractAddr.Bytes(),
			ConsumeUserResourcePercent: ConsumeUserResourcePercent,
		})
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
func (c *Client) GetContractInfo(ctx context.Context, address *types.Address) (*core.SmartContractDataWrapper, error) {
	if address == nil {
		return nil, fmt.Errorf("GetContractInfo failed: address is empty")
	}

	return grpcGenericCallWrapper(c, ctx, "get contract info", func(client api.WalletClient, ctx context.Context) (*core.SmartContractDataWrapper, error) {
		return client.GetContractInfo(ctx, &api.BytesMessage{Value: address.Bytes()})
	})
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

// ExecuteSmartContract executes a smart contract method that modifies the blockchain state.
func (c *Client) ExecuteSmartContract(ctx context.Context, contract *smartcontract.Contract, owner *types.Address, callValue int64, method string, params ...interface{}) (*api.TransactionExtention, error) {
	if contract == nil {
		return nil, fmt.Errorf("execute smart contract failed: contract is nil")
	}
	if owner == nil {
		return nil, fmt.Errorf("execute smart contract failed: owner address is nil")
	}

	data, err := contract.EncodeInput(method, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input for method %s: %v", method, err)
	}

	return c.TriggerSmartContract(ctx, owner, contract, data, callValue)
}

// QuerySmartContract queries a smart contract method that does not modify the blockchain state (constant method).
func (c *Client) QuerySmartContract(ctx context.Context, contract *smartcontract.Contract, owner *types.Address, method string, params ...interface{}) (interface{}, error) {
	if contract == nil {
		return nil, fmt.Errorf("query smart contract failed: contract is nil")
	}
	if owner == nil {
		return nil, fmt.Errorf("query smart contract failed: owner address is nil")
	}

	data, err := contract.EncodeInput(method, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input for method %s: %v", method, err)
	}

	resultBytes, err := c.TriggerConstantSmartContract(ctx, contract, owner, data)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger constant smart contract for method %s: %v", method, err)
	}

	// DecodeResult expects the result of TriggerConstantSmartContract which is [][]byte
	decoded, err := contract.DecodeResult(method, resultBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result for method %s: %v", method, err)
	}

	return decoded, nil
}
