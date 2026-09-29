package rpc

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
)

func TestAssetWrappers(t *testing.T) {
	runWrapperGroup(t, []wrapperCase{
		{"CreateAssetIssue2", "CreateAssetIssue2", "create asset issue2", func(cp ConnProvider, ctx context.Context) error {
			_, err := CreateAssetIssue2(cp, ctx, &core.AssetIssueContract{})
			return err
		}},
		{"UpdateAsset2", "UpdateAsset2", "update asset2", func(cp ConnProvider, ctx context.Context) error {
			_, err := UpdateAsset2(cp, ctx, &core.UpdateAssetContract{})
			return err
		}},
		{"TransferAsset2", "TransferAsset2", "transfer asset2", func(cp ConnProvider, ctx context.Context) error {
			_, err := TransferAsset2(cp, ctx, &core.TransferAssetContract{})
			return err
		}},
		{"ParticipateAssetIssue2", "ParticipateAssetIssue2", "participate asset issue2", func(cp ConnProvider, ctx context.Context) error {
			_, err := ParticipateAssetIssue2(cp, ctx, &core.ParticipateAssetIssueContract{})
			return err
		}},
		{"UnfreezeAsset2", "UnfreezeAsset2", "unfreeze asset2", func(cp ConnProvider, ctx context.Context) error {
			_, err := UnfreezeAsset2(cp, ctx, &core.UnfreezeAssetContract{})
			return err
		}},
		{"GetAssetIssueByAccount", "GetAssetIssueByAccount", "get asset issue by account", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAssetIssueByAccount(cp, ctx, &core.Account{})
			return err
		}},
		{"GetAssetIssueByName", "GetAssetIssueByName", "get asset issue by name", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAssetIssueByName(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetAssetIssueListByName", "GetAssetIssueListByName", "get asset issue list by name", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAssetIssueListByName(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetAssetIssueById", "GetAssetIssueById", "get asset issue by id", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAssetIssueById(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetAssetIssueList", "GetAssetIssueList", "get asset issue list", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetAssetIssueList(cp, ctx, &api.EmptyMessage{})
			return err
		}},
		{"GetPaginatedAssetIssueList", "GetPaginatedAssetIssueList", "get paginated asset issue list", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetPaginatedAssetIssueList(cp, ctx, &api.PaginatedMessage{})
			return err
		}},
	})
}
