// Package smartcontract provides high-level smart contract functionality
package smartcontract

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Manager provides high-level smart contract operations.
//
// The Manager allows you to deploy new smart contracts and perform administrative
// operations on existing contracts. For interacting with deployed contracts,
// use the Instance type which provides methods for calling contract functions.
type Manager struct{ conn lowlevel.ConnProvider }

// NewManager creates a new smart contract manager.
//
// The smart contract manager requires a connection provider (typically a *client.Client)
// to communicate with the TRON network.
//
// Example:
//
//	cli, err := client.NewClient("grpc://127.0.0.1:50051")
//	if err != nil {
//	    // handle error
//	}
//	defer cli.Close()
//
//	contractMgr := smartcontract.NewManager(cli)
func NewManager(conn lowlevel.ConnProvider) *Manager {
	return &Manager{conn: conn}
}

// Instance creates a bound contract instance for a deployed contract address.
// The ABI can be omitted to fetch from the network, or supplied as JSON string
// or *core.SmartContract_ABI.
func (m *Manager) Instance(contractAddress *types.Address, abi ...any) (*Instance, error) {
	return NewInstance(m.conn, contractAddress, abi...)
}

// Deploy deploys a smart contract with constructor parameters.
//
// This method creates a transaction to deploy a new smart contract to the TRON network.
// The transaction is not signed or broadcast - use client.SignAndBroadcast to complete
// the deployment.
//
// Parameters:
//   - ownerAddress: Address that will own the contract
//   - contractName: Human-readable name for the contract
//   - abi: Contract ABI (string, *core.SmartContract_ABI, or nil)
//   - bytecode: Compiled contract bytecode
//   - callValue: TRX amount to send with deployment (in SUN)
//   - consumeUserResourcePercent: Percentage of energy consumed by user (0-100)
//   - originEnergyLimit: Maximum energy the contract can consume
//   - constructorParams: Optional constructor parameters
//
// Example:
//
//	txExt, err := contractMgr.Deploy(ctx, owner, "MyContract", abiJSON, bytecode, 0, 100, 30000, param1, param2)
//	if err != nil {
//	    // handle error
//	}
//
//	// Sign and broadcast the transaction
//	opts := client.DefaultBroadcastOptions()
//	result, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
func (m *Manager) Deploy(ctx context.Context, ownerAddress *types.Address, contractName string, abi any, bytecode []byte, callValue, consumeUserResourcePercent, originEnergyLimit int64, constructorParams ...interface{}) (*api.TransactionExtention, error) {
	// Validate inputs
	if err := utils.ValidateContractName(contractName); err != nil {
		return nil, fmt.Errorf("%w: invalid contract name: %w", types.ErrInvalidParameter, err)
	}
	if len(bytecode) == 0 {
		return nil, fmt.Errorf("%w: bytecode cannot be empty", types.ErrInvalidParameter)
	}
	if callValue < 0 {
		return nil, fmt.Errorf("%w: call value cannot be negative", types.ErrInvalidParameter)
	}
	if err := utils.ValidateConsumeUserResourcePercent(consumeUserResourcePercent); err != nil {
		// preserve precise error if already sentinel-like, else wrap under invalid parameter
		return nil, fmt.Errorf("%w: %w", types.ErrInvalidParameter, err)
	}
	if originEnergyLimit < 0 {
		return nil, fmt.Errorf("%w: origin energy limit cannot be negative", types.ErrInvalidParameter)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	// Process ABI parameter similar to NewInstance
	var contractABI *core.SmartContract_ABI
	var err error
	if abi != nil {
		switch v := abi.(type) {
		case string:
			// Handle ABI JSON string
			if v == "" {
				return nil, fmt.Errorf("%w: empty ABI string", types.ErrInvalidParameter)
			}
			processor := utils.NewABIProcessor(nil)
			contractABI, err = processor.ParseABI(v)
			if err != nil {
				return nil, fmt.Errorf("%w: failed to parse ABI string: %w", types.ErrInvalidParameter, err)
			}
		case *core.SmartContract_ABI:
			// Handle parsed ABI object
			if v == nil {
				return nil, fmt.Errorf("%w: ABI cannot be nil", types.ErrInvalidParameter)
			}
			contractABI = v
		default:
			return nil, fmt.Errorf("%w: ABI must be string or *core.SmartContract_ABI, got %T", types.ErrInvalidParameter, v)
		}
	}

	// Encode constructor parameters if provided
	finalBytecode := bytecode
	if len(constructorParams) > 0 {
		if contractABI == nil {
			return nil, fmt.Errorf("%w: ABI is required when constructor parameters are provided", types.ErrInvalidParameter)
		}

		encodedParams, err := m.encodeConstructor(contractABI, constructorParams)
		if err != nil {
			// keep precise sentinel from encodeConstructor; it's already %w-wrapped
			return nil, fmt.Errorf("%w", err)
		}

		// Append encoded constructor parameters to bytecode
		finalBytecode = append(bytecode, encodedParams...)
	}

	// Create new contract
	newContract := &core.SmartContract{
		OriginAddress:              ownerAddress.Bytes(),
		ContractAddress:            nil, // Will be generated
		Abi:                        contractABI,
		Bytecode:                   finalBytecode,
		CallValue:                  callValue,
		ConsumeUserResourcePercent: consumeUserResourcePercent,
		Name:                       contractName,
		OriginEnergyLimit:          originEnergyLimit,
	}

	req := &core.CreateSmartContract{OwnerAddress: ownerAddress.Bytes(), NewContract: newContract, CallTokenValue: 0, TokenId: 0}
	return lowlevel.TxCall(m.conn, ctx, "deploy contract", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.DeployContract(ctx, req)
	})
}

