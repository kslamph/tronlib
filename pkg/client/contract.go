// package client provides 1:1 wrappers around WalletClient gRPC methods
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Smart contract related gRPC calls

// DeployContract deploys a smart contract
func (c *Client) DeployContract(ctx context.Context, req *core.CreateSmartContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "deploy contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.DeployContract(ctx, req)
	})
}

// TriggerContract triggers a smart contract method
func (c *Client) TriggerContract(ctx context.Context, req *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "trigger contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerContract(ctx, req)
	})
}

// TriggerConstantContract triggers a constant smart contract method (read-only)
func (c *Client) TriggerConstantContract(ctx context.Context, req *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return grpcGenericCallWrapper(c, ctx, "trigger constant contract", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TriggerConstantContract(ctx, req)
	})
}

// EstimateEnergy estimates energy required for smart contract execution
func (c *Client) EstimateEnergy(ctx context.Context, req *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	return grpcGenericCallWrapper(c, ctx, "estimate energy", func(client api.WalletClient, ctx context.Context) (*api.EstimateEnergyMessage, error) {
		return client.EstimateEnergy(ctx, req)
	})
}

// GetContract gets smart contract information
func (c *Client) GetContract(ctx context.Context, req *api.BytesMessage) (*core.SmartContract, error) {
	return grpcGenericCallWrapper(c, ctx, "get contract", func(client api.WalletClient, ctx context.Context) (*core.SmartContract, error) {
		return client.GetContract(ctx, req)
	})
}

// GetContractInfo gets smart contract detailed information
func (c *Client) GetContractInfo(ctx context.Context, req *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
	return grpcGenericCallWrapper(c, ctx, "get contract info", func(client api.WalletClient, ctx context.Context) (*core.SmartContractDataWrapper, error) {
		return client.GetContractInfo(ctx, req)
	})
}

// UpdateSetting updates smart contract settings
func (c *Client) UpdateSetting(ctx context.Context, req *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "update setting", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateSetting(ctx, req)
	})
}

// UpdateEnergyLimit updates smart contract energy limit
func (c *Client) UpdateEnergyLimit(ctx context.Context, req *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "update energy limit", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateEnergyLimit(ctx, req)
	})
}

// ClearContractABI clears smart contract ABI
func (c *Client) ClearContractABI(ctx context.Context, req *core.ClearABIContract) (*api.TransactionExtention, error) {
	return c.grpcTransactionCallWrapper(ctx, "clear contract abi", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ClearContractABI(ctx, req)
	})
}
