// Package smartcontract provides high-level smart contract functionality
package smartcontract

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Manager provides high-level smart contract operations
type Manager struct {
	client *client.Client
}

// NewManager creates a new smart contract manager
func NewManager(client *client.Client) *Manager {
	return &Manager{
		client: client,
	}
}

 // DeployContract deploys a smart contract with constructor parameters
 // abi: ABI can be:
 //   - string: ABI JSON string
 //   - *core.SmartContract_ABI: Parsed ABI object
 //   - nil: No ABI provided
 func (m *Manager) DeployContract(ctx context.Context, ownerAddress *types.Address, contractName string, abi any, bytecode []byte, callValue, consumeUserResourcePercent, originEnergyLimit int64, constructorParams ...interface{}) (*api.TransactionExtention, error) {
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

 	// Process ABI parameter similar to NewContract
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

 	req := &core.CreateSmartContract{
 		OwnerAddress:   ownerAddress.Bytes(),
 		NewContract:    newContract,
 		CallTokenValue: 0,
 		TokenId:        0,
 	}

 	return m.client.DeployContract(ctx, req)
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

	req := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: contractAddress.Bytes(),
		Data:            data,
		CallValue:       callValue,
	}

	return m.client.EstimateEnergy(ctx, req)
}

// GetContract gets smart contract information
func (m *Manager) GetContract(ctx context.Context, contractAddress *types.Address) (*core.SmartContract, error) {
	if contractAddress == nil {
		return nil, fmt.Errorf("%w: invalid contract address: nil", types.ErrInvalidAddress)
	}

	req := &api.BytesMessage{
		Value: contractAddress.Bytes(),
	}

	return m.client.GetContract(ctx, req)
}

// GetContractInfo gets smart contract detailed information
func (m *Manager) GetContractInfo(ctx context.Context, contractAddress *types.Address) (*core.SmartContractDataWrapper, error) {
	if contractAddress == nil {
		return nil, fmt.Errorf("%w: invalid contract address: nil", types.ErrInvalidAddress)
	}

	req := &api.BytesMessage{
		Value: contractAddress.Bytes(),
	}

	return m.client.GetContractInfo(ctx, req)
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

	req := &core.UpdateSettingContract{
		OwnerAddress:               ownerAddress.Bytes(),
		ContractAddress:            contractAddress.Bytes(),
		ConsumeUserResourcePercent: consumeUserResourcePercent,
	}

	return m.client.UpdateSetting(ctx, req)
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

	req := &core.UpdateEnergyLimitContract{
		OwnerAddress:      ownerAddress.Bytes(),
		ContractAddress:   contractAddress.Bytes(),
		OriginEnergyLimit: originEnergyLimit,
	}

	return m.client.UpdateEnergyLimit(ctx, req)
}

// ClearContractABI clears smart contract ABI
func (m *Manager) ClearContractABI(ctx context.Context, ownerAddress, contractAddress *types.Address) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}
	if contractAddress == nil {
		return nil, fmt.Errorf("%w: invalid contract address: nil", types.ErrInvalidAddress)
	}

	req := &core.ClearABIContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: contractAddress.Bytes(),
	}

	return m.client.ClearContractABI(ctx, req)
}