// encodeConstructor encodes constructor parameters for contract deployment
func (m *Manager) encodeConstructor(abi *core.SmartContract_ABI, constructorParams []interface{}) ([]byte, error) {
	if abi == nil {
		if len(constructorParams) > 0 {
			return nil, fmt.Errorf("%w: constructor parameters provided but ABI is nil", types.ErrInvalidParameter)
		}
		// No ABI and no constructor params - return empty data
		return []byte{}, nil
	}

	// Parse ABI to get constructor parameter types
	processor := utils.NewABIProcessor(abi)

	// Get constructor parameter types
	constructorTypes, err := processor.GetConstructorTypes(abi)

	if err != nil {
		// If no constructor found, but parameters provided, that's an error
		if len(constructorParams) > 0 {
			return nil, fmt.Errorf("%w: constructor parameters provided but no constructor found in ABI", types.ErrInvalidParameter)
		}
		// No constructor and no parameters is valid
		return []byte{}, nil
	}

	// Validate parameter count
	if len(constructorParams) != len(constructorTypes) {
		return nil, fmt.Errorf("%w: constructor parameter count mismatch: expected %d, got %d", types.ErrInvalidParameter, len(constructorTypes), len(constructorParams))
	}

	// If no parameters, return empty bytes
	if len(constructorParams) == 0 {
		return []byte{}, nil
	}

	// Encode constructor parameters
	encoded, err := processor.EncodeMethod("", constructorTypes, constructorParams)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encode constructor parameters: %w", types.ErrInvalidParameter, err)
	}

	return encoded, nil
}

// EstimateEnergy estimates energy required for smart contract execution
// Use client.Simulate to know energy required for a transaction
func (m *Manager) EstimateEnergy(ctx context.Context, ownerAddress, contractAddress *types.Address, data []byte, callValue int64) (*api.EstimateEnergyMessage, error) {
	// Validate inputs
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: contract data cannot be empty", types.ErrInvalidParameter)
	}
	if callValue < 0 {
		return nil, fmt.Errorf("%w: call value cannot be negative", types.ErrInvalidParameter)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}
	if contractAddress == nil {
		return nil, fmt.Errorf("%w: invalid contract address: nil", types.ErrInvalidAddress)
	}

	if err := utils.ValidateContractData(data); err != nil {
		return nil, fmt.Errorf("%w: invalid contract data: %w", types.ErrInvalidParameter, err)
	}

	req := &core.TriggerSmartContract{OwnerAddress: ownerAddress.Bytes(), ContractAddress: contractAddress.Bytes(), Data: data, CallValue: callValue}
	return lowlevel.Call(m.conn, ctx, "estimate energy", func(cl api.WalletClient, ctx context.Context) (*api.EstimateEnergyMessage, error) {
		return cl.EstimateEnergy(ctx, req)
	})
}

