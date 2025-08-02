// package client provides 1:1 wrappers around WalletClient gRPC methods
package client

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
)

// Asset related gRPC calls

// CreateAssetIssue2 creates an asset issue (v2 - preferred)
func CreateAssetIssue2(c *client.Client, ctx context.Context, req *core.AssetIssueContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "create asset issue2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateAssetIssue2(ctx, req)
	})
}

// UpdateAsset2 updates an asset (v2 - preferred)
func UpdateAsset2(c *client.Client, ctx context.Context, req *core.UpdateAssetContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "update asset2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateAsset2(ctx, req)
	})
}

// TransferAsset2 transfers an asset (v2 - preferred)
func TransferAsset2(c *client.Client, ctx context.Context, req *core.TransferAssetContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "transfer asset2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TransferAsset2(ctx, req)
	})
}

// ParticipateAssetIssue2 participates in asset issue (v2 - preferred)
func ParticipateAssetIssue2(c *client.Client, ctx context.Context, req *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "participate asset issue2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ParticipateAssetIssue2(ctx, req)
	})
}

// UnfreezeAsset2 unfreezes an asset (v2 - preferred)
func UnfreezeAsset2(c *client.Client, ctx context.Context, req *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
	return grpcTransactionCallWrapper(c, ctx, "unfreeze asset2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnfreezeAsset2(ctx, req)
	})
}

// GetAssetIssueByAccount gets asset issues by account
func GetAssetIssueByAccount(c *client.Client, ctx context.Context, req *core.Account) (*api.AssetIssueList, error) {
	return grpcGenericCallWrapper(c, ctx, "get asset issue by account", func(client api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return client.GetAssetIssueByAccount(ctx, req)
	})
}

// GetAssetIssueByName gets asset issue by name
func GetAssetIssueByName(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.AssetIssueContract, error) {
	return grpcGenericCallWrapper(c, ctx, "get asset issue by name", func(client api.WalletClient, ctx context.Context) (*core.AssetIssueContract, error) {
		return client.GetAssetIssueByName(ctx, req)
	})
}

// GetAssetIssueListByName gets asset issue list by name
func GetAssetIssueListByName(c *client.Client, ctx context.Context, req *api.BytesMessage) (*api.AssetIssueList, error) {
	return grpcGenericCallWrapper(c, ctx, "get asset issue list by name", func(client api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return client.GetAssetIssueListByName(ctx, req)
	})
}

// GetAssetIssueById gets asset issue by ID
func GetAssetIssueById(c *client.Client, ctx context.Context, req *api.BytesMessage) (*core.AssetIssueContract, error) {
	return grpcGenericCallWrapper(c, ctx, "get asset issue by id", func(client api.WalletClient, ctx context.Context) (*core.AssetIssueContract, error) {
		return client.GetAssetIssueById(ctx, req)
	})
}

// GetAssetIssueList gets all asset issues
func GetAssetIssueList(c *client.Client, ctx context.Context, req *api.EmptyMessage) (*api.AssetIssueList, error) {
	return grpcGenericCallWrapper(c, ctx, "get asset issue list", func(client api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return client.GetAssetIssueList(ctx, req)
	})
}

// GetPaginatedAssetIssueList gets paginated asset issue list
func GetPaginatedAssetIssueList(c *client.Client, ctx context.Context, req *api.PaginatedMessage) (*api.AssetIssueList, error) {
	return grpcGenericCallWrapper(c, ctx, "get paginated asset issue list", func(client api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return client.GetPaginatedAssetIssueList(ctx, req)
	})
}
