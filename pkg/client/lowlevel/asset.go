// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// package lowlevel provides 1:1 wrappers around WalletClient gRPC methods
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Asset related gRPC calls

// CreateAssetIssue2 creates an asset issue (v2 - preferred)
func CreateAssetIssue2(cp ConnProvider, ctx context.Context, req *core.AssetIssueContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "create asset issue2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateAssetIssue2(ctx, req)
	})
}

// UpdateAsset2 updates an asset (v2 - preferred)
func UpdateAsset2(cp ConnProvider, ctx context.Context, req *core.UpdateAssetContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "update asset2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UpdateAsset2(ctx, req)
	})
}

// TransferAsset2 transfers an asset (v2 - preferred)
func TransferAsset2(cp ConnProvider, ctx context.Context, req *core.TransferAssetContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "transfer asset2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.TransferAsset2(ctx, req)
	})
}

// ParticipateAssetIssue2 participates in asset issue (v2 - preferred)
func ParticipateAssetIssue2(cp ConnProvider, ctx context.Context, req *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "participate asset issue2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.ParticipateAssetIssue2(ctx, req)
	})
}

// UnfreezeAsset2 unfreezes an asset (v2 - preferred)
func UnfreezeAsset2(cp ConnProvider, ctx context.Context, req *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "unfreeze asset2", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.UnfreezeAsset2(ctx, req)
	})
}

// GetAssetIssueByAccount gets asset issues by account
func GetAssetIssueByAccount(cp ConnProvider, ctx context.Context, req *core.Account) (*api.AssetIssueList, error) {
	return Call(cp, ctx, "get asset issue by account", func(client api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return client.GetAssetIssueByAccount(ctx, req)
	})
}

// GetAssetIssueByName gets asset issue by name
func GetAssetIssueByName(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.AssetIssueContract, error) {
	return Call(cp, ctx, "get asset issue by name", func(client api.WalletClient, ctx context.Context) (*core.AssetIssueContract, error) {
		return client.GetAssetIssueByName(ctx, req)
	})
}

// GetAssetIssueListByName gets asset issue list by name
func GetAssetIssueListByName(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*api.AssetIssueList, error) {
	return Call(cp, ctx, "get asset issue list by name", func(client api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return client.GetAssetIssueListByName(ctx, req)
	})
}

// GetAssetIssueById gets asset issue by ID
func GetAssetIssueById(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*core.AssetIssueContract, error) {
	return Call(cp, ctx, "get asset issue by id", func(client api.WalletClient, ctx context.Context) (*core.AssetIssueContract, error) {
		return client.GetAssetIssueById(ctx, req)
	})
}

// GetAssetIssueList gets all asset issues
func GetAssetIssueList(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.AssetIssueList, error) {
	return Call(cp, ctx, "get asset issue list", func(client api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return client.GetAssetIssueList(ctx, req)
	})
}

// GetPaginatedAssetIssueList gets paginated asset issue list
func GetPaginatedAssetIssueList(cp ConnProvider, ctx context.Context, req *api.PaginatedMessage) (*api.AssetIssueList, error) {
	return Call(cp, ctx, "get paginated asset issue list", func(client api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return client.GetPaginatedAssetIssueList(ctx, req)
	})
}