// GetContract gets smart contract information
func (m *Manager) GetContract(ctx context.Context, contractAddress *types.Address) (*core.SmartContract, error) {
	if contractAddress == nil {
		return nil, fmt.Errorf("%w: invalid contract address: nil", types.ErrInvalidAddress)
	}

	req := &api.BytesMessage{Value: contractAddress.Bytes()}
	return lowlevel.Call(m.conn, ctx, "get contract", func(cl api.WalletClient, ctx context.Context) (*core.SmartContract, error) {
		return cl.GetContract(ctx, req)
	})
}

// GetContractInfo gets smart contract detailed information
func (m *Manager) GetContractInfo(ctx context.Context, contractAddress *types.Address) (*core.SmartContractDataWrapper, error) {
	if contractAddress == nil {
		return nil, fmt.Errorf("%w: invalid contract address: nil", types.ErrInvalidAddress)
	}

	req := &api.BytesMessage{Value: contractAddress.Bytes()}
	return lowlevel.Call(m.conn, ctx, "get contract info", func(cl api.WalletClient, ctx context.Context) (*core.SmartContractDataWrapper, error) {
		return cl.GetContractInfo(ctx, req)
	})
}

// UpdateSetting updates smart contract settings
func (m *Manager) UpdateSetting(ctx context.Context, ownerAddress, contractAddress *types.Address, consumeUserResourcePercent int64) (*api.TransactionExtention, error) {
	if consumeUserResourcePercent < 0 || consumeUserResourcePercent > 100 {
		return nil, fmt.Errorf("%w: consume user resource percent must be between 0 and 100", types.ErrInvalidParameter)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}
	if contractAddress == nil {
		return nil, fmt.Errorf("%w: invalid contract address: nil", types.ErrInvalidAddress)
	}

	req := &core.UpdateSettingContract{OwnerAddress: ownerAddress.Bytes(), ContractAddress: contractAddress.Bytes(), ConsumeUserResourcePercent: consumeUserResourcePercent}
	return lowlevel.TxCall(m.conn, ctx, "update setting", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UpdateSetting(ctx, req)
	})
}

// UpdateEnergyLimit updates smart contract energy limit
func (m *Manager) UpdateEnergyLimit(ctx context.Context, ownerAddress, contractAddress *types.Address, originEnergyLimit int64) (*api.TransactionExtention, error) {
	if originEnergyLimit < 0 {
		return nil, fmt.Errorf("%w: origin energy limit cannot be negative", types.ErrInvalidParameter)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}
	if contractAddress == nil {
		return nil, fmt.Errorf("%w: invalid contract address: nil", types.ErrInvalidAddress)
	}

	req := &core.UpdateEnergyLimitContract{OwnerAddress: ownerAddress.Bytes(), ContractAddress: contractAddress.Bytes(), OriginEnergyLimit: originEnergyLimit}
	return lowlevel.TxCall(m.conn, ctx, "update energy limit", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UpdateEnergyLimit(ctx, req)
	})
}

// ClearContractABI clears smart contract ABI
func (m *Manager) ClearContractABI(ctx context.Context, ownerAddress, contractAddress *types.Address) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}
	if contractAddress == nil {
		return nil, fmt.Errorf("%w: invalid contract address: nil", types.ErrInvalidAddress)
	}

	req := &core.ClearABIContract{OwnerAddress: ownerAddress.Bytes(), ContractAddress: contractAddress.Bytes()}
	return lowlevel.TxCall(m.conn, ctx, "clear contract abi", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ClearContractABI(ctx, req)
	})
}
