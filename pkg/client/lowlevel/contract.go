// Package lowlevel provides 1:1 wrappers around WalletClient gRPC methods
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
)

// Smart contract related gRPC calls

// DeployContract deploys a smart contract
func DeployContract(c *client.Client, ctx context.Context, req *core.CreateSmartContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "deploy contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.DeployContract(ctx, req)
	})
}

// TriggerContract triggers a smart contract method
func TriggerContract(c *client.Client, ctx context.Context, req *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "trigger contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerContract(ctx, req)
	})
}

// TriggerConstantContract triggers a constant smart contract method (read-only)
func TriggerConstantContract(c *client.Client, ctx context.Context, req *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return grpcGenericCallWrapper(c, ctx, "trigger constant contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerConstantContract(ctx, req)
	})
}

// EstimateEnergy estimates energy required for smart contract execution
func EstimateEnergy(c *client.Client, ctx context.Context, req *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "estimate energy", func(client api.WalletClient, ctx context.Context) (*api.EstimateEnergyMessage, error) {
		return client.EstimateEnergy(ctx, req)
	})
}

// GetContract gets smart contract information
func GetContract(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.SmartContract, error) {
	return grpcGenericCallWrapper(c, ctx, "get contract", func(client api.WalletClient, ctx context.Context) (*core.SmartContract, error) {
		return client.GetContract(ctx, req)
	})
}

// GetContractInfo gets smart contract detailed information
func GetContractInfo(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
	return grpcGenericCallWrapper(c, ctx, "get contract info", func(client api.WalletClient, ctx context.Context) (*core.SmartContractDataWrapper, error) {
		return client.GetContractInfo(ctx, req)
	})
}

// UpdateSetting updates smart contract settings
func UpdateSetting(c *client.Client, ctx context.Context, req *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "update setting", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateSetting(ctx, req)
	})
}

// UpdateEnergyLimit updates smart contract energy limit
func UpdateEnergyLimit(c *client.Client, ctx context.Context, req *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "update energy limit", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateEnergyLimit(ctx, req)
	})
}

// ClearContractABI clears smart contract ABI
func ClearContractABI(c *client.Client, ctx context.Context, req *core.ClearABIContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "clear contract abi", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ClearContractABI(ctx, req)
	})
}