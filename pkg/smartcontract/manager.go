// Package smartcontract provides high-level smart contract functionality
package smartcontract

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
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

// DeployContract deploys a smart contract
func (m *Manager) DeployContract(ctx context.Context, ownerAddress string, contractName string, abi string, bytecode string, feeLimit int64, callValue int64, consumeUserResourcePercent int64, originEnergyLimit int64) (*api.TransactionExtention, error) {
	// Validate inputs
	if contractName == "" {
		return nil, fmt.Errorf("contract name cannot be empty")
	}
	if bytecode == "" {
		return nil, fmt.Errorf("bytecode cannot be empty")
	}
	if feeLimit < 0 {
		return nil, fmt.Errorf("fee limit cannot be negative")
	}
	if callValue < 0 {
		return nil, fmt.Errorf("call value cannot be negative")
	}

	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	// Create new contract
	newContract := &core.SmartContract{
		OriginAddress:              addr.Bytes(),
		ContractAddress:            nil, // Will be generated
		Abi:                        &core.SmartContract_ABI{Entrys: []*core.SmartContract_ABI_Entry{}},
		Bytecode:                   []byte(bytecode),
		CallValue:                  callValue,
		ConsumeUserResourcePercent: consumeUserResourcePercent,
		Name:                       contractName,
		OriginEnergyLimit:          originEnergyLimit,
	}

	// Parse ABI if provided
	if abi != "" {
		if err := utils.ValidateABI(abi); err != nil {
			return nil, fmt.Errorf("invalid ABI: %w", err)
		}
		// Note: ABI parsing would be implemented here in a real scenario
		// For now, we'll leave it as empty entries
	}

	req := &core.CreateSmartContract{
		OwnerAddress:    addr.Bytes(),
		NewContract:     newContract,
		CallTokenValue:  0,
		TokenId:         0,
	}

	return lowlevel.DeployContract(m.client, ctx, req)
}

// TriggerContract triggers a smart contract method
func (m *Manager) TriggerContract(ctx context.Context, ownerAddress string, contractAddress string, data []byte, callValue int64, callTokenValue int64, tokenId int64) (*api.TransactionExtention, error) {
	// Validate inputs
	if len(data) == 0 {
		return nil, fmt.Errorf("contract data cannot be empty")
	}
	if callValue < 0 {
		return nil, fmt.Errorf("call value cannot be negative")
	}
	if callTokenValue < 0 {
		return nil, fmt.Errorf("call token value cannot be negative")
	}

	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	contractAddr, err := utils.ValidateAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	if err := utils.ValidateContractData(data); err != nil {
		return nil, fmt.Errorf("invalid contract data: %w", err)
	}

	req := &core.TriggerSmartContract{
		OwnerAddress:     ownerAddr.Bytes(),
		ContractAddress:  contractAddr.Bytes(),
		Data:             data,
		CallValue:        callValue,
		CallTokenValue:   callTokenValue,
		TokenId:          tokenId,
	}

	return lowlevel.TriggerContract(m.client, ctx, req)
}

// TriggerConstantContract triggers a constant smart contract method (read-only)
func (m *Manager) TriggerConstantContract(ctx context.Context, ownerAddress string, contractAddress string, data []byte, callValue int64) (*api.TransactionExtention, error) {
	// Validate inputs
	if len(data) == 0 {
		return nil, fmt.Errorf("contract data cannot be empty")
	}
	if callValue < 0 {
		return nil, fmt.Errorf("call value cannot be negative")
	}

	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	contractAddr, err := utils.ValidateAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	if err := utils.ValidateContractData(data); err != nil {
		return nil, fmt.Errorf("invalid contract data: %w", err)
	}

	req := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddr.Bytes(),
		ContractAddress: contractAddr.Bytes(),
		Data:            data,
		CallValue:       callValue,
	}

	return lowlevel.TriggerConstantContract(m.client, ctx, req)
}

// EstimateEnergy estimates energy required for smart contract execution
func (m *Manager) EstimateEnergy(ctx context.Context, ownerAddress string, contractAddress string, data []byte, callValue int64) (*api.EstimateEnergyMessage, error) {
	// Validate inputs
	if len(data) == 0 {
		return nil, fmt.Errorf("contract data cannot be empty")
	}
	if callValue < 0 {
		return nil, fmt.Errorf("call value cannot be negative")
	}

	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	contractAddr, err := utils.ValidateAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	if err := utils.ValidateContractData(data); err != nil {
		return nil, fmt.Errorf("invalid contract data: %w", err)
	}

	req := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddr.Bytes(),
		ContractAddress: contractAddr.Bytes(),
		Data:            data,
		CallValue:       callValue,
	}

	return lowlevel.EstimateEnergy(m.client, ctx, req)
}

// GetContract gets smart contract information
func (m *Manager) GetContract(ctx context.Context, contractAddress string) (*core.SmartContract, error) {
	addr, err := utils.ValidateAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	req := &api.BytesMessage{
		Value: addr.Bytes(),
	}

	return lowlevel.GetContract(m.client, ctx, req)
}

// GetContractInfo gets smart contract detailed information
func (m *Manager) GetContractInfo(ctx context.Context, contractAddress string) (*core.SmartContractDataWrapper, error) {
	addr, err := utils.ValidateAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	req := &api.BytesMessage{
		Value: addr.Bytes(),
	}

	return lowlevel.GetContractInfo(m.client, ctx, req)
}

// UpdateSetting updates smart contract settings
func (m *Manager) UpdateSetting(ctx context.Context, ownerAddress string, contractAddress string, consumeUserResourcePercent int64) (*api.TransactionExtention, error) {
	if consumeUserResourcePercent < 0 || consumeUserResourcePercent > 100 {
		return nil, fmt.Errorf("consume user resource percent must be between 0 and 100")
	}

	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	contractAddr, err := utils.ValidateAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	req := &core.UpdateSettingContract{
		OwnerAddress:               ownerAddr.Bytes(),
		ContractAddress:            contractAddr.Bytes(),
		ConsumeUserResourcePercent: consumeUserResourcePercent,
	}

	return lowlevel.UpdateSetting(m.client, ctx, req)
}

// UpdateEnergyLimit updates smart contract energy limit
func (m *Manager) UpdateEnergyLimit(ctx context.Context, ownerAddress string, contractAddress string, originEnergyLimit int64) (*api.TransactionExtention, error) {
	if originEnergyLimit < 0 {
		return nil, fmt.Errorf("origin energy limit cannot be negative")
	}

	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	contractAddr, err := utils.ValidateAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	req := &core.UpdateEnergyLimitContract{
		OwnerAddress:        ownerAddr.Bytes(),
		ContractAddress:     contractAddr.Bytes(),
		OriginEnergyLimit:   originEnergyLimit,
	}

	return lowlevel.UpdateEnergyLimit(m.client, ctx, req)
}

// ClearContractABI clears smart contract ABI
func (m *Manager) ClearContractABI(ctx context.Context, ownerAddress string, contractAddress string) (*api.TransactionExtention, error) {
	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	contractAddr, err := utils.ValidateAddress(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	req := &core.ClearABIContract{
		OwnerAddress:    ownerAddr.Bytes(),
		ContractAddress: contractAddr.Bytes(),
	}

	return lowlevel.ClearContractABI(m.client, ctx, req)
}