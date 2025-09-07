// Package lowlevel provides 1:1 wrappers around WalletClient gRPC methods.
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Smart contract related gRPC calls

// DeployContract deploys a smart contract
func DeployContract(cp ConnProvider, ctx context.Context, req *core.CreateSmartContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "deploy contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.DeployContract(ctx, req)
	})
}

// TriggerContract triggers a smart contract method
func TriggerContract(cp ConnProvider, ctx context.Context, req *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "trigger contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerContract(ctx, req)
	})
}

// TriggerConstantContract triggers a constant smart contract method (read-only)
func TriggerConstantContract(cp ConnProvider, ctx context.Context, req *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return Call(cp, ctx, "trigger constant contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerConstantContract(ctx, req)
	})
}

// EstimateEnergy estimates energy required for smart contract execution
func EstimateEnergy(cp ConnProvider, ctx context.Context, req *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	return Call(cp, ctx, "estimate energy", func(client api.WalletClient, ctx context.Context) (*api.EstimateEnergyMessage, error) {
		return client.EstimateEnergy(ctx, req)
	})
}

// GetContract gets smart contract information
func GetContract(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.SmartContract, error) {
	return Call(cp, ctx, "get contract", func(client api.WalletClient, ctx context.Context) (*core.SmartContract, error) {
		return client.GetContract(ctx, req)
	})
}

// GetContractInfo gets smart contract detailed information
func GetContractInfo(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
	return Call(cp, ctx, "get contract info", func(client api.WalletClient, ctx context.Context) (*core.SmartContractDataWrapper, error) {
		return client.GetContractInfo(ctx, req)
	})
}

// UpdateSetting updates smart contract settings
func UpdateSetting(cp ConnProvider, ctx context.Context, req *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "update setting", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateSetting(ctx, req)
	})
}

// UpdateEnergyLimit updates smart contract energy limit
func UpdateEnergyLimit(cp ConnProvider, ctx context.Context, req *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "update energy limit", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateEnergyLimit(ctx, req)
	})
}

// ClearContractABI clears smart contract ABI
func ClearContractABI(cp ConnProvider, ctx context.Context, req *core.ClearABIContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "clear contract abi", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ClearContractABI(ctx, req)
	})
}
